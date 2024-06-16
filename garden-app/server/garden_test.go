package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
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
	return &pkg.Garden{
		Name:        "test-garden",
		TopicPrefix: "test-garden",
		MaxZones:    &two,
		ID:          babyapi.ID{ID: id},
		CreatedAt:   &createdAt,
		LightSchedule: &pkg.LightSchedule{
			Duration:  &pkg.Duration{Duration: 15 * time.Hour},
			StartTime: "22:00:01-07:00",
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
			`{"name":"test-garden","topic_prefix":"test-garden","id":"c5cvhpcbcv45e8bp16dg","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"next_light_action":{"time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(-07:00|Z)","state":"(ON|OFF)"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/action"}\]}`,
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
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(time.Now(), nil)
			storageClient := setupZoneAndGardenStorage(t)

			gr := NewGardenAPI()
			err := gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, nil, nil, slog.Default()))
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
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"next_light_action":{"time":"0001-01-01T00:00:00Z","state":"OFF"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusCreated,
		},
		{
			"SuccessfulWithTemperatureAndHumidity",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "temperature_humidity_sensor": true}`,
			false,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","temperature_humidity_sensor":true,"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"temperature_humidity_data":{"temperature_celsius":50,"humidity_percentage":50},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusCreated,
		},
		{
			"SuccessfulButErrorGettingTemperatureAndHumidity",
			`{"name": "test-garden", "topic_prefix": "test-garden", "max_zones": 2, "temperature_humidity_sensor": true}`,
			true,
			`{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","temperature_humidity_sensor":true,"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
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
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(time.Now(), nil)
			if tt.temperatureHumidityError {
				influxdbClient.On("GetTemperatureAndHumidity", mock.Anything, "test-garden").Return(0.0, 0.0, errors.New("influxdb error"))
			} else {
				influxdbClient.On("GetTemperatureAndHumidity", mock.Anything, "test-garden").Return(50.0, 50.0, nil)
			}

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, nil, nil, slog.Default()))
			assert.NoError(t, err)

			r := httptest.NewRequest("POST", "/gardens", strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")
			w := babytest.TestRequest[*pkg.Garden](t, gr.API, r)

			assert.Equal(t, tt.code, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))
		})
	}
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
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			garden := createExampleGarden()
			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(time.Now(), nil)
			if tt.temperatureHumidityError {
				influxdbClient.On("GetTemperatureAndHumidity", mock.Anything, "test-garden").Return(0.0, 0.0, errors.New("influxdb error"))
			} else {
				influxdbClient.On("GetTemperatureAndHumidity", mock.Anything, "test-garden").Return(50.0, 50.0, nil)
			}

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, nil, nil, slog.Default()))
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodPut, "/gardens/"+garden.ID.String(), strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")
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
			`{"items":\[{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"next_light_action":{"time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(-07:00|Z)","state":"(ON|OFF)"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfulEndDatedTrue",
			"/gardens?end_dated=true",
			`{"items":\[{"name":"test-garden","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","light_schedule":{"duration":"15h0m0s","start_time":"22:00:01-07:00"},"next_light_action":{"time":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d(-07:00|Z)","state":"(ON|OFF)"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_zones":0,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}\]}`,
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			for _, g := range gardens {
				err = storageClient.Gardens.Set(context.Background(), g)
				assert.NoError(t, err)
			}

			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(time.Now(), nil)

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, nil, nil, slog.Default()))
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
	now := time.Now()
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
				Driver: "hashmap",
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
	gardenWithoutLight := createExampleGarden()
	gardenWithoutLight.LightSchedule = nil

	gardenWithZone := createExampleGarden()
	zone1 := createExampleZone()
	zone2 := createExampleZone()
	zone2.ID = babyapi.NewID()

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
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2021-08-03T19:53:14.816332-07:00","light_schedule":{"duration":"2m0s","start_time":"22:00:02-07:00"},"next_light_action":{"time":"0001-01-01T00:00:00Z","state":"OFF"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyRemoveLightSchedule",
			createExampleGarden(),
			nil,
			`{"name": "new name","light_schedule": {}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/[0-9a-v]{20}/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyAddLightSchedule",
			gardenWithoutLight,
			nil,
			`{"name": "new name", "created_at": "2021-08-03T19:53:14.816332-07:00", "light_schedule":{"duration":"2m0s","start_time":"22:00:02-07:00"}}`,
			`{"name":"new name","topic_prefix":"test-garden","id":"[0-9a-v]{20}","max_zones":2,"created_at":"2021-08-03T19:53:14.816332-07:00","light_schedule":{"duration":"2m0s","start_time":"22:00:02-07:00"},"next_light_action":{"time":"0001-01-01T00:00:00Z","state":"OFF"},"health":{"status":"UP","details":"last contact from Garden was \d+(s|ms) ago","last_contact":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)"},"num_zones":1,"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}"},{"rel":"zones","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/action"}\]}`,
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
			influxdbClient.On("GetLastContact", mock.Anything, "test-garden").Return(time.Now(), nil)
			storageClient := setupZoneAndGardenStorage(t)

			for _, z := range tt.zones {
				err := storageClient.Zones.Set(context.Background(), z)
				assert.NoError(t, err)
			}

			gr := NewGardenAPI()
			err := gr.setup(Config{}, storageClient, influxdbClient, worker.NewWorker(storageClient, nil, nil, slog.Default()))
			assert.NoError(t, err)

			r := httptest.NewRequest("PATCH", "/gardens/"+tt.garden.ID.String(), strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")
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
				mqttClient.On("LightTopic", "test-garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", mock.Anything).Return(nil)
			},
			`{"light":{"state":"on"}}`,
			"{}",
			http.StatusAccepted,
		},
		{
			"ExecuteErrorForLightAction",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("LightTopic", "test-garden").Return("", errors.New("template error"))
			},
			`{"light":{"state":"on"}}`,
			`{"status":"Server Error.","error":"unable to execute LightAction: unable to fill MQTT topic template: template error"}`,
			http.StatusInternalServerError,
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
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			gr := NewGardenAPI()
			err = gr.setup(Config{}, storageClient, nil, worker.NewWorker(storageClient, nil, mqttClient, slog.Default()))
			assert.NoError(t, err)

			garden := createExampleGarden()
			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			r := httptest.NewRequest("POST", fmt.Sprintf("/gardens/%s/action", garden.ID), strings.NewReader(tt.body))
			r.Header.Add("Content-Type", "application/json")
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
				mqttClient.On("LightTopic", "test-garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", []byte(`{"state":"ON","for_duration":null}`)).Return(nil)
			},
			`light.state=on`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulLightActionWithQuote",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("LightTopic", "test-garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", []byte(`{"state":"ON","for_duration":null}`)).Return(nil)
			},
			`light.state="on"`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulLightActionOFF",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("LightTopic", "test-garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", []byte(`{"state":"OFF","for_duration":null}`)).Return(nil)
			},
			`light.state=off`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulLightActionOFFWithQuote",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("LightTopic", "test-garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", []byte(`{"state":"OFF","for_duration":null}`)).Return(nil)
			},
			`light.state="off"`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulStopAllWatering",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("StopAllTopic", "test-garden").Return("garden/action/stop", nil)
				mqttClient.On("Publish", "garden/action/stop", mock.Anything).Return(nil)
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
				Driver: "hashmap",
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
					StartTime: "22:00:01-07:00",
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
					StartTime: "22:00:01-07:00",
					Duration:  &pkg.Duration{Duration: 25 * time.Hour},
				},
			},
			"invalid light_schedule.duration >= 24 hours: 25h0m0s",
		},
		{
			"BadStartTimeError",
			&pkg.Garden{
				Name:        "garden",
				TopicPrefix: "garden",
				MaxZones:    &one,
				LightSchedule: &pkg.LightSchedule{
					Duration:  &pkg.Duration{Duration: time.Minute},
					StartTime: "NOT A TIME",
				},
			},
			"error parsing start time: parsing time \"NOT A TIME\" as \"15:04:05Z07:00\": cannot parse \"NOT A TIME\" as \"15\"",
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
			err := tt.gr.Bind(r)
			assert.Equal(t, tt.err, err.Error())
		})
	}
}

func TestUpdateGardenRequest(t *testing.T) {
	now := time.Now()
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
			"InvalidLightScheduleStartTimeError",
			&pkg.Garden{
				LightSchedule: &pkg.LightSchedule{
					StartTime: "NOT A TIME",
				},
			},
			"error parsing start time: parsing time \"NOT A TIME\" as \"15:04:05Z07:00\": cannot parse \"NOT A TIME\" as \"15\"",
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
