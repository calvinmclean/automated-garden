package integrationtests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/controller"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/server"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	configFile      = "testdata/config.yml"
	baseGardensFile = "testdata/gardens_test.yml"
	gardensFile     = "testdata/gardens.yml"
)

var (
	c *controller.Controller
	s *server.Server
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testing in short mode")
	}

	input, err := os.ReadFile(baseGardensFile)
	require.NoError(t, err)

	err = os.WriteFile(gardensFile, input, 0644)
	require.NoError(t, err)

	defer os.RemoveAll(gardensFile)

	serverConfig, controllerConfig := getConfigs(t)

	s, err = server.NewServer(serverConfig)
	require.NoError(t, err)

	c, err = controller.NewController(controllerConfig)
	require.NoError(t, err)

	go c.Start()
	go s.Start()

	defer c.Stop()
	defer s.Stop()

	time.Sleep(500 * time.Millisecond)

	// Run Garden tests
	t.Run("Garden", GardenTests)

	// Run Zone tests
	t.Run("Zone", ZoneTests)
}

func getConfigs(t *testing.T) (server.Config, controller.Config) {
	viper.SetConfigFile(configFile)
	err := viper.ReadInConfig()
	require.NoError(t, err)

	var serverConfig server.Config
	err = viper.Unmarshal(&serverConfig)
	require.NoError(t, err)
	serverConfig.LogConfig.Level = logrus.DebugLevel.String()

	var controllerConfig controller.Config
	err = viper.Unmarshal(&controllerConfig)
	require.NoError(t, err)
	controllerConfig.LogConfig.Level = logrus.DebugLevel.String()

	return serverConfig, controllerConfig
}

func GardenTests(t *testing.T) {
	t.Run("GetGarden", func(t *testing.T) {
		var g server.GardenResponse
		status, err := makeRequest(http.MethodGet, "/gardens/c9i98glvqc7km2vasfig", nil, &g)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)

		assert.Equal(t, "c9i98glvqc7km2vasfig", g.ID.String())
	})
	t.Run("ExecuteStopAction", func(t *testing.T) {
		status, err := makeRequest(
			http.MethodPost,
			"/gardens/c9i98glvqc7km2vasfig/action",
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
			"/gardens/c9i98glvqc7km2vasfig/action",
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
				"/gardens/c9i98glvqc7km2vasfig/action",
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
		startTime := time.Now().In(time.Local).Add(1 * time.Second).Truncate(time.Second)
		newGarden := &server.GardenRequest{
			Garden: &pkg.Garden{
				Name:        "TestGarden",
				TopicPrefix: "test",
				MaxZones:    &maxZones,
				LightSchedule: &pkg.LightSchedule{
					Duration:  &pkg.Duration{Duration: 14 * time.Hour},
					StartTime: startTime.Format(pkg.LightTimeFormat),
				},
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
		status, err = makeRequest(http.MethodGet, fmt.Sprintf("/gardens/%s", g.ID.String()), nil, &getG)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, startTime.Add(1*time.Second), getG.NextLightAction.Time.Local())

		time.Sleep(3 * time.Second)

		// Check for light action turning it off, plus adhoc schedule to turn it back on
		c.AssertLightActions(t,
			action.LightAction{State: pkg.LightStateOff, ForDuration: &pkg.Duration{Duration: time.Second}},
			action.LightAction{State: pkg.LightStateOn},
		)
	})
	t.Run("ChangeLightScheduleStartTimeResetsLightSchedule", func(t *testing.T) {
		// Reschedule Light to turn in in 1 second, for 1 second
		newStartTime := time.Now().Add(1 * time.Second).Truncate(time.Second)
		var g server.GardenResponse
		status, err := makeRequest(http.MethodPatch, "/gardens/c9i98glvqc7km2vasfig", pkg.Garden{
			LightSchedule: &pkg.LightSchedule{
				StartTime: newStartTime.Format(pkg.LightTimeFormat),
				Duration:  &pkg.Duration{Duration: time.Second},
			},
		}, &g)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, newStartTime.Format(pkg.LightTimeFormat), g.LightSchedule.StartTime)

		time.Sleep(100 * time.Millisecond)

		// Make sure NextOnTime and state are changed
		var g2 server.GardenResponse
		status, err = makeRequest(http.MethodGet, "/gardens/c9i98glvqc7km2vasfig", nil, &g2)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, newStartTime, g2.NextLightAction.Time.Truncate(time.Second).Local())
		assert.Equal(t, pkg.LightStateOn, g2.NextLightAction.State)

		time.Sleep(2 * time.Second)

		// Assert both LightActions
		c.AssertLightActions(t,
			action.LightAction{State: pkg.LightStateOn},
			action.LightAction{State: pkg.LightStateOff},
		)
	})
}

func ZoneTests(t *testing.T) {
	t.Run("ExecuteWaterAction", func(t *testing.T) {
		status, err := makeRequest(
			http.MethodPost,
			"/gardens/c9i98glvqc7km2vasfig/zones/c9i99otvqc7kmt8hjio0/action",
			action.ZoneAction{Water: &action.WaterAction{
				Duration: &pkg.Duration{Duration: time.Second * 3},
			}},
			&struct{}{},
		)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, status)

		time.Sleep(100 * time.Millisecond)

		id, err := xid.FromString("c9i99otvqc7kmt8hjio0")
		assert.NoError(t, err)
		c.AssertWaterActions(t, action.WaterMessage{
			Duration: 3000,
			ZoneID:   id,
			Position: 0,
		})
	})
	t.Run("CheckWateringHistory", func(t *testing.T) {
		// This test needs a few repeats to get a reliable pass, which is fine
		retries := 0

		var history server.ZoneWaterHistoryResponse
		for retries < 10 && history.Count < 1 {
			time.Sleep(300 * time.Millisecond)

			status, err := makeRequest(http.MethodGet, "/gardens/c9i98glvqc7km2vasfig/zones/c9i99otvqc7kmt8hjio0/history", nil, &history)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, status)
		}

		assert.Equal(t, 1, history.Count)
		assert.Equal(t, "3s", history.Average)
		assert.Equal(t, "3s", history.Total)
	})
	t.Run("ChangeWaterScheduleStartTimeResetsWaterSchedule", func(t *testing.T) {
		// Reschedule to Water in 1 second, for 1 second
		newStartTime := time.Now().Add(1 * time.Second).Truncate(time.Second)
		var z server.ZoneResponse
		status, err := makeRequest(http.MethodPatch, "/gardens/c9i98glvqc7km2vasfig/zones/c9i99otvqc7kmt8hjio0", pkg.Zone{
			WaterSchedule: &pkg.WaterSchedule{
				StartTime: &newStartTime,
				Duration:  &pkg.Duration{Duration: time.Second},
			},
		}, &z)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, newStartTime, z.WaterSchedule.StartTime.Local())

		time.Sleep(100 * time.Millisecond)

		// Make sure NextWater is changed
		var z2 server.ZoneResponse
		status, err = makeRequest(http.MethodGet, "/gardens/c9i98glvqc7km2vasfig/zones/c9i99otvqc7kmt8hjio0", nil, &z2)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, newStartTime, z2.NextWaterTime.Truncate(time.Second).Local())

		time.Sleep(2 * time.Second)

		// Assert WaterAction
		id, err := xid.FromString("c9i99otvqc7kmt8hjio0")
		assert.NoError(t, err)
		c.AssertWaterActions(t,
			action.WaterMessage{
				Duration: 1000,
				ZoneID:   id,
				Position: 0,
			},
		)
	})
}

func makeRequest(method, path string, body, response interface{}) (int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = http.NoBody
	}

	req, err := http.NewRequest(method, "http://localhost:8080"+path, reqBody)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

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
