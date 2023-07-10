package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		ID:               id,
		CreatedAt:        &createdAt,
		WaterScheduleIDs: []xid.ID{id},
	}
}

func TestZoneContextMiddleware(t *testing.T) {
	zr := ZonesResource{}
	zone := createExampleZone()
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		p := getZoneFromContext(r.Context())
		if zone != p {
			t.Errorf("Unexpected Zone saved in request context. Expected %v but got %v", zone, p)
		}
		render.Status(r, http.StatusOK)
	}

	router := chi.NewRouter()
	router.Route(fmt.Sprintf("/zone/{%s}", zonePathParam), func(r chi.Router) {
		r.Use(zr.zoneContextMiddleware)
		r.Get("/", testHandler)
	})

	tests := []struct {
		name     string
		zone     *pkg.Zone
		path     string
		code     int
		expected string
	}{
		{
			"Successful",
			zone,
			"/zone/c5cvhpcbcv45e8bp16dg",
			http.StatusOK,
			"",
		},
		{
			"ErrorInvalidID",
			zone,
			"/zone/not-an-xid",
			http.StatusBadRequest,
			`{"status":"Invalid request.","error":"xid: invalid ID"}`,
		},
		{
			"NotFoundError",
			nil,
			"/zone/c5cvhpcbcv45e8bp16dg",
			http.StatusNotFound,
			`{"status":"Resource not found."}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			garden := createExampleGarden()
			garden.Zones[zone.ID] = tt.zone
			ctx := context.WithValue(context.Background(), gardenCtxKey, garden)
			r := httptest.NewRequest("GET", tt.path, nil).WithContext(ctx)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.code {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.code)
			}
			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			if actual != tt.expected {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, tt.expected)
			}
		})
	}
}

func TestZoneRestrictEndDatedMiddleware(t *testing.T) {
	zone := createExampleZone()
	endDatedZone := createExampleZone()
	endDate := time.Now().Add(-1 * time.Minute)
	endDatedZone.EndDate = &endDate
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, http.StatusOK)
	}

	router := chi.NewRouter()
	router.Route("/zone", func(r chi.Router) {
		r.Use(restrictEndDatedMiddleware("Zone", zoneCtxKey))
		r.Get("/", testHandler)
	})

	tests := []struct {
		name     string
		zone     *pkg.Zone
		code     int
		expected string
	}{
		{
			"ZoneNotEndDated",
			zone,
			http.StatusOK,
			"",
		},
		{
			"ZoneEndDated",
			endDatedZone,
			http.StatusBadRequest,
			`{"status":"Invalid request.","error":"resource not available for end-dated Zone"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), zoneCtxKey, tt.zone)
			r := httptest.NewRequest("GET", "/zone", nil).WithContext(ctx)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.code {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.code)
			}
			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			if actual != tt.expected {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, tt.expected)
			}
		})
	}
}

func TestGetZone(t *testing.T) {
	one := 1
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
			`{"name":"test-zone","id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":null,"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52.891386-07:00","duration":"1s","water_schedule_id":"c5cvhpcbcv45e8bp16dg"},"links":\[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}\]}`,
		},
		{
			"SuccessfulWithMoisture",
			false,
			[]*pkg.WaterSchedule{{
				ID:        id,
				Duration:  &pkg.Duration{Duration: time.Second},
				Interval:  &pkg.Duration{Duration: 24 * time.Hour},
				StartTime: &createdAt,
				WeatherControl: &weather.Control{
					SoilMoisture: &weather.SoilMoistureControl{
						MinimumMoisture: &one,
					},
				},
			}},
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetMoisture", mock.Anything, mock.Anything, mock.Anything).Return(float64(2), nil)
				influxdbClient.On("Close")
			},
			`{"name":"test-zone","id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":null,"weather_data":{"soil_moisture_percent":2},"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52.891386-07:00","duration":"1s","water_schedule_id":"c5cvhpcbcv45e8bp16dg"},"links":\[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}\]}`,
		},
		{
			"SuccessfulWithMoistureRainAndTemperatureData",
			false,
			[]*pkg.WaterSchedule{{
				ID:        id,
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Hour},
				StartTime: &createdAt,
				WeatherControl: &weather.Control{
					SoilMoisture: &weather.SoilMoistureControl{
						MinimumMoisture: &one,
					},
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
				influxdbClient.On("GetMoisture", mock.Anything, mock.Anything, mock.Anything).Return(float64(2), nil)
				influxdbClient.On("Close")
			},
			`{"name":"test-zone","id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":null,"weather_data":{"rain":{"mm":25.4,"scale_factor":0},"average_temperature":{"celsius":80,"scale_factor":1.5},"soil_moisture_percent":2},"next_water":{"time":"2023-\d\d-\d\dT11:24:52.891386-07:00","duration":"0s","water_schedule_id":"c5cvhpcbcv45e8bp16dg"},"links":\[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}\]}`,
		},
		{
			"SuccessfulWithMoistureRainAndTemperatureDataButWeatherDataExcluded",
			true,
			[]*pkg.WaterSchedule{{
				ID:        id,
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				Duration:  &pkg.Duration{Duration: time.Hour},
				StartTime: &createdAt,
				WeatherControl: &weather.Control{
					SoilMoisture: &weather.SoilMoistureControl{
						MinimumMoisture: &one,
					},
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
			func(influxdbClient *influxdb.MockClient) {},
			`{"name":"test-zone","id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":null,"next_water":{"time":"2023-\d\d-\d\dT11:24:52.891386-07:00","duration":"1h0m0s","water_schedule_id":"c5cvhpcbcv45e8bp16dg"},"links":\[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}\]}`,
		},
		{
			"ErrorGettingMoisture",
			false,
			[]*pkg.WaterSchedule{{
				ID:        id,
				Duration:  &pkg.Duration{Duration: time.Second},
				Interval:  &pkg.Duration{Duration: time.Hour * 24},
				StartTime: &createdAt,
				WeatherControl: &weather.Control{
					SoilMoisture: &weather.SoilMoistureControl{
						MinimumMoisture: &one,
					},
				},
			}},
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetMoisture", mock.Anything, mock.Anything, mock.Anything).Return(float64(2), errors.New("influxdb error"))
				influxdbClient.On("Close")
			},
			`{"name":"test-zone","id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":null,"weather_data":{},"next_water":{"time":"\d\d\d\d-\d\d-\d\dT11:24:52.891386-07:00","duration":"1s","water_schedule_id":"c5cvhpcbcv45e8bp16dg"},"links":\[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}\]}`,
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
				err = storageClient.SaveWaterSchedule(ws)
				assert.NoError(t, err)
			}

			err = storageClient.SaveWeatherClientConfig(createExampleWeatherClientConfig())
			assert.NoError(t, err)

			zr := ZonesResource{
				influxdbClient: influxdbClient,
				storageClient:  storageClient,
				worker:         worker.NewWorker(storageClient, influxdbClient, nil, logrus.New()),
			}
			zr.worker.StartAsync()

			for _, ws := range tt.waterSchedules {
				err := zr.worker.ScheduleWaterAction(ws)
				assert.NoError(t, err)
			}

			garden := createExampleGarden()
			zone := createExampleZone()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			zoneCtx := context.WithValue(gardenCtx, zoneCtxKey, zone)
			r := httptest.NewRequest("GET", fmt.Sprintf("/zone?exclude_weather_data=%t", tt.excludeWeatherData), nil).WithContext(zoneCtx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(zr.getZone)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != http.StatusOK {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
			}

			// check HTTP response body
			matcher := regexp.MustCompile(tt.expectedRegexp)
			actual := strings.TrimSpace(w.Body.String())
			if !matcher.MatchString(actual) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
			}

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
			func(mqttClient *mqtt.MockClient) {},
			"bad request",
			`{"status":"Invalid request.","error":"invalid character 'b' looking for beginning of value"}`,
			http.StatusBadRequest,
		},
		{
			"SuccessfulWaterAction",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("WaterTopic", "test-garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
			},
			`{"water":{"duration":1000}}`,
			"null",
			http.StatusAccepted,
		},
		{
			"ExecuteErrorForWaterAction",
			func(mqttClient *mqtt.MockClient) {
				mqttClient.On("WaterTopic", "test-garden").Return("", errors.New("template error"))
			},
			`{"water":{"duration":1000}}`,
			`{"status":"Server Error.","error":"unable to execute WaterAction: unable to fill MQTT topic template: template error"}`,
			http.StatusInternalServerError,
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

			zr := ZonesResource{
				worker: worker.NewWorker(storageClient, nil, mqttClient, logrus.New()),
			}
			garden := createExampleGarden()
			zone := createExampleZone()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			zoneCtx := context.WithValue(gardenCtx, zoneCtxKey, zone)
			r := httptest.NewRequest("POST", "/zone", strings.NewReader(tt.body)).WithContext(zoneCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(zr.zoneAction)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.status {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.status)
			}

			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			if actual != tt.expected {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, tt.expected)
			}
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
			`{"name":"new name","id":"c5cvhpcbcv45e8bp16dg","position":0,"created_at":"2021-10-03T11:24:52.891386-07:00","water_schedule_ids":["c5cvhpcbcv45e8bp16dg"],"skip_count":null,"next_water":{"message":"no active WaterSchedules"},"links":[{"rel":"self","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg"},{"rel":"garden","href":"/gardens/c5cvhpcbcv45e8bp16dg"},{"rel":"action","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/action"},{"rel":"history","href":"/gardens/c5cvhpcbcv45e8bp16dg/zones/c5cvhpcbcv45e8bp16dg/history"}]}`,
			http.StatusOK,
		},
		{
			"BadRequest",
			"this is not json",
			`{"status":"Invalid request.","error":"invalid character 'h' in literal true (expecting 'r')"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorWaterScheduleNotFound",
			`{"water_schedule_ids":["chkodpg3lcj13q82mq40"]}`,
			`{"status":"Invalid request.","error":"unable to update Zone with non-existent WaterSchedule [\"chkodpg3lcj13q82mq40\"]"}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := setupZonePlantGardenStorage(t)

			err := storageClient.SaveWaterSchedule(createExampleWaterSchedule())
			assert.NoError(t, err)

			zr := ZonesResource{
				storageClient: storageClient,
				worker:        worker.NewWorker(storageClient, nil, nil, logrus.New()),
			}
			garden := createExampleGarden()
			zone := createExampleZone()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			zoneCtx := context.WithValue(gardenCtx, zoneCtxKey, zone)
			r := httptest.NewRequest("PATCH", "/zone", strings.NewReader(tt.body)).WithContext(zoneCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(zr.updateZone)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.status {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.status)
			}

			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			if actual != tt.expected {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, tt.expected)
			}
		})
	}
}

func TestEndDateZone(t *testing.T) {
	now := time.Now()
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
			`{"name":"test-zone","id":"[0-9a-v]{20}","position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","end_date":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":null,"next_water":{},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"}\]}`,
			http.StatusOK,
		},
		{
			"SuccessfullyDeleteZone",
			endDatedZone,
			"",
			http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := setupZonePlantGardenStorage(t)

			err := storageClient.SaveWaterSchedule(createExampleWaterSchedule())
			assert.NoError(t, err)

			zr := ZonesResource{
				storageClient: storageClient,
				worker:        worker.NewWorker(storageClient, nil, nil, logrus.New()),
			}

			garden := createExampleGarden()
			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			zoneCtx := context.WithValue(gardenCtx, zoneCtxKey, tt.zone)
			r := httptest.NewRequest("DELETE", "/zone", nil).WithContext(zoneCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(zr.endDateZone)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.code {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.code)
			}

			// check HTTP response body
			matcher := regexp.MustCompile(tt.expectedRegexp)
			actual := strings.TrimSpace(w.Body.String())
			if !matcher.MatchString(actual) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
			}
		})
	}
}

func TestGetAllZones(t *testing.T) {
	storageClient := setupWaterScheduleStorage(t)
	zr := ZonesResource{
		worker:        worker.NewWorker(storageClient, nil, nil, logrus.New()),
		storageClient: storageClient,
	}
	garden := createExampleGarden()
	zone := createExampleZone()
	endDatedZone := createExampleZone()
	endDatedZone.ID = xid.New()
	now := time.Now()
	endDatedZone.EndDate = &now
	garden.Zones[zone.ID] = zone
	garden.Zones[endDatedZone.ID] = endDatedZone

	gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)

	tests := []struct {
		name      string
		targetURL string
		expected  []*pkg.Zone
	}{
		{
			"SuccessfulEndDatedFalse",
			"/zone",
			[]*pkg.Zone{zone},
		},
		{
			"SuccessfulEndDatedTrue",
			"/zone?end_dated=true",
			[]*pkg.Zone{zone, endDatedZone},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", tt.targetURL, nil).WithContext(gardenCtx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(zr.getAllZones)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != http.StatusOK {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, http.StatusOK)
			}

			zoneJSON, _ := json.Marshal(zr.NewAllZonesResponse(context.Background(), tt.expected, garden, false))
			// When the expected result contains more than one Zone, on some occasions it might be out of order
			var reverseZoneJSON []byte
			if len(tt.expected) > 1 {
				reverseZoneJSON, _ = json.Marshal(zr.NewAllZonesResponse(context.Background(), []*pkg.Zone{tt.expected[1], tt.expected[0]}, &pkg.Garden{}, false))
			}
			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			if actual != string(zoneJSON) && actual != string(reverseZoneJSON) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, string(zoneJSON))
			}

		})
	}
}

func TestCreateZone(t *testing.T) {
	otherCreatedAt := createdAt.Add(-1 * time.Hour)
	otherWS := &pkg.WaterSchedule{
		ID:        id2,
		Duration:  &pkg.Duration{Duration: time.Second * 10},
		Interval:  &pkg.Duration{Duration: time.Hour * 24},
		StartTime: &otherCreatedAt,
	}
	gardenWithZone := createExampleGarden()
	gardenWithZone.Zones[xid.New()] = &pkg.Zone{}
	gardenWithZone.Zones[xid.New()] = &pkg.Zone{}

	// Predict NextWaterTime so I can test it better
	now := time.Now()
	expectedNextWaterTime := time.Date(now.Year(), now.Month(), now.Day(), createdAt.Hour(), createdAt.Minute(), createdAt.Second(), createdAt.Nanosecond(), createdAt.Location())
	if now.Hour() >= 11 {
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
			`{"name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"],"start_time":"2021-10-03T11:24:52.891386-07:00"}}`,
			fmt.Sprintf(`{"name":"test-zone","id":"[0-9a-v]{20}","position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":null,"next_water":{"time":"%d-%02d-%02dT11:24:52.891386-07:00","duration":"1s","water_schedule_id":"c5cvhpcbcv45e8bp16dg"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/action"},{"rel":"history","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/history"}\]}`, expectedNextWaterTime.Year(), expectedNextWaterTime.Month(), expectedNextWaterTime.Day()),
			http.StatusCreated,
		},
		{
			"SuccessfulWithSkipCount",
			[]*pkg.WaterSchedule{createExampleWaterSchedule()},
			createExampleGarden(),
			`{"name":"test-zone","skip_count":3,"position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"],"start_time":"2021-10-03T11:24:52.891386-07:00"}}`,
			fmt.Sprintf(`{"name":"test-zone","id":"[0-9a-v]{20}","position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg"\],"skip_count":3,"next_water":{"time":"%d-%02d-%02dT11:24:52.891386-07:00","duration":"1s","water_schedule_id":"c5cvhpcbcv45e8bp16dg","message":"skip_count 3 affected the time"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/action"},{"rel":"history","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/history"}\]}`, expectedNextWaterTimeWithSkip.Year(), expectedNextWaterTimeWithSkip.Month(), expectedNextWaterTimeWithSkip.Day()),
			http.StatusCreated,
		},
		{
			"SuccessfulMultipleWaterSchedules",
			[]*pkg.WaterSchedule{createExampleWaterSchedule(), otherWS},
			createExampleGarden(),
			`{"name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg","chkodpg3lcj13q82mq40"],"start_time":"2021-10-03T11:24:52.891386-07:00"}}`,
			`{"name":"test-zone","id":"[0-9a-v]{20}","position":0,"created_at":"\d{4}-\d{2}-\d\dT\d\d:\d\d:\d\d\.\d+(-07:00|Z)","water_schedule_ids":\["c5cvhpcbcv45e8bp16dg","chkodpg3lcj13q82mq40"\],"skip_count":null,"next_water":{"time":"\d\d\d\d-\d\d-\d\dT10:24:52.891386-07:00","duration":"10s","water_schedule_id":"chkodpg3lcj13q82mq40"},"links":\[{"rel":"self","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}"},{"rel":"garden","href":"/gardens/[0-9a-v]{20}"},{"rel":"action","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/action"},{"rel":"history","href":"/gardens/[0-9a-v]{20}/zones/[0-9a-v]{20}/history"}\]}`,
			http.StatusCreated,
		},
		{
			"ErrorNegativeZonePosition",
			nil,
			createExampleGarden(),
			`{"name":"test-zone","position":-1,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"json: cannot unmarshal number -1 into Go struct field ZoneRequest.position of type uint"}`,
			http.StatusBadRequest,
		},
		{
			"ErrorMaxZonesExceeded",
			nil,
			gardenWithZone,
			`{"name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"]}`,
			`{"status":"Invalid request.","error":"adding a Zone would exceed Garden's max_zones=2"}`,
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
		// {
		// 	"ErrorWaterScheduleNotFound",
		// 	nil,
		// 	createExampleGarden(),
		// 	`{"name":"test-zone","position":0,"water_schedule_ids":["c5cvhpcbcv45e8bp16dg"],"start_time":"2021-10-03T11:24:52.891386-07:00"}}`,
		// 	`{"status":"Invalid request.","error":"unable to create Zone with non-existent WaterSchedule \[\\\"c5cvhpcbcv45e8bp16dg\\\"\]"}`,
		// 	http.StatusBadRequest,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := setupZonePlantGardenStorage(t)

			for _, ws := range tt.waterSchedules {
				err := storageClient.SaveWaterSchedule(ws)
				assert.NoError(t, err)
			}

			zr := ZonesResource{
				storageClient: storageClient,
				worker:        worker.NewWorker(storageClient, nil, nil, logrus.New()),
			}
			for _, ws := range tt.waterSchedules {
				err := zr.worker.ScheduleWaterAction(ws)
				assert.NoError(t, err)
			}
			zr.worker.StartAsync()
			defer zr.worker.Stop()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, tt.garden)
			r := httptest.NewRequest("POST", "/zone", strings.NewReader(tt.body)).WithContext(gardenCtx)
			r.Header.Add("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h := http.HandlerFunc(zr.createZone)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.code {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.code)
			}

			// check HTTP response body
			matcher := regexp.MustCompile(tt.expectedRegexp)
			actual := strings.TrimSpace(w.Body.String())
			if !matcher.MatchString(actual) {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, matcher.String())
			}
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
				influxdbClient.On("GetWaterHistory", mock.Anything, uint(0), "test-garden", time.Hour*72, uint64(0)).Return([]map[string]interface{}{}, nil)
				influxdbClient.On("Close")
			},
			"",
			`{"history":null,"count":0,"average":"0s","total":"0s"}`,
			http.StatusOK,
		},
		{
			"SuccessfulWaterHistory",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetWaterHistory", mock.Anything, uint(0), "test-garden", time.Hour*72, uint64(0)).
					Return([]map[string]interface{}{{"Duration": 3000, "RecordTime": recordTime}}, nil)
				influxdbClient.On("Close")
			},
			"",
			`{"history":[{"duration":"3s","record_time":"2021-10-03T11:24:52.891386-07:00"}],"count":1,"average":"3s","total":"3s"}`,
			http.StatusOK,
		},
		{
			"SuccessfulWaterHistoryWithLimit",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetWaterHistory", mock.Anything, uint(0), "test-garden", time.Hour*72, uint64(1)).
					Return([]map[string]interface{}{
						{"Duration": 3000, "RecordTime": recordTime},
					}, nil)
				influxdbClient.On("Close")
			},
			"?limit=1",
			`{"history":[{"duration":"3s","record_time":"2021-10-03T11:24:52.891386-07:00"}],"count":1,"average":"3s","total":"3s"}`,
			http.StatusOK,
		},
		{
			"InfluxDBClientError",
			func(influxdbClient *influxdb.MockClient) {
				influxdbClient.On("GetWaterHistory", mock.Anything, uint(0), "test-garden", time.Hour*72, uint64(0)).
					Return([]map[string]interface{}{}, errors.New("influxdb error"))
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

			zr := ZonesResource{
				influxdbClient: influxdbClient,
			}
			garden := createExampleGarden()
			zone := createExampleZone()

			gardenCtx := context.WithValue(context.Background(), gardenCtxKey, garden)
			zoneCtx := context.WithValue(gardenCtx, zoneCtxKey, zone)
			r := httptest.NewRequest("GET", fmt.Sprintf("/history%s", tt.queryParams), nil).WithContext(zoneCtx)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(zr.waterHistory)

			h.ServeHTTP(w, r)

			// check HTTP response status code
			if w.Code != tt.status {
				t.Errorf("Unexpected status code: got %v, want %v", w.Code, tt.status)
			}

			// check HTTP response body
			actual := strings.TrimSpace(w.Body.String())
			if actual != tt.expected {
				t.Errorf("Unexpected response body:\nactual   = %v\nexpected = %v", actual, tt.expected)
			}
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestGetNextWaterTime(t *testing.T) {
	tests := []struct {
		name         string
		expectedDiff time.Duration
	}{
		{"ZeroSkip", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			zr := ZonesResource{
				worker: worker.NewWorker(storageClient, nil, nil, logrus.New()),
			}
			ws := createExampleWaterSchedule()

			err = zr.worker.ScheduleWaterAction(ws)
			assert.NoError(t, err)
			zr.worker.StartAsync()
			defer zr.worker.Stop()

			NextWaterTime := zr.worker.GetNextWaterTime(ws)
			NextWaterTimeWithSkip := zr.worker.GetNextWaterTime(ws)

			diff := NextWaterTimeWithSkip.Sub(*NextWaterTime)
			if diff != tt.expectedDiff {
				t.Errorf("Unexpected difference between next watering times: expected=%v, actual=%v", tt.expectedDiff, diff)
			}
		})
	}
}
