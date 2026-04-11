package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
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
			`{"name":"test-garden","topic_prefix":"test-garden","id":"c5cvhpcbcv45e8bp16dg","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"temperature_humidity_sensor":false,"next_light_action":{"time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(-07:00|Z)","state":"(ON|OFF)"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/action"}\]}`,
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
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2023-08-23T10:00:00Z","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"next_light_action":{"time":"2023-08-23T13:00:01-07:00","state":"OFF"},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusCreated,
		},
		{
			"SuccessfulWithControllerConfig",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "controller_config":{"temperature_humidity_pin":1,"light_pin":2,"valve_pins":[3,4,5],"pump_pins":[6,7,8]}}`,
			false,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2023-08-23T10:00:00Z","controller_config":{"valve_pins":\[3,4,5\],"pump_pins":\[6,7,8\],"light_pin":2,"temperature_humidity_pin":1},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusCreated,
		},
		{
			"SuccessfulWithTemperatureAndHumidity",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "temperature_humidity_sensor": true}`,
			false,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2023-08-23T10:00:00Z","temperature_humidity_sensor":true,"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"temperature_humidity_data":{"temperature_celsius":50,"humidity_percentage":50},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusCreated,
		},
		{
			"SuccessfulButErrorGettingTemperatureAndHumidity",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "temperature_humidity_sensor": true}`,
			true,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2023-08-23T10:00:00Z","temperature_humidity_sensor":true,"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "sqlite",
				Options: map[string]any{
					"data_source_name": ":memory:",
				},
			})
			assert.NoError(t, err)

			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(clock.Now(), nil)
			influxdbClient.On("Close")
			if tt.temperatureHumidityError {
				influxdbClient.On("GetTemperatureAndHumidity", mock.Anything, "test-garden").Return(0.0, 0.0, errors.New("influxdb error"))
			} else {
				influxdbClient.On("GetTemperatureAndHumidity", mock.Anything, "test-garden").Return(50.0, 50.0, nil)
			}

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodPost, "/gardens", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-TZ-Offset", "420")
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
		Driver: "sqlite",
		Options: map[string]any{
			"data_source_name": ":memory:",
		},
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
		r.Header.Set("X-TZ-Offset", "420")
		w := babytest.TestRequest(t, gr.API, r)

		assert.Equal(t, http.StatusCreated, w.Code)
		err = json.Unmarshal(w.Body.Bytes(), &g)
		assert.NoError(t, err)
	})

	t.Run("GetZonesForGarden", func(t *testing.T) {
		zones, err := gr.storageClient.Zones.Search(context.Background(), g.GetID(), nil)
		assert.NoError(t, err)

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
			assert.Equal(t, fmt.Sprintf("Zone %d", i+1), zone.Name)
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
			`{"id":"c5cvhpcbcv45e8bp16dg","name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "temperature_humidity_sensor": true}`,
			false,
			``,
			http.StatusOK,
		},
		{
			"SuccessfulButErrorGettingTemperatureAndHumidity",
			`{"id":"c5cvhpcbcv45e8bp16dg","name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "temperature_humidity_sensor": true}`,
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
				Driver: "sqlite",
				Options: map[string]any{
					"data_source_name": ":memory:",
				},
			})
			assert.NoError(t, err)

			garden := createExampleGarden()
			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(clock.Now(), nil)
			if tt.temperatureHumidityError {
				influxdbClient.On("GetTemperatureAndHumidity", mock.Anything, "test-garden").Return(0.0, 0.0, errors.New("influxdb error"))
			} else {
				influxdbClient.On("GetTemperatureAndHumidity", mock.Anything, "test-garden").Return(50.0, 50.0, nil)
			}

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodPut, "/gardens/"+garden.ID.String(), strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-TZ-Offset", "420")
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
			`{"items":\[{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"temperature_humidity_sensor":false,"next_light_action":{"time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(-07:00|Z)","state":"(ON|OFF)"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfulEndDatedTrue",
			"/gardens?end_dated=true",
			`{"items":\[{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"temperature_humidity_sensor":false,"next_light_action":{"time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(-07:00|Z)","state":"(ON|OFF)"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}\]}`,
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "sqlite",
				Options: map[string]any{
					"data_source_name": ":memory:",
				},
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
				Driver: "sqlite",
				Options: map[string]any{
					"data_source_name": ":memory:",
				},
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
			`{"name": "new name", "created_at": "2021-08-03T19:53:14.816332-07:00", "light_schedule":{"duration":"2m0s","start_time":"22:00:02-07:00"}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2021-08-03T19:53:14.816332-07:00","light_schedule":{"duration":"2m0s","start_time":"22:00:02-07:00"},"temperature_humidity_sensor":false,"next_light_action":{"time":"2023-08-23T22:00:02-07:00","state":"ON"},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
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
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"temperature_humidity_sensor":false,"notification_client_id":"c5cvhpcbcv45e8bp16dg","next_light_action":{"time":"2023-08-23T13:00:01-07:00","state":"OFF"},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyRemoveLightSchedule",
			createExampleGarden(),
			nil,
			`{"name": "new name","light_schedule": {}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(\.\d+)?(-07:00|Z)","temperature_humidity_sensor":false,"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyAddLightSchedule",
			gardenWithoutLight,
			nil,
			`{"name": "new name", "created_at": "2021-08-03T19:53:14.816332-07:00", "light_schedule":{"duration":"2m0s","start_time":"22:00:02-07:00"}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2021-08-03T19:53:14.816332-07:00","light_schedule":{"duration":"2m0s","start_time":"22:00:02-07:00"},"temperature_humidity_sensor":false,"next_light_action":{"time":"2023-08-23T22:00:02-07:00","state":"ON"},"health":{"status":"UP","details":"last contact from Garden was 0s ago","last_contact":"2023-08-23T10:00:00Z"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
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
			r.Header.Set("X-TZ-Offset", "420")
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
				mqttClient.On("Publish", "test-garden/command/light", mock.Anything).Return(nil)
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
				Driver: "sqlite",
				Options: map[string]any{
					"data_source_name": ":memory:",
				},
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
				mqttClient.On("Publish", "test-garden/command/light", []byte(`{"state":"ON"}`)).Return(nil)
			},
			`light.state=on`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulLightActionWithQuote",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", "test-garden/command/light", []byte(`{"state":"ON"}`)).Return(nil)
			},
			`light.state="on"`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulLightActionOFF",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", "test-garden/command/light", []byte(`{"state":"OFF"}`)).Return(nil)
			},
			`light.state=off`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulLightActionOFFWithQuote",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", "test-garden/command/light", []byte(`{"state":"OFF"}`)).Return(nil)
			},
			`light.state="off"`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulStopAllWatering",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", "test-garden/command/stop_all", mock.Anything).Return(nil)
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
				Driver: "sqlite",
				Options: map[string]any{
					"data_source_name": ":memory:",
				},
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
			"invalid light_schedule.duration >= 24 hours: 25h0m0s",
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
				TemperatureHumidityPin:      pointer[uint](0),
				LightPin:                    pointer[uint](0),
				TemperatureHumidityInterval: &pkg.Duration{},
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
			"invalid light_schedule.duration >= 24 hours: 25h0m0s",
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
				for _, zone := range zones {
					influxdbClient.On("GetWaterHistory", mock.Anything, zone.GetID(), topicPrefix, 72*time.Hour, uint64(5)).
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
				// First zone is actively watering (started 30 seconds ago, duration 60 seconds)
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[0].GetID(), topicPrefix, 72*time.Hour, uint64(5)).
					Return([]pkg.WaterHistory{
						{
							Status:    pkg.WaterStatusStarted,
							StartedAt: now.Add(-30 * time.Second),
							Duration:  pkg.Duration{Duration: 60 * time.Second},
						},
					}, nil)
				// Second zone has no activity
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[1].GetID(), topicPrefix, 72*time.Hour, uint64(5)).
					Return([]pkg.WaterHistory{}, nil)
			},
			expectedActive:   true,
			expectedQueue:    0,
			expectedZoneName: "Zone 1",
		},
		{
			name: "MultipleZonesQueued",
			setupInfluxDB: func(influxdbClient *influxdb.MockClient, topicPrefix string, zones []*pkg.Zone) {
				// First zone has 2 queued items
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[0].GetID(), topicPrefix, 72*time.Hour, uint64(5)).
					Return([]pkg.WaterHistory{
						{Status: pkg.WaterStatusSent, SentAt: now.Add(-5 * time.Minute)},
						{Status: pkg.WaterStatusSent, SentAt: now.Add(-4 * time.Minute)},
					}, nil)
				// Second zone has 1 queued item
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[1].GetID(), topicPrefix, 72*time.Hour, uint64(5)).
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
				// First zone is actively watering with 2 queued items (events are processed in order)
				// When Started is found first, queue=0 from that zone
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[0].GetID(), topicPrefix, 72*time.Hour, uint64(5)).
					Return([]pkg.WaterHistory{
						{
							Status:    pkg.WaterStatusStarted,
							StartedAt: now.Add(-10 * time.Second),
							Duration:  pkg.Duration{Duration: 30 * time.Second},
						},
					}, nil)
				// Second zone has 1 queued item
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[1].GetID(), topicPrefix, 72*time.Hour, uint64(5)).
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
				for _, zone := range zones {
					influxdbClient.On("GetWaterHistory", mock.Anything, zone.GetID(), topicPrefix, 72*time.Hour, uint64(5)).
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
				// First zone is actively watering
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[0].GetID(), topicPrefix, 72*time.Hour, uint64(5)).
					Return([]pkg.WaterHistory{
						{
							Status:    pkg.WaterStatusStarted,
							StartedAt: now.Add(-15 * time.Second),
							Duration:  pkg.Duration{Duration: 60 * time.Second},
						},
					}, nil)
				// Second zone is queued
				influxdbClient.On("GetWaterHistory", mock.Anything, zones[1].GetID(), topicPrefix, 72*time.Hour, uint64(5)).
					Return([]pkg.WaterHistory{
						{Status: pkg.WaterStatusSent, SentAt: now.Add(-1 * time.Minute)},
						{Status: pkg.WaterStatusSent, SentAt: now.Add(-2 * time.Minute)},
					}, nil)
			},
			expectedActive:   true,
			expectedQueue:    2,
			expectedZoneName: "Zone 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "sqlite",
				Options: map[string]any{
					"data_source_name": ":memory:",
				},
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

			// Create GardenResponse and call getActiveWatering
			resp := gr.NewGardenResponse(garden)
			ctx := context.Background()
			resp.getActiveWatering(ctx)

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
		Driver: "sqlite",
		Options: map[string]any{
			"data_source_name": ":memory:",
		},
	})
	assert.NoError(t, err)

	// Create garden with no zones
	garden := createExampleGarden()
	err = storageClient.Gardens.Set(context.Background(), garden)
	assert.NoError(t, err)

	influxdbClient := new(influxdb.MockClient)
	// No zones, so GetWaterHistory should not be called

	gr := NewGardenAPI()
	err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
	assert.NoError(t, err)

	// Create GardenResponse and call getActiveWatering
	resp := gr.NewGardenResponse(garden)
	ctx := context.Background()
	resp.getActiveWatering(ctx)

	// Assertions
	assert.Equal(t, uint(0), resp.WateringQueue, "WateringQueue should be 0")
	assert.Nil(t, resp.ActiveWatering, "ActiveWatering should be nil")
}
