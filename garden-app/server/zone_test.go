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

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	babytest "github.com/calvinmclean/babyapi/test"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	id, _  = xid.FromString("c5cvhpcbcv45e8bp16dg")
	id2, _ = xid.FromString("chkodpg3lcj13q82mq40")
)

func createExampleZone() *pkg.Zone {
	createdAt, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	pos := uint(0)
	return &pkg.Zone{
		Name:             "test-zone",
		Position:         &pos,
		ID:               babyapi.ID{ID: id},
		GardenID:         id,
		CreatedAt:        &createdAt,
		WaterScheduleIDs: []xid.ID{id},
	}
}

func setupWaterScheduleStorage(t *testing.T) *storage.Client {
	t.Helper()

	ws := createExampleWaterSchedule()

	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	err = storageClient.WaterSchedules.Set(context.Background(), ws)
	assert.NoError(t, err)

	return storageClient
}

func setupStorage(t *testing.T, garden *pkg.Garden) *storage.Client {
	t.Helper()

	zone := createExampleZone()
	zone.GardenID = garden.ID.ID

	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	err = storageClient.Gardens.Set(context.Background(), garden)
	assert.NoError(t, err)

	err = storageClient.Zones.Set(context.Background(), zone)
	assert.NoError(t, err)

	return storageClient
}

func setupZoneAndGardenStorage(t *testing.T) *storage.Client {
	t.Helper()

	garden := createExampleGarden()
	zone := createExampleZone()

	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	err = storageClient.Gardens.Set(context.Background(), garden)
	assert.NoError(t, err)

	err = storageClient.Zones.Set(context.Background(), zone)
	assert.NoError(t, err)

	return storageClient
}

func float32Pointer(n float64) *float32 {
	f := float32(n)
	return &f
}

func pointer[T any](v T) *T {
	return &v
}

func TestGetZone(t *testing.T) {
	weatherClientID, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")

	tests := []struct {
		name               string
		excludeWeatherData bool
		waterSchedules     []*pkg.WaterSchedule
		setupMock          func(*influxdb.MockClient)
		expectedRegexp     string
	}{
		{
			"Successful",
			false,
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			func(_ *influxdb.MockClient) {},
			`{"name":"test-zone","id":"c5cvhpcbcv45e8bp16dg","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":null,"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52-07:00","duration":"1s","water_schedule_id":"c5cvhpcbcv45e8bp16dg"},"links":\[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}\]}`,
		},
		{
			"SuccessfulWithRainAndTemperatureData",
			false,
			[]*pkg.WaterSchedule{{
				ID:        babyapi.ID{ID: id},
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Hour},
				StartTime: pkg.NewStartTime(createdAt),
				WeatherControl: &weather.Control{
					Rain: &weather.ScaleControl{
						BaselineValue: float32Pointer(0),
						Factor:        float32Pointer(0),
						Range:         float32Pointer(25.4),
						ClientID:      weatherClientID,
					},
					Temperature: &weather.ScaleControl{
						BaselineValue: float32Pointer(30),
						Factor:        float32Pointer(0.5),
						Range:         float32Pointer(10),
						ClientID:      weatherClientID,
					},
				},
			}},
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("Close")
			},
			`{"name":"test-zone","id":"c5cvhpcbcv45e8bp16dg","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":null,"weather_data":{"rain":{"mm":25.4,"scale_factor":0},"average_temperature":{"celsius":80,"scale_factor":1.5}},"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52-07:00","duration":"0s","water_schedule_id":"c5cvhpcbcv45e8bp16dg"},"links":\[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}\]}`,
		},
		{
			"SuccessfulWithRainAndTemperatureDataButWeatherDataExcluded",
			true,
			[]*pkg.WaterSchedule{{
				ID:        babyapi.ID{ID: id},
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Hour},
				StartTime: pkg.NewStartTime(createdAt),
				WeatherControl: &weather.Control{
					Rain: &weather.ScaleControl{
						BaselineValue: float32Pointer(0),
						Factor:        float32Pointer(0),
						Range:         float32Pointer(25.4),
						ClientID:      weatherClientID,
					},
					Temperature: &weather.ScaleControl{
						BaselineValue: float32Pointer(30),
						Factor:        float32Pointer(0.5),
						Range:         float32Pointer(10),
						ClientID:      weatherClientID,
					},
				},
			}},
			func(_ *influxdb.MockClient) {},
			`{"name":"test-zone","id":"c5cvhpcbcv45e8bp16dg","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":null,"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52-07:00","duration":"1h0m0s","water_schedule_id":"c5cvhpcbcv45e8bp16dg"},"links":\[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}\]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(influxdbClient)
			influxdbClient.On("Close")

			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			for _, ws := range tt.waterSchedules {
				err = storageClient.WaterSchedules.Set(context.Background(), ws)
				assert.NoError(t, err)
			}

			err = storageClient.WeatherClientConfigs.Set(context.Background(), createExampleWeatherClientConfig())
			assert.NoError(t, err)

			zr := NewZonesAPI()
			zr.setup(storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))
			zr.worker.StartAsync()

			for _, ws := range tt.waterSchedules {
				err := zr.worker.ScheduleWaterAction(ws)
				assert.NoError(t, err)
			}

			garden := createExampleGarden()
			zone := createExampleZone()

			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)
			err = storageClient.Zones.Set(context.Background(), zone)
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/gardens/%s/zones/%s?exclude_weather_data=%t", garden.ID, zone.ID, tt.excludeWeatherData), http.NoBody)
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestWithParentRoute(t, zr.API, garden, "Gardens", "/gardens", r)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))

			zr.worker.Stop()
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestZoneAction(t *testing.T) {
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
			"SuccessfulWaterAction",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", "test-garden/command/water", mock.Anything).Return(nil)
			},
			`{"water":{"duration":1000}}`,
			"{}",
			http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			tt.setupMock(mqttClient)
			mqttClient.On("Disconnect", uint(100)).Return()

			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			zr := NewZonesAPI()
			zr.setup(storageClient, nil, worker.NewWorker(storageClient, nil, mqttClient, slog.Default()))

			zr.worker.StartAsync()

			garden := createExampleGarden()
			zone := createExampleZone()

			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)
			err = storageClient.Zones.Set(context.Background(), zone)
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/gardens/%s/zones/%s/action", garden.ID, zone.ID), strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestWithParentRoute(t, zr.API, garden, "Gardens", "/gardens", r)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))

			zr.worker.Stop()
			mqttClient.AssertExpectations(t)
		})
	}
}

func TestZoneActionForm(t *testing.T) {
	worker.CreateNewID = func() xid.ID { return xid.NilID() }
	defer func() { worker.CreateNewID = xid.New }()

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
			`{"status":"Invalid request.","error":"not_found doesn't exist in action.ZoneAction"}`,
			http.StatusBadRequest,
		},
		{
			"SuccessfulWaterActionInteger",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", "test-garden/command/water", []byte(`{"duration":1000,"zone_id":"c5cvhpcbcv45e8bp16dg","position":0,"id":"00000000000000000000","source":"command"}`)).Return(nil)
			},
			`water.duration=1000`,
			"{}",
			http.StatusAccepted,
		},
		{
			"SuccessfulWaterActionString",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("Publish", "test-garden/command/water", []byte(`{"duration":2000,"zone_id":"c5cvhpcbcv45e8bp16dg","position":0,"id":"00000000000000000000","source":"command"}`)).Return(nil)
			},
			`water.duration=2s`,
			"{}",
			http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			tt.setupMock(mqttClient)
			mqttClient.On("Disconnect", uint(100)).Return()

			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			zr := NewZonesAPI()
			zr.setup(storageClient, nil, worker.NewWorker(storageClient, nil, mqttClient, slog.Default()))

			zr.worker.StartAsync()

			garden := createExampleGarden()
			zone := createExampleZone()

			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)
			err = storageClient.Zones.Set(context.Background(), zone)
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/gardens/%s/zones/%s/action", garden.ID, zone.ID), bytes.NewBufferString(tt.body))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestWithParentRoute(t, zr.API, garden, "Gardens", "/gardens", r)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))

			zr.worker.Stop()
			mqttClient.AssertExpectations(t)
		})
	}
}

func TestUpdateZone(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
		status   int
	}{
		{
			"Successful",
			`{"name":"new name"}`,
			`{"name":"new name","id":"c5cvhpcbcv45e8bp16dg","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":["c5cvhpcbcv45e8bp16dg"],"skip_count":null,"next_water":{"message":"no active WaterSchedules"},"links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}]}`,
			http.StatusOK,
		},
		{
			"BadRequest",
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true (expecting 'r')"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorCannotChangeGardenID",
			`{"garden_id": "c5cvhpcbcv45e8bp16dg"}`,
			`{"status":"Invalid request.","error":"unable to change GardenID"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorWaterScheduleNotFound",
			`{"water_schedule_ids":["chkodpg3lcj13q82mq40"]}`,
			`{"status":"Invalid request.","error":"error getting WaterSchedule with ID \"chkodpg3lcj13q82mq40\": resource not found"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := setupZoneAndGardenStorage(t)

			err := storageClient.WaterSchedules.Set(context.Background(), createExampleWaterSchedule())
			assert.NoError(t, err)

			zr := NewZonesAPI()
			zr.setup(storageClient, nil, worker.NewWorker(storageClient, nil, nil, slog.Default()))

			garden := createExampleGarden()
			zone := createExampleZone()

			r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/gardens/%s/zones/%s", garden.ID, zone.ID), strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestWithParentRoute(t, zr.API, garden, "Gardens", "/gardens", r)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestEndDateZone(t *testing.T) {
	now := clock.Now()
	endDatedZone := createExampleZone()
	endDatedZone.EndDate = &now

	tests := []struct {
		name           string
		zone           *pkg.Zone
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			createExampleZone(),
			``,
			http.StatusOK,
		},
		{
			"SuccessfullyDeleteZone",
			endDatedZone,
			``,
			http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := setupZoneAndGardenStorage(t)

			err := storageClient.WaterSchedules.Set(context.Background(), createExampleWaterSchedule())
			assert.NoError(t, err)

			zr := NewZonesAPI()
			zr.setup(storageClient, nil, worker.NewWorker(storageClient, nil, nil, slog.Default()))

			garden := createExampleGarden()
			zone := createExampleZone()

			r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/gardens/%s/zones/%s", garden.ID, zone.ID), http.NoBody)
			w := babytest.TestWithParentRoute(t, zr.API, garden, "Gardens", "/gardens", r)

			assert.Equal(t, tt.code, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestGetAllZones(t *testing.T) {
	storageClient := setupWaterScheduleStorage(t)

	zr := NewZonesAPI()
	zr.setup(storageClient, nil, worker.NewWorker(storageClient, nil, nil, slog.Default()))

	garden := createExampleGarden()
	zone := createExampleZone()
	endDatedZone := createExampleZone()
	endDatedZoneID, _ := xid.FromString("cl85o60cj6rmh16lpmog")
	endDatedZone.ID = babyapi.ID{ID: endDatedZoneID}
	endDate, _ := time.Parse(time.RFC3339Nano, "2023-11-11T22:01:12.733064-07:00")
	endDatedZone.EndDate = &endDate

	err := storageClient.Gardens.Set(context.Background(), garden)
	assert.NoError(t, err)
	err = storageClient.Zones.Set(context.Background(), zone)
	assert.NoError(t, err)
	err = storageClient.Zones.Set(context.Background(), endDatedZone)
	assert.NoError(t, err)

	tests := []struct {
		name      string
		targetURL string
		expected  string
		reverse   string // in the case with 2 zones, sometimes they are in a different order
	}{
		{
			"SuccessfulEndDatedFalse",
			"",
			`{"items":[{"name":"test-zone","id":"c5cvhpcbcv45e8bp16dg","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":["c5cvhpcbcv45e8bp16dg"],"skip_count":null,"next_water":{"message":"no active WaterSchedules"},"links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}]}]}`,
			``,
		},
		{
			"SuccessfulEndDatedTrue",
			"?end_dated=true",
			`{"items":[{"name":"test-zone","id":"c5cvhpcbcv45e8bp16dg","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":["c5cvhpcbcv45e8bp16dg"],"skip_count":null,"next_water":{"message":"no active WaterSchedules"},"links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}]},{"name":"test-zone","id":"cl85o60cj6rmh16lpmog","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","end_date":"2023-11-11T22:01:12.733064-07:00","water_schedule_ids":["c5cvhpcbcv45e8bp16dg"],"skip_count":null,"next_water":{},"links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/cl85o60cj6rmh16lpmog"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"}]}]}`,
			`{"items":[{"name":"test-zone","id":"cl85o60cj6rmh16lpmog","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","end_date":"2023-11-11T22:01:12.733064-07:00","water_schedule_ids":["c5cvhpcbcv45e8bp16dg"],"skip_count":null,"next_water":{},"links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/cl85o60cj6rmh16lpmog"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"}]},{"name":"test-zone","id":"c5cvhpcbcv45e8bp16dg","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":["c5cvhpcbcv45e8bp16dg"],"skip_count":null,"next_water":{"message":"no active WaterSchedules"},"links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}]}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/gardens/%s/zones%s", garden.ID, tt.targetURL), http.NoBody)
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestWithParentRoute(t, zr.API, garden, "Gardens", "/gardens", r)

			assert.Equal(t, http.StatusOK, w.Code)
			actual := strings.TrimSpace(w.Body.String())
			if actual != tt.expected && actual != tt.reverse {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, tt.expected)
			}
		})
	}
}

func TestCreateZone(t *testing.T) {
	otherCreatedAt := createdAt.Add(-1 * time.Second)
	otherWS := &pkg.WaterSchedule{
		ID:        babyapi.ID{ID: id2},
		Duration:  &pkg.Duration{Duration: time.Second * 10},
		Interval:  &pkg.Duration{Duration: time.Hour * 24},
		StartTime: pkg.NewStartTime(otherCreatedAt),
	}
	gardenWithZone := createExampleGarden()
	gardenWithZone.ID = babyapi.ID{ID: id2}
	one := uint(1)
	gardenWithZone.MaxZones = &one

	// Predict NextWaterTime so I can test it better
	now := clock.Now()
	expectedNextWaterTime := time.Date(now.Year(), now.Month(), now.Day(), createdAt.Hour(), createdAt.Minute(), createdAt.Second(), createdAt.Nanosecond(), createdAt.Location())
	if now.After(expectedNextWaterTime) {
		expectedNextWaterTime = expectedNextWaterTime.Add(24 * time.Hour)
	}
	expectedNextWaterTimeWithSkip := expectedNextWaterTime.Add(72 * time.Hour)

	tests := []struct {
		name           string
		waterSchedules []*pkg.WaterSchedule
		garden         *pkg.Garden
		body           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			`{"name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			fmt.Sprintf(`{"name":"test-zone","id":"[0-9a-v]{20}","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":null,"next_water":{"time":"%d-%02d-%02dT11:24:52-07:00","duration":"1s","water_schedule_id":"c5cvhpcbcv45e8bp16dg"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/action"},{"rel":"history","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/history"}\]}`, expectedNextWaterTime.Year(), expectedNextWaterTime.Month(), expectedNextWaterTime.Day()),
			http.StatusCreated,
		},
		{
			"SuccessfulWithSkipCount",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			`{"name":"test-zone","skip_count":3,"position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			fmt.Sprintf(`{"name":"test-zone","id":"[0-9a-v]{20}","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":3,"next_water":{"time":"%d-%02d-%02dT11:24:52-07:00","duration":"1s","water_schedule_id":"c5cvhpcbcv45e8bp16dg","message":"skip_count 3 affected the time"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/action"},{"rel":"history","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/history"}\]}`, expectedNextWaterTimeWithSkip.Year(), expectedNextWaterTimeWithSkip.Month(), expectedNextWaterTimeWithSkip.Day()),
			http.StatusCreated,
		},
		{
			"SuccessfulMultipleWaterSchedules",
			[]*pkg.WaterSchedule{createExampleWaterSchedule(), otherWS},
			createExampleGarden(),
			`{"name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg","chkodpg3lcj13q82mq40"]}`,
			`{"name":"test-zone","id":"[0-9a-v]{20}","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg","chkodpg3lcj13q82mq40"\],"skip_count":null,"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:51-07:00","duration":"10s","water_schedule_id":"chkodpg3lcj13q82mq40"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/action"},{"rel":"history","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/history"}\]}`,
			http.StatusCreated,
		},
		{
			"SuccessfulWithGardenIDSet",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			`{"garden_id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			fmt.Sprintf(`{"name":"test-zone","id":"[0-9a-v]{20}","garden_id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":null,"next_water":{"time":"%d-%02d-%02dT11:24:52-07:00","duration":"1s","water_schedule_id":"c5cvhpcbcv45e8bp16dg"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/action"},{"rel":"history","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/history"}\]}`, expectedNextWaterTime.Year(), expectedNextWaterTime.Month(), expectedNextWaterTime.Day()),
			http.StatusCreated,
		},
		{
			"ErrorCannotSetGardenIDDifferentFromPath",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			`{"garden_id":"chkodpg3lcj13q82mq40","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"garden_id for zone must match URL path"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorCannotSetID",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			`{"id":"c5cvhpcbcv45e8bp16dg","garden_id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"unable to manually set ID"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorNegativeZonePosition",
			nil,
			createExampleGarden(),
			`{"name":"test-zone","position":-1,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"json: cannot unmarshal number -1 into Go struct field Zone.position of type uint"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorMaxZonesExceeded",
			nil,
			gardenWithZone,
			`{"name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"adding a Zone would exceed Garden's max_zones=1"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorInvalidZonePosition",
			nil,
			createExampleGarden(),
			`{"name":"test-zone","position":2,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"position invalid for Garden with max_zones=2"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorBadRequestBadJSON",
			nil,
			createExampleGarden(),
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true \(expecting 'r'\)"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorWaterScheduleNotFound",
			nil,
			createExampleGarden(),
			`{"name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"error getting WaterSchedule with ID \\"c5cvhpcbcv45e8bp16dg\\": resource not found"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := setupStorage(t, tt.garden)

			for _, ws := range tt.waterSchedules {
				err := storageClient.WaterSchedules.Set(context.Background(), ws)
				assert.NoError(t, err)
			}

			zr := NewZonesAPI()
			zr.setup(storageClient, nil, worker.NewWorker(storageClient, nil, nil, slog.Default()))

			for _, ws := range tt.waterSchedules {
				err := zr.worker.ScheduleWaterAction(ws)
				assert.NoError(t, err)
			}
			zr.worker.StartAsync()
			defer zr.worker.Stop()

			r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/gardens/%s/zones", tt.garden.ID), strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestWithParentRoute(t, zr.API, tt.garden, "Gardens", "/gardens", r)

			assert.Equal(t, tt.code, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestUpdateZonePUT(t *testing.T) {
	otherCreatedAt := createdAt.Add(-1 * time.Second)
	otherWS := &pkg.WaterSchedule{
		ID:        babyapi.ID{ID: id2},
		Duration:  &pkg.Duration{Duration: time.Second * 10},
		Interval:  &pkg.Duration{Duration: time.Hour * 24},
		StartTime: pkg.NewStartTime(otherCreatedAt),
	}
	gardenWithZone := createExampleGarden()
	gardenWithZone.ID = babyapi.ID{ID: id2}
	one := uint(1)
	gardenWithZone.MaxZones = &one

	tests := []struct {
		name           string
		waterSchedules []*pkg.WaterSchedule
		garden         *pkg.Garden
		body           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			``,
			http.StatusOK,
		},
		{
			"SuccessfulWithGardenIDSet",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			`{"id":"c5cvhpcbcv45e8bp16dg","garden_id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			``,
			http.StatusOK,
		},
		{
			"ErrorCannotSetGardenIDDifferentFromPath",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			`{"id":"c5cvhpcbcv45e8bp16dg","garden_id":"chkodpg3lcj13q82mq40","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"garden_id for zone must match URL path"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorMissingID",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			`{"name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"missing required id field"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorWrongID",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			`{"id":"chkodpg3lcj13q82mq40","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"id must match URL path"}`,
			http.StatusBadRequest,
		},
		{
			"SuccessfulWithSkipCount",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","skip_count":3,"position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			``,
			http.StatusOK,
		},
		{
			"SuccessfulMultipleWaterSchedules",
			[]*pkg.WaterSchedule{createExampleWaterSchedule(), otherWS},
			createExampleGarden(),
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg","chkodpg3lcj13q82mq40"]}`,
			``,
			http.StatusOK,
		},
		{
			"ErrorNegativeZonePosition",
			nil,
			createExampleGarden(),
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":-1,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"json: cannot unmarshal number -1 into Go struct field Zone.position of type uint"}`,
			http.StatusBadRequest,
		},
		{
			"MaxZonesEdit_NoErrorMaxZonesExceeded",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			gardenWithZone,
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","skip_count":3,"position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			``,
			http.StatusOK,
		},
		{
			"ErrorInvalidZonePosition",
			nil,
			createExampleGarden(),
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":2,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"position invalid for Garden with max_zones=2"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorBadRequestBadJSON",
			nil,
			createExampleGarden(),
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true \(expecting 'r'\)"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorWaterScheduleNotFound",
			nil,
			createExampleGarden(),
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"error getting WaterSchedule with ID \\"c5cvhpcbcv45e8bp16dg\\": resource not found"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := setupStorage(t, tt.garden)

			zone := createExampleZone()
			zone.GardenID = tt.garden.ID.ID
			err := storageClient.Zones.Set(context.Background(), zone)
			assert.NoError(t, err)

			for _, ws := range tt.waterSchedules {
				err := storageClient.WaterSchedules.Set(context.Background(), ws)
				assert.NoError(t, err)
			}

			zr := NewZonesAPI()
			zr.setup(storageClient, nil, worker.NewWorker(storageClient, nil, nil, slog.Default()))

			for _, ws := range tt.waterSchedules {
				err := zr.worker.ScheduleWaterAction(ws)
				assert.NoError(t, err)
			}
			zr.worker.StartAsync()
			defer zr.worker.Stop()

			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/gardens/%s/zones/%s", tt.garden.ID, zone.ID), strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestWithParentRoute(t, zr.API, tt.garden, "Gardens", "/gardens", r)

			assert.Equal(t, tt.code, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestCreateZonePUT(t *testing.T) {
	otherCreatedAt := createdAt.Add(-1 * time.Second)
	otherWS := &pkg.WaterSchedule{
		ID:        babyapi.ID{ID: id2},
		Duration:  &pkg.Duration{Duration: time.Second * 10},
		Interval:  &pkg.Duration{Duration: time.Hour * 24},
		StartTime: pkg.NewStartTime(otherCreatedAt),
	}
	gardenWithZone := createExampleGarden()
	gardenWithZone.ID = babyapi.ID{ID: id2}
	one := uint(1)
	gardenWithZone.MaxZones = &one

	tests := []struct {
		name           string
		waterSchedules []*pkg.WaterSchedule
		garden         *pkg.Garden
		id             string
		body           string
		expectedRegexp string
		code           int
	}{
		{
			"Successful",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			"c5cvhpcbcv45e8bp16dg",
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			``,
			http.StatusOK,
		},
		{
			"SuccessfulWithGardenIDSet",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			"c5cvhpcbcv45e8bp16dg",
			`{"id":"c5cvhpcbcv45e8bp16dg","garden_id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			``,
			http.StatusOK,
		},
		{
			"ErrorCannotSetGardenIDDifferentFromPath",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			"c5cvhpcbcv45e8bp16dg",
			`{"id":"c5cvhpcbcv45e8bp16dg","garden_id":"chkodpg3lcj13q82mq40","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"garden_id for zone must match URL path"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorMissingID",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			"c5cvhpcbcv45e8bp16dg",
			`{"name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"missing required id field"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorWrongID",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			"c5cvhpcbcv45e8bp16dg",
			`{"id":"chkodpg3lcj13q82mq40","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"id must match URL path"}`,
			http.StatusBadRequest,
		},
		{
			"SuccessfulWithSkipCount",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			"c5cvhpcbcv45e8bp16dg",
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","skip_count":3,"position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			``,
			http.StatusOK,
		},
		{
			"SuccessfulMultipleWaterSchedules",
			[]*pkg.WaterSchedule{createExampleWaterSchedule(), otherWS},
			createExampleGarden(),
			"c5cvhpcbcv45e8bp16dg",
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg","chkodpg3lcj13q82mq40"]}`,
			``,
			http.StatusOK,
		},
		{
			"ErrorNegativeZonePosition",
			nil,
			createExampleGarden(),
			"c5cvhpcbcv45e8bp16dg",
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":-1,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"json: cannot unmarshal number -1 into Go struct field Zone.position of type uint"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorMaxZonesExceeded",
			nil,
			gardenWithZone,
			"chkodpg3lcj13q82mq40",
			`{"id":"chkodpg3lcj13q82mq40","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"adding a Zone would exceed Garden's max_zones=1"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorInvalidZonePosition",
			nil,
			createExampleGarden(),
			"c5cvhpcbcv45e8bp16dg",
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":2,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"position invalid for Garden with max_zones=2"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorBadRequestBadJSON",
			nil,
			createExampleGarden(),
			"c5cvhpcbcv45e8bp16dg",
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true \(expecting 'r'\)"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorWaterScheduleNotFound",
			nil,
			createExampleGarden(),
			"c5cvhpcbcv45e8bp16dg",
			`{"id":"c5cvhpcbcv45e8bp16dg","name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"error getting WaterSchedule with ID \\"c5cvhpcbcv45e8bp16dg\\": resource not found"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := setupStorage(t, tt.garden)

			for _, ws := range tt.waterSchedules {
				err := storageClient.WaterSchedules.Set(context.Background(), ws)
				assert.NoError(t, err)
			}

			zr := NewZonesAPI()
			zr.setup(storageClient, nil, worker.NewWorker(storageClient, nil, nil, slog.Default()))

			for _, ws := range tt.waterSchedules {
				err := zr.worker.ScheduleWaterAction(ws)
				assert.NoError(t, err)
			}
			zr.worker.StartAsync()
			defer zr.worker.Stop()

			r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/gardens/%s/zones/%s", tt.garden.ID, tt.id), strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestWithParentRoute(t, zr.API, tt.garden, "Gardens", "/gardens", r)

			assert.Equal(t, tt.code, w.Code)
			assert.Regexp(t, tt.expectedRegexp, strings.TrimSpace(w.Body.String()))
		})
	}
}

func TestWaterHistory(t *testing.T) {
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
				influxdbClient.On("GetWaterHistory", mock.Anything, id.String(), "test-garden", time.Hour*72, uint64(5)).Return([]pkg.WaterHistory{}, nil)
				influxdbClient.On("Close")
			},
			"",
			`{"history":[],"count":0,"average":"0s","total":"0s"}`,
			http.StatusOK,
		},
		{
			"SuccessfulWaterHistory",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetWaterHistory", mock.Anything, id.String(), "test-garden", time.Hour*72, uint64(5)).
					Return([]pkg.WaterHistory{
						{
							Duration:    pkg.Duration{Duration: 3 * time.Second},
							Status:      pkg.WaterStatusCompleted,
							Source:      string(action.SourceCommand),
							SentAt:      recordTime,
							StartedAt:   recordTime,
							CompletedAt: recordTime,
							EventID:     "00000000000000000000",
						},
					}, nil)
				influxdbClient.On("Close")
			},
			"",
			`{"history":[{"duration":"3s","event_id":"00000000000000000000","status":"complete","source":"command","sent_at":"2021-10-03T11:24:52.891386-07:00","started_at":"2021-10-03T11:24:52.891386-07:00","completed_at":"2021-10-03T11:24:52.891386-07:00"}],"count":1,"average":"3s","total":"3s"}`,
			http.StatusOK,
		},
		{
			"SuccessfulWaterHistory_SentOnly",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetWaterHistory", mock.Anything, id.String(), "test-garden", time.Hour*72, uint64(5)).
					Return([]pkg.WaterHistory{
						{
							Duration: pkg.Duration{Duration: 3 * time.Second},
							Status:   pkg.WaterStatusSent,
							Source:   string(action.SourceCommand),
							SentAt:   recordTime,
							EventID:  "00000000000000000000",
						},
					}, nil)
				influxdbClient.On("Close")
			},
			"",
			`{"history":[{"duration":"3s","event_id":"00000000000000000000","status":"sent","source":"command","sent_at":"2021-10-03T11:24:52.891386-07:00"}],"count":0,"average":"0s","total":"0s"}`,
			http.StatusOK,
		},
		{
			"SuccessfulWaterHistoryWithLimit",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetWaterHistory", mock.Anything, id.String(), "test-garden", time.Hour*72, uint64(1)).
					Return([]pkg.WaterHistory{
						{
							Duration:    pkg.Duration{Duration: 3 * time.Second},
							Status:      pkg.WaterStatusCompleted,
							Source:      string(action.SourceCommand),
							SentAt:      recordTime,
							StartedAt:   recordTime,
							CompletedAt: recordTime,
							EventID:     "00000000000000000000",
						},
					}, nil)
				influxdbClient.On("Close")
			},
			"?limit=1",
			`{"history":[{"duration":"3s","event_id":"00000000000000000000","status":"complete","source":"command","sent_at":"2021-10-03T11:24:52.891386-07:00","started_at":"2021-10-03T11:24:52.891386-07:00","completed_at":"2021-10-03T11:24:52.891386-07:00"}],"count":1,"average":"3s","total":"3s"}`,
			http.StatusOK,
		},
		{
			"InfluxDBClientError",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetWaterHistory", mock.Anything, id.String(), "test-garden", time.Hour*72, uint64(5)).
					Return([]pkg.WaterHistory{}, errors.New("influxdb error"))
				influxdbClient.On("Close")
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
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			zr := NewZonesAPI()
			zr.setup(storageClient, influxdbClient, worker.NewWorker(storageClient, influxdbClient, nil, slog.Default()))

			garden := createExampleGarden()
			zone := createExampleZone()

			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)
			err = storageClient.Zones.Set(context.Background(), zone)
			assert.NoError(t, err)

			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/gardens/%s/zones/%s/history%s", garden.ID, zone.ID, tt.queryParams), http.NoBody)
			r.Header.Set("X-TZ-Offset", "420")
			w := babytest.TestWithParentRoute(t, zr.API, garden, "Gardens", "/gardens", r)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.expected, strings.TrimSpace(w.Body.String()))

			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestGetNextWaterTime(t *testing.T) {
	clock.MockTime()
	defer clock.Reset()

	activePeriodWS := createExampleWaterSchedule()
	activePeriodWS.ActivePeriod = &pkg.ActivePeriod{
		StartMonth: "September",
		EndMonth:   "November",
	}

	tests := []struct {
		name         string
		ws           *pkg.WaterSchedule
		expectedTime time.Time
	}{
		{
			"ZeroSkip",
			createExampleWaterSchedule(),
			time.Date(2023, time.August, 23, 18, 24, 52, 0, time.UTC),
		},
		{
			"OutsideActivePeriod",
			activePeriodWS,
			time.Date(2023, time.September, 0o1, 18, 24, 52, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			zr := &ZonesAPI{
				worker: worker.NewWorker(storageClient, nil, nil, slog.Default()),
			}

			err = zr.worker.ScheduleWaterAction(tt.ws)
			assert.NoError(t, err)
			zr.worker.StartAsync()
			defer zr.worker.Stop()

			nextWaterTime := zr.worker.GetNextWaterTime(tt.ws)
			require.Equal(t, tt.expectedTime, *nextWaterTime)
		})
	}
}

func TestZoneRequest(t *testing.T) {
	pos := uint(0)
	tests := []struct {
		name string
		z    *pkg.Zone
		err  string
	}{
		{
			"EmptyPositionError",
			&pkg.Zone{
				Name: "zone",
			},
			"missing required position field",
		},
		{
			"EmptyNameError",
			&pkg.Zone{
				Position:         &pos,
				WaterScheduleIDs: []xid.ID{id},
			},
			"missing required name field",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		pr := &pkg.Zone{
			Name:             "zone",
			Position:         &pos,
			WaterScheduleIDs: []xid.ID{id},
		}
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		err := pr.Bind(r)
		assert.NoError(t, err)
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			err := tt.z.Bind(r)
			assert.Error(t, err)
			assert.Equal(t, tt.err, err.Error())
		})
	}
}

func TestUpdateZoneRequest(t *testing.T) {
	pp := uint(0)
	now := clock.Now()
	tests := []struct {
		name string
		z    *pkg.Zone
		err  string
	}{
		{
			"ManualSpecificationOfIDError",
			&pkg.Zone{ID: babyapi.NewID()},
			"updating ID is not allowed",
		},
		{
			"EndDateError",
			&pkg.Zone{
				EndDate: &now,
			},
			"to end-date a Zone, please use the DELETE endpoint",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		pr := &pkg.Zone{
			Name:     "zone",
			Position: &pp,
		}
		r := httptest.NewRequest(http.MethodPatch, "/", nil)
		err := pr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading ZoneRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPatch, "/", nil)
			err := tt.z.Bind(r)
			if err == nil {
				t.Error("Expected error reading ZoneRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}
