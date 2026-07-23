package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/worker"

	"github.com/calvinmclean/babyapi"
	babytest "github.com/calvinmclean/babyapi/test"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createExampleGarden() *pkg.Garden {
	two := uint(2)
	createdAt, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	id, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	startTime, _ := pkg.StartTimeFromString("22:00:01-07:00")
	return &pkg.Garden{
		Name:        "test-garden",
		TopicPrefix: "test-garden",
		MaxZones:    &two,
		ID:          babyapi.ID{ID: id},
		CreatedAt:   &createdAt,
		LightSchedule: &pkg.LightSchedule{
			Duration:  &pkg.Duration{Duration: 15 * time.Hour},
			StartTime: startTime,
		},
	}
}

func TestGetGarden(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
		code     int
	}{
		{
			"Successful",
			"/gardens/c5cvhpcbcv45e8bp16dg",
			`{"name":"test-garden","topic_prefix":"test-garden","id":"c5cvhpcbcv45e8bp16dg","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","light_schedule":{"duration":"15h","start_time":"22:00:01-07:00"},"next_light_action":{"time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(-07:00|Z)","state":"(ON|OFF)"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/action"},\{"rel":"water_history","href":"/gardens/c5cvhpcbcv45e8bp16dg/water_history"},{"rel":"controller_logs","href":"/gardens/c5cvhpcbcv45e8bp16dg/controller-logs"}\]}`,
			http.StatusOK,
		},
		{
			"StatusNotFound",
			"/gardens/chkodpg3lcj13q82mq40",
			`{"status":"Resource not found."}`,
			http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(clock.Now(), nil)
			influxdbClient.On("Close")
			storageClient := setupZoneAndGardenStorage(t)

			gr := NewGardenAPI()
			err := gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
			assert.NoError(t, err)

			gr.worker.StartAsync()

			r := httptest.NewRequest("GET", tt.path, http.NoBody)
			w := babytest.TestRequest[*pkg.Garden](t, gr.API, r)

			assert.Equal(t, tt.code, w.Code)
			assert.Regexp(t, tt.expected, strings.TrimSpace(w.Body.String()))

			gr.worker.Stop()
		})
	}
}

func TestGetGardenWithFanSchedule(t *testing.T) {
	influxdbClient := new(influxdb.MockClient)
	influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(clock.Now(), nil)
	influxdbClient.On("Close")
	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	assert.NoError(t, err)

	garden := createExampleGarden()
	power := uint(50)
	garden.FanSchedule = &pkg.FanSchedule{
		Duration: &pkg.Duration{Duration: 30 * time.Minute},
		Interval: &pkg.Duration{Duration: 2 * time.Hour},
		Power:    &power,
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	assert.NoError(t, err)

	gr := NewGardenAPI()
	err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
	assert.NoError(t, err)

	gr.worker.StartAsync()

	r := httptest.NewRequest(http.MethodGet, "/gardens/c5cvhpcbcv45e8bp16dg", http.NoBody)
	w := babytest.TestRequest[*pkg.Garden](t, gr.API, r)

	assert.Equal(t, http.StatusOK, w.Code)
	// Verify response contains fan_schedule and next_fan_action with regex for dynamic timestamp
	assert.Regexp(t, `"fan_schedule":\{"duration":"30m","interval":"2h","power":50,"only_with_light":false\}`, w.Body.String())
	assert.Regexp(t, `"next_fan_action":\{"time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","is_active":(true|false)\}`, w.Body.String())

	gr.worker.Stop()
}

func TestCreateGarden(t *testing.T) {
	_ = clock.MockTime()
	t.Cleanup(clock.Reset)

	tests := []struct {
		name                     string
		body                     string
		temperatureHumidityError bool
		expectedRegexp           string
		code                     int
	}{
		{
			"Successful",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "light_schedule": {"duration": "15h", "start_time": "22:00:01-07:00"}}`,
			false,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2023-08-23T10:00:00Z","light_schedule":{"duration":"15h","start_time":"22:00:01-07:00"},"next_light_action":{"time":"2023-08-23T13:00:01-07:00","state":"OFF"},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"},{"rel":"water_history","href":"/gardens/[0-9a-v]{20}/water_history"},{"rel":"controller_logs","href":"/gardens/[0-9a-v]{20}/controller-logs"}\]}`,
			http.StatusCreated,
		},
		{
			"SuccessfulWithControllerConfig",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "controller_config":{"sensors":[{"name":"Ambient","type":"DHT22","pin":21,"interval":"5s"}],"light_pin":2,"valve_pins":[3,4,5],"pump_pins":[6,7,8]}}`,
			false,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2023-08-23T10:00:00Z","controller_config":{"valve_pins":\[3,4,5\],"pump_pins":\[6,7,8\],"light_pin":2,"sensors":\[{"id":"[0-9a-v]{20}","name":"Ambient","type":"DHT22","pin":21,"interval":"5s"}\]},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"sensors_data":\[{"id":"[0-9a-v]{20}","name":"Ambient","type":"DHT22","temperature_celsius":50,"humidity_percentage":50}\],"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"},\{"rel":"water_history","href":"/gardens/[0-9a-v]{20}/water_history"},{"rel":"controller_logs","href":"/gardens/[0-9a-v]{20}/controller-logs"}\]}`,
			http.StatusCreated,
		},
		{
			"SuccessfulWithTemperatureAndHumidity",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "controller_config":{"sensors":[{"name":"Ambient","type":"DHT22","pin":21,"interval":"5s"}]}}`,
			false,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2023-08-23T10:00:00Z","controller_config":{"sensors":\[{"id":"[0-9a-v]{20}","name":"Ambient","type":"DHT22","pin":21,"interval":"5s"}\]},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"sensors_data":\[{"id":"[0-9a-v]{20}","name":"Ambient","type":"DHT22","temperature_celsius":50,"humidity_percentage":50}\],"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"},\{"rel":"water_history","href":"/gardens/[0-9a-v]{20}/water_history"},{"rel":"controller_logs","href":"/gardens/[0-9a-v]{20}/controller-logs"}\]}`,
			http.StatusCreated,
		},
		{
			"SuccessfulButErrorGettingTemperatureAndHumidity",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "controller_config":{"sensors":[{"name":"Ambient","type":"DHT22","pin":21,"interval":"5s"}]}}`,
			true,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2023-08-23T10:00:00Z","controller_config":{"sensors":\[{"id":"[0-9a-v]{20}","name":"Ambient","type":"DHT22","pin":21,"interval":"5s"}\]},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"sensors_data":\[{"id":"","name":"","type":""}\],"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"},\{"rel":"water_history","href":"/gardens/[0-9a-v]{20}/water_history"},{"rel":"controller_logs","href":"/gardens/[0-9a-v]{20}/controller-logs"}\]}`,
			http.StatusCreated,
		},
		{
			"SuccessfulWithFanSchedule",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "fan_schedule": {"duration": "30m", "interval": "2h", "power": 50}}`,
			false,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2023-08-23T10:00:00Z","fan_schedule":{"duration":"30m","interval":"2h","power":50,"only_with_light":false},"next_fan_action":{"time":"2023-08-23T10:30:00Z","is_active":true},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"},{"rel":"water_history","href":"/gardens/[0-9a-v]{20}/water_history"},{"rel":"controller_logs","href":"/gardens/[0-9a-v]{20}/controller-logs"}\]}`,
			http.StatusCreated,
		},
		{
			"ErrorNegativeMaxZones",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones":-2, "light_schedule": {"duration": "15h", "start_time": "22:00:01-07:00"}}`,
			false,
			`{"status":"Invalid request.","error":"json: cannot unmarshal number -2 into Go struct field Garden.max_zones of type uint"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorInvalidRequestBody",
			"{}",
			false,
			`{"status":"Invalid request.","error":"missing required name field"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorInvalidStartTime",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "light_schedule": {"duration": "15h", "start_time": "invalid"}}`,
			false,
			`{"status":"Invalid request.","error":"error parsing start time: parsing time \\"invalid\\" as \\"15:04:05Z07:00\\": cannot parse \\"invalid\\" as \\"15\\""}`,
			http.StatusBadRequest,
		},
		{
			"ErrorBadRequestInvalidStartTime",
			`{"name":"test-garden", "topic_prefix":"test-garden", "max_zones": 1,"light_schedule": {"duration":"1h","start_time":"NOT A TIME"}}`,
			false,
			`{"status":"Invalid request.","error":"error parsing start time: parsing time \\"NOT A TIME\\" as \\"15:04:05Z07:00\\": cannot parse \\"NOT A TIME\\" as \\"15\\""}`,
			http.StatusBadRequest,
		},
		{
			"ErrorCannotSetID",
			`{"id":"c5cvhpcbcv45e8bp16dg", "name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "light_schedule": {"duration": "15h", "start_time": "22:00:01-07:00"}}`,
			false,
			`{"status":"Invalid request.","error":"unable to manually set ID"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorDuplicateTopicPrefix",
			`{"name": "duplicate-garden", "topic_prefix": "test-garden", "max_zones": 2}`,
			false,
			`{"status":"Conflict","error":"topic_prefix \\"test-garden\\" is already in use"}`,
			http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(clock.Now(), nil)
			influxdbClient.On("Close")
			if tt.temperatureHumidityError {
				influxdbClient.On("GetSensorReading", mock.Anything, "test-garden", mock.AnythingOfType("string")).Return(influxdb.SensorReading{}, errors.New("influxdb error"))
			} else {
				influxdbClient.On("GetSensorReading", mock.Anything, "test-garden", mock.AnythingOfType("string")).Return(influxdb.SensorReading{
					Temperature: pointer[float64](50.0),
					Humidity:    pointer[float64](50.0),
				}, nil)
			}

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
			assert.NoError(t, err)

			// For duplicate topic_prefix test, first create a garden with the same topic_prefix
			if tt.name == "ErrorDuplicateTopicPrefix" {
				firstGarden := `{"name": "first-garden", "topic_prefix": "test-garden", "max_zones": 2}`
				r1 := httptest.NewRequest(http.MethodPost, "/gardens", strings.NewReader(firstGarden))
				r1.Header.Set("Content-Type", "application/json")
				w1 := babytest.TestRequest[*pkg.Garden](t, gr.API, r1)
				assert.Equal(t, http.StatusCreated, w1.Code)
			}

			r := httptest.NewRequest(http.MethodPost, "/gardens", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			w := babytest.TestRequest[*pkg.Garden](t, gr.API, r)

			assert.Equal(t, tt.code, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestCreateGarden_AutoCreateZones(t *testing.T) {
	mockClock := clock.MockTime()
	now := mockClock.Now()
	t.Cleanup(clock.Reset)

	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	assert.NoError(t, err)

	influxdbClient := new(influxdb.MockClient)
	influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(clock.Now(), nil)
	influxdbClient.On("Close")

	gr := NewGardenAPI()
	err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
	assert.NoError(t, err)

	var g pkg.Garden
	t.Run("CreateGarden", func(t *testing.T) {
		body := `{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 4, "light_schedule": {"duration": "15h", "start_time": "22:00:01-07:00"}}`
		r := httptest.NewRequest(http.MethodPost, "/gardens?create_zones=true", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := babytest.TestRequest(t, gr.API, r)

		assert.Equal(t, http.StatusCreated, w.Code)
		err = json.Unmarshal(w.Body.Bytes(), &g)
		assert.NoError(t, err)
	})

	t.Run("GetZonesForGarden", func(t *testing.T) {
		zones := make([]*pkg.Zone, 0)
		for zone, err := range gr.storageClient.Zones.Search(context.Background(), g.GetID(), nil) {
			assert.NoError(t, err)
			zones = append(zones, zone)
		}

		assert.Len(t, zones, 4)

		zoneNames := make([]string, 4)
		slices.SortFunc(zones, func(a, b *pkg.Zone) int {
			return strings.Compare(a.Name, b.Name)
		})
		for i, zone := range zones {
			zoneNames[i] = zone.Name
			assert.False(t, zone.EndDated())
			assert.Equal(t, now, *zone.CreatedAt)
			assert.EqualValues(t, i, *zone.Position)
		}

		assert.ElementsMatch(t, []string{
			"Zone 1",
			"Zone 2",
			"Zone 3",
			"Zone 4",
		}, zoneNames)
	})
}

func TestUpdateGardenPUT(t *testing.T) {
	tests := []struct {
		name                     string
		body                     string
		temperatureHumidityError bool
		expectedRegexp           string
		code                     int
	}{
		{
			"Successful",
			`{"id":"c5cvhpcbcv45e8bp16dg","name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "light_schedule": {"duration": "15h", "start_time": "22:00:01-07:00"}}`,
			false,
			``,
			http.StatusOK,
		},
		{
			"SuccessfulWithTemperatureAndHumidity",
			`{"id":"c5cvhpcbcv45e8bp16dg","name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "controller_config":{"sensors":[{"name":"Ambient","type":"DHT22","pin":21,"interval":"5s"}]}}`,
			false,
			``,
			http.StatusOK,
		},
		{
			"SuccessfulButErrorGettingTemperatureAndHumidity",
			`{"id":"c5cvhpcbcv45e8bp16dg","name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "controller_config":{"sensors":[{"name":"Ambient","type":"DHT22","pin":21,"interval":"5s"}]}}`,
			true,
			``,
			http.StatusOK,
		},
		{
			"ErrorNegativeMaxZones",
			`{"id":"c5cvhpcbcv45e8bp16dg","name": "test-garden", "topic_prefix": "test-garden", "max_zones":-2, "light_schedule": {"duration": "15h", "start_time": "22:00:01-07:00"}}`,
			false,
			`{"status":"Invalid request.","error":"json: cannot unmarshal number -2 into Go struct field Garden.max_zones of type uint"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorInvalidRequestBody",
			`{}`,
			false,
			`{"status":"Invalid request.","error":"missing required id field"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorWrongID",
			`{"id":"chkodpg3lcj13q82mq40","name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "light_schedule": {"duration": "15h", "start_time": "22:00:01-07:00"}}`,
			false,
			`{"status":"Invalid request.","error":"id must match URL path"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorInvalidRequestBody",
			`{"id":"c5cvhpcbcv45e8bp16dg"}`,
			false,
			`{"status":"Invalid request.","error":"missing required name field"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorBadRequestInvalidStartTime",
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-garden", "topic_prefix":"test-garden", "max_zones": 1,"light_schedule": {"duration":"1h","start_time":"NOT A TIME"}}`,
			false,
			`{"status":"Invalid request.","error":"error parsing start time: parsing time \\"NOT A TIME\\" as \\"15:04:05Z07:00\\": cannot parse \\"NOT A TIME\\" as \\"15\\""}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			garden := createExampleGarden()
			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(clock.Now(), nil)
			if tt.temperatureHumidityError {
				influxdbClient.On("GetSensorReading", mock.Anything, "test-garden", mock.AnythingOfType("string")).Return(influxdb.SensorReading{}, errors.New("influxdb error"))
			} else {
				influxdbClient.On("GetSensorReading", mock.Anything, "test-garden", mock.AnythingOfType("string")).Return(influxdb.SensorReading{
					Temperature: pointer[float64](50.0),
					Humidity:    pointer[float64](50.0),
				}, nil)
			}

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodPut, "/gardens/"+garden.ID.String(), strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			w := babytest.TestRequest[*pkg.Garden](t, gr.API, r)

			assert.Equal(t, tt.code, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestGetAllGardens(t *testing.T) {
	gardens := []*pkg.Garden{createExampleGarden()}

	tests := []struct {
		name           string
		targetURL      string
		expectedRegexp string
		status         int
	}{
		{
			"SuccessfulEndDatedFalse",
			"/gardens",
			`{"items":\[{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","light_schedule":{"duration":"15h","start_time":"22:00:01-07:00"},"next_light_action":{"time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(-07:00|Z)","state":"(ON|OFF)"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"},\{"rel":"water_history","href":"/gardens/[0-9a-v]{20}/water_history"},{"rel":"controller_logs","href":"/gardens/[0-9a-v]{20}/controller-logs"}\]}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfulEndDatedTrue",
			"/gardens?end_dated=true",
			`{"items":\[{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","light_schedule":{"duration":"15h","start_time":"22:00:01-07:00"},"next_light_action":{"time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(-07:00|Z)","state":"(ON|OFF)"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"},\{"rel":"water_history","href":"/gardens/[0-9a-v]{20}/water_history"},{"rel":"controller_logs","href":"/gardens/[0-9a-v]{20}/controller-logs"}\]}\]}`,
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			for _, g := range gardens {
				err = storageClient.Gardens.Set(context.Background(), g)
				assert.NoError(t, err)
			}

			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(clock.Now(), nil)

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
			assert.NoError(t, err)

			r := httptest.NewRequest("GET", tt.targetURL, http.NoBody)
			w := babytest.TestRequest[*pkg.Garden](t, gr.API, r)

			assert.Equal(t, http.StatusOK, w.Code)
			actual := strings.TrimSpace(w.Body.String())
			assert.Regexp(t, tt.expectedRegexp, actual)
		})
	}
}

func TestEndDateGarden(t *testing.T) {
	now := clock.Now()
	endDatedGarden := createExampleGarden()
	endDatedGarden.EndDate = &now

	gardenWithZone := createExampleGarden()
	zone := createExampleZone()

	tests := []struct {
		name           string
		garden         *pkg.Garden
		zone           *pkg.Zone
		expectedRegexp string
		status         int
	}{
		{
			"Successful",
			createExampleGarden(),
			nil,
			``,
			http.StatusOK,
		},
		{
			"SuccessfullyDeleteGarden",
			endDatedGarden,
			nil,
			``,
			http.StatusOK,
		},
		{
			"ErrorEndDatingGardenWithZones",
			gardenWithZone,
			zone,
			`{"status":"Invalid request.","error":"unable to end-date Garden with active Zones"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			err = storageClient.Gardens.Set(context.Background(), tt.garden)
			assert.NoError(t, err)

			if tt.zone != nil {
				err = storageClient.Zones.Set(context.Background(), tt.zone)
				assert.NoError(t, err)
			}

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, nil, worker.NewWorker(storageClient, nil, nil, slog.Default()))
			assert.NoError(t, err)

			r := httptest.NewRequest("DELETE", fmt.Sprintf("/gardens/%s", tt.garden.ID), http.NoBody)
			w := babytest.TestRequest[*pkg.Garden](t, gr.API, r)

			assert.Equal(t, tt.status, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestUpdateGarden(t *testing.T) {
	_ = clock.MockTime()
	t.Cleanup(clock.Reset)

	gardenWithoutLight := createExampleGarden()
	gardenWithoutLight.LightSchedule = nil

	gardenWithZone := createExampleGarden()
	zone1 := createExampleZone()
	zone2 := createExampleZone()
	zone2.ID = babyapi.NewID()

	notificationClient := &notifications.Client{
		ID:   babyapi.ID{ID: id},
		Name: "TestClient",
		URL:  "fake://",
	}

	tests := []struct {
		name           string
		garden         *pkg.Garden
		zones          []*pkg.Zone
		body           string
		expectedRegexp string
		status         int
	}{
		{
			"Successful",
			createExampleGarden(),
			nil,
			`{"name": "new name", "created_at": "2021-08-03T19:53:14.816332-07:00", "light_schedule":{"duration":"2m","start_time":"22:00:02-07:00"}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2021-08-03T19:53:14.816332-07:00","light_schedule":{"duration":"2m","start_time":"22:00:02-07:00"},"next_light_action":{"time":"2023-08-23T22:00:02-07:00","state":"ON"},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"},\{"rel":"water_history","href":"/gardens/[0-9a-v]{20}/water_history"},{"rel":"controller_logs","href":"/gardens/[0-9a-v]{20}/controller-logs"}\]}`,
			http.StatusOK,
		},
		{
			"AddNotificationClientIDErrorNotFound",
			createExampleGarden(),
			nil,
			`{"notification_client_id":"NOTIFICATION_CLIENT_ID"}`,
			`{"status":"Invalid request.","error":"error getting NotificationClient with ID \\"NOTIFICATION_CLIENT_ID\\": resource not found"}`,
			http.StatusBadRequest,
		},
		{
			"AddNotificationClientIDSuccess",
			createExampleGarden(),
			nil,
			`{"notification_client_id":"c5cvhpcbcv45e8bp16dg"}`,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","light_schedule":{"duration":"15h","start_time":"22:00:01-07:00"},"notification_client_id":"c5cvhpcbcv45e8bp16dg","next_light_action":{"time":"2023-08-23T13:00:01-07:00","state":"OFF"},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"},\{"rel":"water_history","href":"/gardens/[0-9a-v]{20}/water_history"},{"rel":"controller_logs","href":"/gardens/[0-9a-v]{20}/controller-logs"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyRemoveLightSchedule",
			createExampleGarden(),
			nil,
			`{"name": "new name","light_schedule": {}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"},\{"rel":"water_history","href":"/gardens/[0-9a-v]{20}/water_history"},{"rel":"controller_logs","href":"/gardens/[0-9a-v]{20}/controller-logs"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyAddLightSchedule",
			gardenWithoutLight,
			nil,
			`{"name": "new name", "created_at": "2021-08-03T19:53:14.816332-07:00", "light_schedule":{"duration":"2m","start_time":"22:00:02-07:00"}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2021-08-03T19:53:14.816332-07:00","light_schedule":{"duration":"2m","start_time":"22:00:02-07:00"},"next_light_action":{"time":"2023-08-23T22:00:02-07:00","state":"ON"},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"},\{"rel":"water_history","href":"/gardens/[0-9a-v]{20}/water_history"},{"rel":"controller_logs","href":"/gardens/[0-9a-v]{20}/controller-logs"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyAddFanSchedule",
			createExampleGarden(),
			nil,
			`{"fan_schedule": {"duration": "30m", "interval": "2h", "power": 50}}`,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"c5cvhpcbcv45e8bp16dg","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","light_schedule":{"duration":"15h","start_time":"22:00:01-07:00"},"fan_schedule":{"duration":"30m","interval":"2h","power":50,"only_with_light":false},"next_light_action":{"time":"2023-08-23T13:00:01-07:00","state":"OFF"},"next_fan_action":{"time":"2023-08-23T10:30:00Z","is_active":true},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/action"},{"rel":"water_history","href":"/gardens/c5cvhpcbcv45e8bp16dg/water_history"},{"rel":"controller_logs","href":"/gardens/c5cvhpcbcv45e8bp16dg/controller-logs"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyRemoveFanSchedule",
			func() *pkg.Garden {
				g := createExampleGarden()
				power := uint(50)
				g.FanSchedule = &pkg.FanSchedule{
					Duration: &pkg.Duration{Duration: 30 * time.Minute},
					Interval: &pkg.Duration{Duration: 2 * time.Hour},
					Power:    &power,
				}
				return g
			}(),
			nil,
			`{"fan_schedule": {}}`,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","light_schedule":{"duration":"15h","start_time":"22:00:01-07:00"},"next_light_action":{"time":"2023-08-23T13:00:01-07:00","state":"OFF"},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"},{"rel":"water_history","href":"/gardens/[0-9a-v]{20}/water_history"},{"rel":"controller_logs","href":"/gardens/[0-9a-v]{20}/controller-logs"}\]}`,
			http.StatusOK,
		},
		{
			"ErrorInvalidRequestBody",
			createExampleGarden(),
			nil,
			"abc",
			`{"status":"Invalid request.","error":"invalid character 'a' looking for beginning of value"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorReducingMaxZones",
			gardenWithZone,
			[]*pkg.Zone{zone1, zone2},
			`{"max_zones": 1}`,
			`{"status":"Invalid request.","error":"unable to set max_zones less than current num_zones=2"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(clock.Now(), nil)
			storageClient := setupZoneAndGardenStorage(t)

			err := storageClient.NotificationClientConfigs.Set(context.Background(), notificationClient)
			assert.NoError(t, err)

			for _, z := range tt.zones {
				err := storageClient.Zones.Set(context.Background(), z)
				assert.NoError(t, err)
			}

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodPatch, "/gardens/"+tt.garden.ID.String(), strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			w := babytest.TestRequest[*pkg.Garden](t, gr.API, r)

			assert.Equal(t, tt.status, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestGardenAction(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mqtt.MockClient)
		body      string
		expected  string
		status    int
	}{
		{
			"BadRequest",
			func(_ *mqtt.MockClient) {},
			"bad request",
			`{"status":"Invalid request.","error":"invalid character 'b' looking for beginning of value"}`,
			http.StatusBadRequest,
		},
		{
			"SuccessfulLightAction",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", mock.Anything, "test-garden/command/light", mock.Anything).Return(nil)
			},
			`{"light":{"state":"on"}}`,
			"{}",
			http.StatusAccepted,
		},
		{
			"ErrorInvalidLightState",
			func(_ *mqtt.MockClient) {},
			`{"light":{"state":"BAD"}}`,
			`{"status":"Invalid request.","error":"cannot unmarshal \"BAD\" into Go value of type *pkg.LightState"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			tt.setupMock(mqttClient)

			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, nil, worker.NewWorker(storageClient, nil, mqttClient, slog.Default()))
			assert.NoError(t, err)

			garden := createExampleGarden()
			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/gardens/%s/action", garden.ID), strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			w := babytest.TestRequest[*pkg.Garden](t, gr.API, r)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))
			mqttClient.AssertExpectations(t)
		})
	}
}

func TestGardenActionForm(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mqtt.MockClient)
		body      string
		expected  string
		status    int
	}{
		{
			"BadRequest",
			func(_ *mqtt.MockClient) {},
			"not_found=x",
			`{"status":"Invalid request.","error":"not_found doesn't exist in action.GardenAction"}`,
			http.StatusBadRequest,
		},
		{
			"SuccessfulLightAction",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", mock.Anything, "test-garden/command/light", []byte(`{"state":"ON"}`)).Return(nil)
			},
			`light.state=on`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulLightActionWithQuote",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", mock.Anything, "test-garden/command/light", []byte(`{"state":"ON"}`)).Return(nil)
			},
			`light.state="on"`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulLightActionOFF",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", mock.Anything, "test-garden/command/light", []byte(`{"state":"OFF"}`)).Return(nil)
			},
			`light.state=off`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulLightActionOFFWithQuote",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", mock.Anything, "test-garden/command/light", []byte(`{"state":"OFF"}`)).Return(nil)
			},
			`light.state="off"`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulStopAllWatering",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", mock.Anything, "test-garden/command/stop_all", mock.Anything).Return(nil)
			},
			`stop.all=true`,
			"{}",
			http.StatusAccepted,
		},
		{
			"ErrorInvalidLightState",
			func(_ *mqtt.MockClient) {},
			`light.state=BAD`,
			`{"status":"Invalid request.","error":"cannot unmarshal BAD into Go value of type *pkg.LightState"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			tt.setupMock(mqttClient)

			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, nil, worker.NewWorker(storageClient, nil, mqttClient, slog.Default()))
			assert.NoError(t, err)

			garden := createExampleGarden()
			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/gardens/%s/action", garden.ID), bytes.NewBufferString(tt.body))
			r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			w := babytest.TestRequest(t, gr.API, r)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))
			mqttClient.AssertExpectations(t)
		})
	}
}

func TestGardenActionFirmwareUpdate(t *testing.T) {
	firmwareData := []byte("fake firmware binary data")

	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/u", r.URL.Path)
		assert.True(t, strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data"))

		r.Body = http.MaxBytesReader(w, r.Body, int64(maxFirmwareUploadSize+1024))
		err := r.ParseMultipartForm(int64(maxFirmwareUploadSize + 1024))
		assert.NoError(t, err)

		file, header, err := r.FormFile("update")
		assert.NoError(t, err)
		defer func() { _ = file.Close() }()

		assert.Equal(t, "firmware.bin", header.Filename)

		received, err := io.ReadAll(file)
		assert.NoError(t, err)
		assert.Equal(t, firmwareData, received)

		w.WriteHeader(http.StatusOK)
	}))
	defer uploadServer.Close()

	assetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/firmware.bin", r.URL.Path)
		_, _ = w.Write(firmwareData)
	}))
	defer assetServer.Close()

	releaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/releases/tags/controller-latest", r.URL.Path)
		_, _ = fmt.Fprintf(w, `{"assets":[{"name":"firmware.bin","browser_download_url":"%s/firmware.bin"}]}`, assetServer.URL)
	}))
	defer releaseServer.Close()

	tests := []struct {
		name         string
		buildRequest func(string) *http.Request
		expected     string
		status       int
	}{
		{
			"SuccessfulDirectUpload",
			func(path string) *http.Request {
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				_ = writer.WriteField("firmware_update.latest", "false")
				part, _ := writer.CreateFormFile("firmware_update.file", "firmware.bin")
				_, _ = part.Write(firmwareData)
				_ = writer.Close()

				r := httptest.NewRequest(http.MethodPost, path, &body)
				r.Header.Set("Content-Type", writer.FormDataContentType())
				return r
			},
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulLatest",
			func(path string) *http.Request {
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				_ = writer.WriteField("firmware_update.latest", "true")
				_ = writer.Close()

				r := httptest.NewRequest(http.MethodPost, path, &body)
				r.Header.Set("Content-Type", writer.FormDataContentType())
				return r
			},
			"{}",
			http.StatusAccepted,
		},
		{
			"InvalidExtension",
			func(path string) *http.Request {
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				_ = writer.WriteField("firmware_update.latest", "false")
				part, _ := writer.CreateFormFile("firmware_update.file", "firmware.txt")
				_, _ = part.Write(firmwareData)
				_ = writer.Close()

				r := httptest.NewRequest(http.MethodPost, path, &body)
				r.Header.Set("Content-Type", writer.FormDataContentType())
				return r
			},
			`{"status":"Invalid request.","error":"firmware file must have .bin extension"}`,
			http.StatusBadRequest,
		},
		{
			"FileTooLarge",
			func(path string) *http.Request {
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				_ = writer.WriteField("firmware_update.latest", "false")
				part, _ := writer.CreateFormFile("firmware_update.file", "firmware.bin")
				_, _ = part.Write(make([]byte, maxFirmwareUploadSize+1))
				_ = writer.Close()

				r := httptest.NewRequest(http.MethodPost, path, &body)
				r.Header.Set("Content-Type", writer.FormDataContentType())
				return r
			},
			`{"status":"Invalid request.","error":"firmware file exceeds maximum size of 3 MB"}`,
			http.StatusBadRequest,
		},
		{
			"MissingFile",
			func(path string) *http.Request {
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				_ = writer.WriteField("firmware_update.latest", "false")
				_ = writer.Close()

				r := httptest.NewRequest(http.MethodPost, path, &body)
				r.Header.Set("Content-Type", writer.FormDataContentType())
				return r
			},
			`{"status":"Invalid request.","error":"firmware_update file is required when latest is not true"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)

			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			w := worker.NewWorker(storageClient, nil, mqttClient, slog.Default(),
				worker.WithFirmwareUpdateUploadURLFunc(func(string) string { return uploadServer.URL + "/u" }),
				worker.WithFirmwareUpdateReleaseURLFunc(func() string { return releaseServer.URL + "/releases/tags/controller-latest" }),
			)

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, nil, w)
			assert.NoError(t, err)

			garden := createExampleGarden()
			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			r := tt.buildRequest(fmt.Sprintf("/gardens/%s/action", garden.ID))
			resp := babytest.TestRequest[*pkg.Garden](t, gr.API, r)

			assert.Equal(t, tt.status, resp.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(resp.Body.String()))
		})
	}
}

func TestGardenWaterHistory(t *testing.T) {
	recordTime, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	tests := []struct {
		name        string
		setupMock   func(*influxdb.MockClient)
		queryParams string
		expected    string
		status      int
	}{
		{
			"BadRequestInvalidLimit",
			func(*influxdb.MockClient) {},
			"?limit=-1",
			`{"status":"Invalid request.","error":"strconv.ParseUint: parsing \"-1\": invalid syntax"}`,
			http.StatusBadRequest,
		},
		{
			"BadRequestInvalidTimeRange",
			func(*influxdb.MockClient) {},
			"?range=notTime",
			`{"status":"Invalid request.","error":"time: invalid duration \"notTime\""}`,
			http.StatusBadRequest,
		},
		{
			"SuccessfulWaterHistoryEmpty",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetGardenWaterHistory", mock.Anything, "test-garden", time.Hour*72, uint64(20), true).Return([]pkg.WaterHistory{}, nil)
			},
			"",
			`{"history":[],"count":0,"average":"0s","total":"0s"}`,
			http.StatusOK,
		},
		{
			"SuccessfulWaterHistory",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetGardenWaterHistory", mock.Anything, "test-garden", time.Hour*72, uint64(20), true).
					Return([]pkg.WaterHistory{
						{
							Duration:    pkg.Duration{Duration: 3 * time.Second},
							Status:      pkg.WaterStatusCompleted,
							Source:      string(action.SourceCommand),
							SentAt:      recordTime,
							StartedAt:   recordTime,
							CompletedAt: recordTime,
							EventID:     "00000000000000000000",
							ZoneID:      "c5cvhpcbcv45e8bp16dg",
						},
					}, nil)
			},
			"",
			`{"history":[{"duration":"3s","event_id":"00000000000000000000","zone_id":"c5cvhpcbcv45e8bp16dg","status":"complete","source":"command","sent_at":"2021-10-03T11:24:52.891386-07:00","started_at":"2021-10-03T11:24:52.891386-07:00","completed_at":"2021-10-03T11:24:52.891386-07:00"}],"count":1,"average":"3s","total":"3s"}`,
			http.StatusOK,
		},
		{
			"InfluxDBClientError",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetGardenWaterHistory", mock.Anything, "test-garden", time.Hour*72, uint64(20), true).
					Return([]pkg.WaterHistory{}, errors.New("influxdb error"))
			},
			"",
			`{"status":"Server Error.","error":"influxdb error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(influxdbClient)

			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
			assert.NoError(t, err)

			garden := createExampleGarden()
			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/gardens/%s/water_history%s", garden.ID, tt.queryParams), http.NoBody)
			w := babytest.TestRequest[*pkg.Garden](t, gr.API, r)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))

			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestControllerLogs(t *testing.T) {
	recordTime, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	tests := []struct {
		name        string
		setupMock   func(*influxdb.MockClient)
		queryParams string
		expected    string
		status      int
	}{
		{
			"BadRequestInvalidLimit",
			func(*influxdb.MockClient) {},
			"?limit=-1",
			`{"status":"Invalid request.","error":"strconv.ParseUint: parsing \"-1\": invalid syntax"}`,
			http.StatusBadRequest,
		},
		{
			"BadRequestInvalidTimeRange",
			func(*influxdb.MockClient) {},
			"?range=notTime",
			`{"status":"Invalid request.","error":"time: invalid duration \"notTime\""}`,
			http.StatusBadRequest,
		},
		{
			"SuccessfulEmpty",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetControllerLogs", mock.Anything, "test-garden", time.Hour*72, uint64(50)).Return([]pkg.ControllerLog{}, nil)
			},
			"",
			`{"logs":[],"count":0}`,
			http.StatusOK,
		},
		{
			"Successful",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetControllerLogs", mock.Anything, "test-garden", time.Hour*72, uint64(50)).
					Return([]pkg.ControllerLog{
						{
							Time:    recordTime,
							Level:   "error",
							Source:  "wifi_manager",
							Message: "error restarting mDNS after reconnect",
							Details: map[string]string{
								"reset_reason": "Reset due to power-on event.",
								"device":       "esp32",
							},
						},
					}, nil)
			},
			"",
			`{"logs":[{"time":"2021-10-03T11:24:52.891386-07:00","level":"error","source":"wifi_manager","message":"error restarting mDNS after reconnect","details":{"device":"esp32","reset_reason":"Reset due to power-on event."}}],"count":1}`,
			http.StatusOK,
		},
		{
			"InfluxDBClientError",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetControllerLogs", mock.Anything, "test-garden", time.Hour*72, uint64(50)).
					Return([]pkg.ControllerLog{}, errors.New("influxdb error"))
			},
			"",
			`{"status":"Server Error.","error":"influxdb error"}`,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(influxdbClient)

			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
			assert.NoError(t, err)

			garden := createExampleGarden()
			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/gardens/%s/controller-logs%s", garden.ID, tt.queryParams), http.NoBody)
			w := babytest.TestRequest[*pkg.Garden](t, gr.API, r)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))

			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestGardenRequest(t *testing.T) {
	startTime, _ := pkg.StartTimeFromString("22:00:01-07:00")
	zero := uint(0)
	one := uint(1)
	tests := []struct {
		name string
		gr   *pkg.Garden
		err  string
	}{
		{
			"EmptyRequestError",
			nil,
			"missing required Garden fields",
		},
		{
			"MissingNameError",
			&pkg.Garden{},
			"missing required name field",
		},
		{
			"MissingTopicPrefixError",
			&pkg.Garden{
				Name: "garden",
			},
			"missing required topic_prefix field",
		},
		{
			"InvalidTopicPrefixError$",
			&pkg.Garden{
				Name:        "garden",
				TopicPrefix: "garden$",
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError#",
			&pkg.Garden{
				Name:        "garden",
				TopicPrefix: "garden#",
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError*",
			&pkg.Garden{
				Name:        "garden",
				TopicPrefix: "garden*",
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError>",
			&pkg.Garden{
				Name:        "garden",
				TopicPrefix: "garden>",
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError+",
			&pkg.Garden{
				Name:        "garden",
				TopicPrefix: "garden+",
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError/",
			&pkg.Garden{
				Name:        "garden",
				TopicPrefix: "garden/",
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"MissingMaxZonesError",
			&pkg.Garden{
				Name:        "garden",
				TopicPrefix: "garden",
			},
			"missing required max_zones field",
		},
		{
			"MaxZonesZeroError",
			&pkg.Garden{
				Name:        "garden",
				TopicPrefix: "garden",
				MaxZones:    &zero,
			},
			"max_zones must not be 0",
		},
		{
			"EmptyLightScheduleDurationError",
			&pkg.Garden{
				Name:        "garden",
				TopicPrefix: "garden",
				MaxZones:    &one,
				LightSchedule: &pkg.LightSchedule{
					StartTime: startTime,
				},
			},
			"missing required light_schedule.duration field",
		},
		{
			"EmptyLightScheduleStartTimeError",
			&pkg.Garden{
				Name:        "garden",
				TopicPrefix: "garden",
				MaxZones:    &one,
				LightSchedule: &pkg.LightSchedule{
					Duration: &pkg.Duration{Duration: time.Minute},
				},
			},
			"missing required light_schedule.start_time field",
		},
		{
			"DurationGreaterThanOrEqualTo24HoursError",
			&pkg.Garden{
				Name:        "garden",
				TopicPrefix: "garden",
				MaxZones:    &one,
				LightSchedule: &pkg.LightSchedule{
					StartTime: startTime,
					Duration:  &pkg.Duration{Duration: 25 * time.Hour},
				},
			},
			"invalid light_schedule.duration >= 24 hours: 1d1h",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		gr := &pkg.Garden{
			TopicPrefix: "garden",
			Name:        "garden",
			MaxZones:    &one,
		}
		r := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
		err := gr.Bind(r)
		assert.NoError(t, err)
	})
	t.Run("SuccessfulRemoveControllerConfigPins", func(t *testing.T) {
		gr := &pkg.Garden{
			TopicPrefix: "garden",
			Name:        "garden",
			MaxZones:    &one,
			ControllerConfig: &pkg.ControllerConfig{
				LightPin: pointer[uint](0),
				Sensors: []pkg.SensorConfig{
					{Name: "Ambient", Type: "DHT22", Pin: 0, Interval: pkg.Duration{}},
				},
			},
		}
		r := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
		err := gr.Bind(r)
		assert.NoError(t, err)
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
			err := tt.gr.Bind(r)
			assert.Equal(t, tt.err, err.Error())
		})
	}
}

func TestUpdateGardenRequest(t *testing.T) {
	now := clock.Now()
	zero := uint(0)
	tests := []struct {
		name string
		gr   *pkg.Garden
		err  string
	}{
		{
			"EmptyRequestError",
			nil,
			"missing required Garden fields",
		},
		{
			"InvalidTopicPrefixError$",
			&pkg.Garden{
				TopicPrefix: "garden$",
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError#",
			&pkg.Garden{
				TopicPrefix: "garden#",
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError*",
			&pkg.Garden{
				TopicPrefix: "garden*",
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError>",
			&pkg.Garden{
				TopicPrefix: "garden>",
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError+",
			&pkg.Garden{
				TopicPrefix: "garden+",
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"InvalidTopicPrefixError/",
			&pkg.Garden{
				TopicPrefix: "garden/",
			},
			"one or more invalid characters in Garden topic_prefix",
		},
		{
			"DurationGreaterThanOrEqualTo24HoursError",
			&pkg.Garden{
				LightSchedule: &pkg.LightSchedule{
					Duration: &pkg.Duration{Duration: 25 * time.Hour},
				},
			},
			"invalid light_schedule.duration >= 24 hours: 1d1h",
		},
		{
			"EndDateError",
			&pkg.Garden{
				EndDate: &now,
			},
			"to end-date a Garden, please use the DELETE endpoint",
		},
		{
			"MaxZonesZeroError",
			&pkg.Garden{
				MaxZones: &zero,
			},
			"max_zones must not be 0",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		gr := &pkg.Garden{
			Name: "garden",
		}
		r := httptest.NewRequest(http.MethodPatch, "/", http.NoBody)
		err := gr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading pkg.Garden JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPatch, "/", http.NoBody)
			err := tt.gr.Bind(r)
			if err == nil {
				t.Error("Expected error reading pkg.Garden JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}

func TestGardenResponseGetActiveWatering(t *testing.T) {
	mockClock := clock.MockTime()
	t.Cleanup(clock.Reset)
	now := mockClock.Now()

	tests := []struct {
		name             string
		setupInfluxDB    func(*influxdb.MockClient, string, []*pkg.Zone)
		expectedActive   bool
		expectedQueue    uint
		expectedZoneName string
	}{
		{
			name: "NoWateringActivity",
			setupInfluxDB: func(influxdbClient *influxdb.MockClient, topicPrefix string, zones []*pkg.Zone) {
				// Mock health check
				influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(now, nil)
				for _, zone := range zones {
					influxdbClient.On("GetWaterHistory", mock.Anything, zone.GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
						Return([]pkg.WaterHistory{}, nil)
				}
			},
			expectedActive:   false,
			expectedQueue:    0,
			expectedZoneName: "",
		},
		{
			name: "ActiveWateringInProgress",
			setupInfluxDB: func(influxdbClient *influxdb.MockClient, topicPrefix string, zones []*pkg.Zone) {
				// Mock health check
				influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(now, nil)
				// First zone is actively watering (started 30 seconds ago, duration 60 seconds)
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[0].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{
						{
							Status:    pkg.WaterStatusStarted,
							StartedAt: now.Add(-30 * time.Second),
							Duration:  pkg.Duration{Duration: 60 * time.Second},
						},
					}, nil)
				// Second zone has no activity
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[1].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{}, nil)
			},
			expectedActive:   true,
			expectedQueue:    0,
			expectedZoneName: "Zone 1",
		},
		{
			name: "MultipleZonesQueued",
			setupInfluxDB: func(influxdbClient *influxdb.MockClient, topicPrefix string, zones []*pkg.Zone) {
				// Mock health check
				influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(now, nil)
				// First zone has 2 queued items
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[0].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{
						{Status: pkg.WaterStatusSent, SentAt: now.Add(-5 * time.Minute)},
						{Status: pkg.WaterStatusSent, SentAt: now.Add(-4 * time.Minute)},
					}, nil)
				// Second zone has 1 queued item
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[1].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{
						{Status: pkg.WaterStatusSent, SentAt: now.Add(-3 * time.Minute)},
					}, nil)
			},
			expectedActive:   false,
			expectedQueue:    3,
			expectedZoneName: "",
		},
		{
			name: "ActiveWateringWithQueue",
			setupInfluxDB: func(influxdbClient *influxdb.MockClient, topicPrefix string, zones []*pkg.Zone) {
				// Mock health check
				influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(now, nil)
				// First zone is actively watering with 2 queued items (events are processed in order)
				// When Started is found first, queue=0 from that zone
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[0].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{
						{
							Status:    pkg.WaterStatusStarted,
							StartedAt: now.Add(-10 * time.Second),
							Duration:  pkg.Duration{Duration: 30 * time.Second},
						},
					}, nil)
				// Second zone has 1 queued item
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[1].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{
						{Status: pkg.WaterStatusSent, SentAt: now.Add(-30 * time.Second)},
					}, nil)
			},
			expectedActive:   true,
			expectedQueue:    1,
			expectedZoneName: "Zone 1",
		},
		{
			name: "InfluxDBError",
			setupInfluxDB: func(influxdbClient *influxdb.MockClient, topicPrefix string, zones []*pkg.Zone) {
				// Mock health check
				influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(now, nil)
				for _, zone := range zones {
					influxdbClient.On("GetWaterHistory", mock.Anything, zone.GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
						Return([]pkg.WaterHistory{}, errors.New("influxdb connection error"))
				}
			},
			expectedActive:   false,
			expectedQueue:    0,
			expectedZoneName: "",
		},
		{
			name: "FirstZoneActiveSecondZoneQueued",
			setupInfluxDB: func(influxdbClient *influxdb.MockClient, topicPrefix string, zones []*pkg.Zone) {
				// Mock health check
				influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(now, nil)
				// First zone is actively watering
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[0].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{
						{
							Status:    pkg.WaterStatusStarted,
							StartedAt: now.Add(-15 * time.Second),
							Duration:  pkg.Duration{Duration: 60 * time.Second},
						},
					}, nil)
				// Second zone is queued
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[1].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{
						{Status: pkg.WaterStatusSent, SentAt: now.Add(-1 * time.Minute)},
						{Status: pkg.WaterStatusSent, SentAt: now.Add(-2 * time.Minute)},
					}, nil)
			},
			expectedActive:   true,
			expectedQueue:    2,
			expectedZoneName: "Zone 1",
		},
		{
			name: "CancelledEventClearsQueue",
			setupInfluxDB: func(influxdbClient *influxdb.MockClient, topicPrefix string, zones []*pkg.Zone) {
				// Mock health check
				influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(now, nil)
				// First zone has a queued item (newest) and a cancelled event before it
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[0].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{
						{Status: pkg.WaterStatusSent, SentAt: now.Add(-1 * time.Minute)},
						{Status: pkg.WaterStatusCancelled, SentAt: now.Add(-5 * time.Minute)},
					}, nil)
				// Second zone has no activity
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[1].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{}, nil)
			},
			expectedActive:   false,
			expectedQueue:    1,
			expectedZoneName: "",
		},
		{
			name: "ActiveWateringAfterCancelledWithQueue",
			setupInfluxDB: func(influxdbClient *influxdb.MockClient, topicPrefix string, zones []*pkg.Zone) {
				// Mock health check
				influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(now, nil)
				// First zone: queued sent (newest), active started, then cancelled (oldest)
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[0].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{
						{Status: pkg.WaterStatusSent, SentAt: now.Add(-5 * time.Second)},
						{
							Status:    pkg.WaterStatusStarted,
							StartedAt: now.Add(-30 * time.Second),
							Duration:  pkg.Duration{Duration: 60 * time.Second},
						},
						{Status: pkg.WaterStatusCancelled, SentAt: now.Add(-10 * time.Minute)},
					}, nil)
				// Second zone has no activity
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[1].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{}, nil)
			},
			expectedActive:   true,
			expectedQueue:    1,
			expectedZoneName: "Zone 1",
		},
		{
			name: "CancelledHidesPreviousStarted",
			setupInfluxDB: func(influxdbClient *influxdb.MockClient, topicPrefix string, zones []*pkg.Zone) {
				// Mock health check
				influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(now, nil)
				// First zone: cancelled (newer) then started (older). CalculateWaterProgress should return empty since cancelled is terminal.
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[0].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{
						{Status: pkg.WaterStatusCancelled, SentAt: now.Add(-1 * time.Minute)},
						{
							Status:    pkg.WaterStatusStarted,
							StartedAt: now.Add(-2 * time.Minute),
							Duration:  pkg.Duration{Duration: 60 * time.Second},
						},
					}, nil)
				// Second zone has no activity
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[1].GetID(), topicPrefix, 72*time.Hour, uint64(5), true).
					Return([]pkg.WaterHistory{}, nil)
			},
			expectedActive:   false,
			expectedQueue:    0,
			expectedZoneName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)

			// Create garden
			garden := createExampleGarden()
			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			// Create zones for the garden
			zones := []*pkg.Zone{
				{
					ID:       babyapi.NewID(),
					GardenID: garden.ID.ID,
					Name:     "Zone 1",
					Position: func(i uint) *uint { return &i }(0),
				},
				{
					ID:       babyapi.NewID(),
					GardenID: garden.ID.ID,
					Name:     "Zone 2",
					Position: func(i uint) *uint { return &i }(1),
				},
			}
			for _, zone := range zones {
				err := storageClient.Zones.Set(context.Background(), zone)
				assert.NoError(t, err)
			}

			// Setup InfluxDB mock
			influxdbClient := new(influxdb.MockClient)
			tt.setupInfluxDB(influxdbClient, garden.TopicPrefix, zones)

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
			assert.NoError(t, err)

			// Create GardenResponse and call fetchInfluxDBData with includeActiveWatering=true
			resp := gr.NewGardenResponse(garden)
			ctx := context.Background()
			resp.fetchInfluxDBData(ctx, slog.Default(), true)

			// Assertions
			assert.Equal(t, tt.expectedQueue, resp.WateringQueue, "WateringQueue mismatch")
			if tt.expectedActive {
				assert.NotNil(t, resp.ActiveWatering, "ActiveWatering should not be nil")
				assert.Equal(t, tt.expectedZoneName, resp.ActiveWatering.ZoneName, "ZoneName mismatch")
			} else {
				assert.Nil(t, resp.ActiveWatering, "ActiveWatering should be nil")
			}

			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestGardenResponseNoZones(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	assert.NoError(t, err)

	// Create garden with no zones
	garden := createExampleGarden()
	err = storageClient.Gardens.Set(context.Background(), garden)
	assert.NoError(t, err)

	influxdbClient := new(influxdb.MockClient)
	// No zones, so GetWaterHistory should not be called
	// Mock health check
	influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(clock.Now(), nil)

	gr := NewGardenAPI()
	err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
	assert.NoError(t, err)

	// Create GardenResponse and call fetchInfluxDBData with includeActiveWatering=true
	resp := gr.NewGardenResponse(garden)
	ctx := context.Background()
	resp.fetchInfluxDBData(ctx, slog.Default(), true)

	// Assertions
	assert.Equal(t, uint(0), resp.WateringQueue, "WateringQueue should be 0")
	assert.Nil(t, resp.ActiveWatering, "ActiveWatering should be nil")

	influxdbClient.AssertExpectations(t)
}
