package integrationtests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/controller"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	fake_notification "github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/fake"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather/fake"
	"github.com/calvinmclean/automated-garden/garden-app/server"
	"github.com/rs/xid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	configFile = "testdata/config.yml"
)

var c *controller.Controller

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testing in short mode")
	}

	serverConfig, controllerConfig := getConfigs(t)

	api := server.NewAPI()
	err := api.Setup(serverConfig, true)
	require.NoError(t, err)

	c, err = controller.NewController(controllerConfig)
	require.NoError(t, err)

	go c.Start()
	go func() {
		serveErr := api.Serve(":8080")
		if serveErr != nil {
			panic(serveErr.Error())
		}
	}()

	defer c.Stop()
	defer api.Stop()

	time.Sleep(500 * time.Millisecond)

	t.Run("Garden", GardenTests)
	t.Run("Zone", ZoneTests)
	t.Run("WaterSchedule", WaterScheduleTests)
	t.Run("ControllerStartupNotification", ControllerStartupNotificationTest)
}

func getConfigs(t *testing.T) (server.Config, controller.Config) {
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	require.NoError(t, err)

	var serverConfig server.Config
	err = viper.Unmarshal(&serverConfig)
	require.NoError(t, err)
	serverConfig.LogConfig.Level = slog.LevelDebug.String()

	var controllerConfig controller.Config
	err = viper.Unmarshal(&controllerConfig)
	require.NoError(t, err)
	controllerConfig.LogConfig.Level = slog.LevelDebug.String()

	return serverConfig, controllerConfig
}

func CreateGardenTest(t *testing.T) string {
	var g server.GardenResponse

	t.Run("CreateGarden", func(t *testing.T) {
		status, err := makeRequest(http.MethodPost, "/gardens", `{
			"name": "Test",
			"topic_prefix": "test",
			"max_zones": 3,
			"light_schedule": {
				"duration": "14h",
				"start_time": "22:00:00-07:00"
			},
			"temperature_humidity_sensor": true
		}`, &g)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusCreated, status)
	})

	return g.ID.String()
}

func GardenTests(t *testing.T) {
	gardenID := CreateGardenTest(t)

	t.Run("GetGarden", func(t *testing.T) {
		var g server.GardenResponse
		status, err := makeRequest(http.MethodGet, "/gardens/"+gardenID, http.NoBody, &g)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)

		assert.Equal(t, gardenID, g.ID.String())
		assert.Equal(t, uint(3), *g.MaxZones)
		assert.Equal(t, uint(0), g.NumZones)
	})
	t.Run("ExecuteStopAction", func(t *testing.T) {
		status, err := makeRequest(
			http.MethodPost,
			fmt.Sprintf("/gardens/%s/action", gardenID),
			action.GardenAction{Stop: &action.StopAction{}},
			&struct{}{},
		)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, status)

		time.Sleep(100 * time.Millisecond)

		c.AssertStopActions(t, 1)
	})
	t.Run("ExecuteStopAllAction", func(t *testing.T) {
		status, err := makeRequest(
			http.MethodPost,
			fmt.Sprintf("/gardens/%s/action", gardenID),
			action.GardenAction{Stop: &action.StopAction{All: true}},
			&struct{}{},
		)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, status)

		time.Sleep(100 * time.Millisecond)

		c.AssertStopAllActions(t, 1)
	})
	for _, state := range []pkg.LightState{pkg.LightStateOn, pkg.LightStateOff, pkg.LightStateToggle} {
		t.Run("ExecuteLightAction"+state.String(), func(t *testing.T) {
			status, err := makeRequest(
				http.MethodPost,
				fmt.Sprintf("/gardens/%s/action", gardenID),
				action.GardenAction{Light: &action.LightAction{State: state}},
				&struct{}{},
			)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusAccepted, status)

			time.Sleep(100 * time.Millisecond)

			c.AssertLightActions(t, action.LightAction{State: state})
		})
	}
	t.Run("ExecuteLightActionWithDelay", func(t *testing.T) {
		// Create new Garden with LightOnTime in the near future, so LightDelay will assume the light is currently off,
		// meaning adhoc action is going to be predictably delayed
		maxZones := uint(1)
		startTime := pkg.NewStartTime(clock.Now().In(time.Local).Add(1 * time.Second).Truncate(time.Second))
		newGarden := &pkg.Garden{
			Name:        "TestGarden",
			TopicPrefix: "test",
			MaxZones:    &maxZones,
			LightSchedule: &pkg.LightSchedule{
				Duration:  &pkg.Duration{Duration: 14 * time.Hour},
				StartTime: startTime,
			},
		}

		var g server.GardenResponse
		status, err := makeRequest(http.MethodPost, "/gardens", newGarden, &g)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, status)

		// Execute light action with delay
		status, err = makeRequest(
			http.MethodPost,
			fmt.Sprintf("/gardens/%s/action", g.ID.String()),
			action.GardenAction{Light: &action.LightAction{
				State:       pkg.LightStateOff,
				ForDuration: &pkg.Duration{Duration: time.Second},
			}},
			&struct{}{},
		)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, status)

		time.Sleep(100 * time.Millisecond)

		// Make sure NextOnTime is correctly delayed
		var getG server.GardenResponse
		status, err = makeRequest(http.MethodGet, fmt.Sprintf("/gardens/%s", g.ID.String()), http.NoBody, &getG)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, startTime.Time.Add(1*time.Second), getG.NextLightAction.Time.Local())

		time.Sleep(3 * time.Second)

		// Check for light action turning it off, plus adhoc schedule to turn it back on
		c.AssertLightActions(t,
			action.LightAction{State: pkg.LightStateOff, ForDuration: &pkg.Duration{Duration: time.Second}},
			action.LightAction{State: pkg.LightStateOn},
		)
	})
	t.Run("ChangeLightScheduleStartTimeResetsLightSchedule", func(t *testing.T) {
		// Reschedule Light to turn in in 2 second, for 1 second
		newStartTimeDelay := 2 * time.Second
		newStartTime := pkg.NewStartTime(clock.Now().Add(newStartTimeDelay).Truncate(time.Second))

		t.Run("ModifyLightSchedule", func(t *testing.T) {
			var g server.GardenResponse
			status, err := makeRequest(http.MethodPatch, "/gardens/"+gardenID, pkg.Garden{
				LightSchedule: &pkg.LightSchedule{
					StartTime: newStartTime,
					Duration:  &pkg.Duration{Duration: time.Second},
				},
			}, &g)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, status)
			assert.Equal(t, newStartTime.String(), g.LightSchedule.StartTime.String())
		})

		time.Sleep(100 * time.Millisecond)

		t.Run("CheckNewNextOnTime", func(t *testing.T) {
			var g server.GardenResponse
			status, err := makeRequest(http.MethodGet, "/gardens/"+gardenID, nil, &g)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, status)
			assert.Equal(t, newStartTime.String(), pkg.NewStartTime(g.NextLightAction.Time.Local()).String())
			assert.Equal(t, pkg.LightStateOn, g.NextLightAction.State)
		})

		// wait a little extra
		time.Sleep(2*newStartTimeDelay + 500*time.Millisecond)

		// Assert both LightActions
		c.AssertLightActions(t,
			action.LightAction{State: pkg.LightStateOn},
			action.LightAction{State: pkg.LightStateOff},
		)
	})
	t.Run("GetGardenToCheckInfluxDBData", func(t *testing.T) {
		var g server.GardenResponse
		status, err := makeRequest(http.MethodGet, "/gardens/"+gardenID, http.NoBody, &g)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)

		// The health status timing can be inconsistent, so it should be retried
		retries := 1
		for g.Health.Status != pkg.HealthStatusUp && retries <= 5 {
			time.Sleep(time.Duration(retries) * time.Second)

			status, err := makeRequest(http.MethodGet, "/gardens/"+gardenID, http.NoBody, &g)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, status)

			retries++
		}

		assert.Equal(t, pkg.HealthStatusUp, g.Health.Status)
		assert.Equal(t, 50.0, g.TemperatureHumidityData.TemperatureCelsius)
		assert.Equal(t, 50.0, g.TemperatureHumidityData.HumidityPercentage)
	})
}

func CreateZoneTest(t *testing.T, gardenID, waterScheduleID string) string {
	var z server.ZoneResponse

	t.Run("CreateZone", func(t *testing.T) {
		status, err := makeRequest(http.MethodPost, fmt.Sprintf("/gardens/%s/zones", gardenID), fmt.Sprintf(`{
			"name": "Zone 1",
			"position": 0,
			"water_schedule_ids": ["%s"]
		}`, waterScheduleID), &z)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusCreated, status)
	})

	return z.ID.String()
}

func CreateWaterScheduleTest(t *testing.T) string {
	var ws server.WaterScheduleResponse

	t.Run("CreateWaterSchedule", func(t *testing.T) {
		status, err := makeRequest(http.MethodPost, "/water_schedules", `{
			"duration": "10s",
			"interval": "24h",
			"start_time": "08:00:00-07:00"
		}`, &ws)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusCreated, status)
	})

	return ws.ID.String()
}

func CreateWeatherClientTest(t *testing.T, opts fake.Config) xid.ID {
	var wcr weather.Config

	t.Run("CreateWeatherClient", func(t *testing.T) {
		status, err := makeRequest(http.MethodPost, "/weather_clients", fmt.Sprintf(`{
			"type": "fake",
			"options": {
				"avg_high_temperature": %f,
				"rain_interval": "%s",
				"rain_mm": %f
			}
		}`, opts.AverageHighTemperature, opts.RainInterval, opts.RainMM), &wcr)
		assert.NoError(t, err)

		assert.Equal(t, http.StatusCreated, status)
	})

	return wcr.ID.ID
}

func ZoneTests(t *testing.T) {
	gardenID := CreateGardenTest(t)
	waterScheduleID := CreateWaterScheduleTest(t)
	zoneID := CreateZoneTest(t, gardenID, waterScheduleID)

	t.Run("ExecuteWaterAction", func(t *testing.T) {
		status, err := makeRequest(
			http.MethodPost,
			fmt.Sprintf("/gardens/%s/zones/%s/action", gardenID, zoneID),
			action.ZoneAction{Water: &action.WaterAction{
				Duration: &pkg.Duration{Duration: time.Second * 3},
			}},
			&struct{}{},
		)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, status)

		time.Sleep(100 * time.Millisecond)

		id, err := xid.FromString(zoneID)
		assert.NoError(t, err)
		c.AssertWaterActions(t, action.WaterMessage{
			Duration: 3000,
			ZoneID:   id.String(),
			Position: 0,
		})
	})
	t.Run("CheckWateringHistory", func(t *testing.T) {
		// This test needs a few repeats to get a reliable pass, which is fine
		retries := 0

		var history server.ZoneWaterHistoryResponse
		for retries < 10 && history.Count < 1 {
			time.Sleep(300 * time.Millisecond)

			status, err := makeRequest(
				http.MethodGet,
				fmt.Sprintf("/gardens/%s/zones/%s/history", gardenID, zoneID),
				http.NoBody,
				&history,
			)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, status)
		}

		assert.Equal(t, 1, history.Count)
		assert.Equal(t, "3s", history.Average)
		assert.Equal(t, "3s", history.Total)
	})
	t.Run("ChangeWaterScheduleStartTimeResetsWaterSchedule", func(t *testing.T) {
		// Reschedule to Water in 2 second, for 1 second
		newStartTime := clock.Now().Add(2 * time.Second).Truncate(time.Second)
		var ws server.WaterScheduleResponse
		status, err := makeRequest(http.MethodPatch, "/water_schedules/"+waterScheduleID, pkg.WaterSchedule{
			StartTime: pkg.NewStartTime(newStartTime),
			Duration:  &pkg.Duration{Duration: time.Second},
		}, &ws)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, pkg.NewStartTime(newStartTime).String(), ws.WaterSchedule.StartTime.String())

		time.Sleep(100 * time.Millisecond)

		// Make sure NextWater is changed
		var z2 server.ZoneResponse
		status, err = makeRequest(http.MethodGet, fmt.Sprintf("/gardens/%s/zones/%s", gardenID, zoneID), http.NoBody, &z2)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, newStartTime, z2.NextWater.Time.Truncate(time.Second).Local())

		time.Sleep(3 * time.Second)

		// Assert WaterAction
		id, err := xid.FromString(zoneID)
		assert.NoError(t, err)
		c.AssertWaterActions(t,
			action.WaterMessage{
				Duration: 1000,
				ZoneID:   id.String(),
				Position: 0,
			},
		)
	})
}

func pointer[T any](v T) *T {
	return &v
}

func WaterScheduleTests(t *testing.T) {
	gardenID := CreateGardenTest(t)
	waterScheduleID := CreateWaterScheduleTest(t)
	_ = CreateZoneTest(t, gardenID, waterScheduleID)

	t.Run("ChangeRainControlResultsInCorrectScalingForNextAction", func(t *testing.T) {
		// Create WeatherClient with rain control
		weatherClientWithRain := CreateWeatherClientTest(t, fake.Config{
			RainInterval: "24h",
			RainMM:       25.4,
		})

		// Reschedule to Water in 2 second, for 1 second
		newStartTime := clock.Now().Add(2 * time.Second).Truncate(time.Second)
		var ws server.WaterScheduleResponse
		status, err := makeRequest(http.MethodPatch, "/water_schedules/"+waterScheduleID, pkg.WaterSchedule{
			StartTime: pkg.NewStartTime(newStartTime),
			Duration:  &pkg.Duration{Duration: time.Second},
		}, &ws)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, pkg.NewStartTime(newStartTime).String(), ws.WaterSchedule.StartTime.String())

		time.Sleep(100 * time.Millisecond)

		// Set WaterSchedule to use weather client with rain delay
		var ws2 server.WaterScheduleResponse
		status, err = makeRequest(http.MethodPatch, "/water_schedules/"+waterScheduleID, pkg.WaterSchedule{
			WeatherControl: &weather.Control{
				Rain: &weather.ScaleControl{
					BaselineValue: pointer[float32](0),
					Factor:        pointer[float32](0),
					Range:         pointer[float32](25.4),
					ClientID:      weatherClientWithRain,
				},
			},
		}, &ws2)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)

		// Make sure NextWater.Duration is now 0
		var ws3 server.WaterScheduleResponse
		status, err = makeRequest(http.MethodGet, fmt.Sprintf("/water_schedules/%s", waterScheduleID), http.NoBody, &ws3)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "0s", ws3.NextWater.Duration.String())

		time.Sleep(3 * time.Second)

		// Assert that no watering occurred because the rain should result in a skip
		assert.NoError(t, err)
		c.AssertWaterActions(t)
	})
}

func ControllerStartupNotificationTest(t *testing.T) {
	var g server.GardenResponse
	t.Run("CreateGarden", func(t *testing.T) {
		status, err := makeRequest(http.MethodPost, "/gardens", `{
				"name": "Notification",
				"topic_prefix": "notification",
				"max_zones": 3
			}`, &g)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, status)
	})

	var nc notifications.Client
	t.Run("CreateNotificationClient", func(t *testing.T) {
		status, err := makeRequest(http.MethodPost, "/notification_clients", `{
				"name": "fake client",
				"type": "fake",
				"options": {}
			}`, &nc)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, status)
	})

	t.Run("EnableNotificationsForGarden", func(t *testing.T) {
		status, err := makeRequest(http.MethodPatch, "/gardens/"+g.GetID(), pkg.Garden{
			NotificationClientID: pointer(nc.GetID()),
		}, &g)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("PublishStartupLogAndCheckNotification", func(t *testing.T) {
		err := c.PublishStartupLog(g.TopicPrefix)
		require.NoError(t, err)

		time.Sleep(500 * time.Millisecond)

		lastMsg := fake_notification.LastMessage()
		require.Equal(t, "Notification connected", lastMsg.Title)
		require.Equal(t, "garden-controller setup complete", lastMsg.Message)
	})
}

func makeRequest(method, path string, body, response interface{}) (int, error) {
	// TODO: Use babyapi Client
	var reqBody io.Reader
	switch v := body.(type) {
	case nil:
	case string:
		reqBody = bytes.NewBuffer([]byte(v))
	case io.Reader:
		reqBody = v
	default:
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, "http://localhost:8080"+path, reqBody)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	err = json.Unmarshal(data, response)
	if err != nil {
		return 0, err
	}
	return resp.StatusCode, nil
}
