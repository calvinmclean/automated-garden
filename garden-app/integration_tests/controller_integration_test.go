//go:build controller_integration

package integrationtests

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/server"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

const (
	controllerIntegrationConfigFile = "testdata/controller_integration_config.yml"
	controllerIntegrationTimeout    = 90 * time.Second
)

// TestControllerIntegration exercises a physical ESP32 controller. Configure its
// WiFiManager WiFi connection and topic prefix before running this test; see README.md.
func TestControllerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping controller integration test in short mode")
	}

	serverConfig := controllerIntegrationServerConfig(t)
	api := server.NewAPI()
	require.NoError(t, api.Setup(serverConfig, true))
	t.Cleanup(api.Stop)

	go func() {
		if err := api.SetAddress(":8080").Serve(); err != nil {
			panic(err)
		}
	}()

	require.Eventually(t, func() bool {
		status, err := makeRequest(http.MethodGet, "/gardens", http.NoBody, &struct{}{})
		return err == nil && status == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "garden-app did not start")

	topicPrefix := os.Getenv("CONTROLLER_INTEGRATION_TOPIC_PREFIX")
	if topicPrefix == "" {
		topicPrefix = "controller-integration"
	}
	mqttServer := controllerIntegrationMQTTServer(t)
	controllerAddress := os.Getenv("CONTROLLER_INTEGRATION_CONTROLLER_IP")

	var gardenID, zoneID string
	var controllerInfoUpdatedAt time.Time
	t.Run("CreateGardenAndZone", func(t *testing.T) {
		gardenID = createControllerIntegrationGarden(t, topicPrefix)
		waterScheduleID := CreateWaterScheduleTest(t)
		zoneID = CreateZoneTest(t, gardenID, waterScheduleID)
	})

	t.Run("SetupControllerMQTT", func(t *testing.T) {
		status, err := makeRequest(
			http.MethodPost,
			fmt.Sprintf("/gardens/%s/action", gardenID),
			action.GardenAction{ControllerSetup: &action.ControllerSetupAction{
				Server:            mqttServer,
				TopicPrefix:       topicPrefix,
				Port:              1883,
				ControllerAddress: controllerAddress,
			}},
			&struct{}{},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, status)
	})

	t.Run("WaitForControllerConnection", func(t *testing.T) {
		var garden server.GardenResponse
		require.Eventually(t, func() bool {
			status, err := makeRequest(http.MethodGet, "/gardens/"+gardenID, http.NoBody, &garden)
			return err == nil && status == http.StatusOK && garden.ControllerInfo != nil && garden.ControllerInfo.UpdatedAt != nil
		}, controllerIntegrationTimeout, 500*time.Millisecond,
			"controller did not connect after setup; verify WiFi, topic prefix, and the controller IP override")
		controllerInfoUpdatedAt = *garden.ControllerInfo.UpdatedAt
	})

	t.Run("UpdateControllerConfig", func(t *testing.T) {
		status, err := makeRequest(
			http.MethodPost,
			fmt.Sprintf("/gardens/%s/action", gardenID),
			action.GardenAction{Update: &action.UpdateAction{Config: true}},
			&struct{}{},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, status)
	})

	t.Run("WaitForControllerReconnect", func(t *testing.T) {
		var garden server.GardenResponse
		require.Eventually(t, func() bool {
			status, err := makeRequest(http.MethodGet, "/gardens/"+gardenID, http.NoBody, &garden)
			return err == nil && status == http.StatusOK && garden.ControllerInfo != nil && garden.ControllerInfo.UpdatedAt != nil && garden.ControllerInfo.UpdatedAt.After(controllerInfoUpdatedAt)
		}, controllerIntegrationTimeout, 500*time.Millisecond,
			"controller did not reconnect after the configuration update; verify its MQTT host and topic prefix")
	})

	t.Run("WaterActionAndHistory", func(t *testing.T) {
		status, err := makeRequest(
			http.MethodPost,
			fmt.Sprintf("/gardens/%s/zones/%s/action", gardenID, zoneID),
			action.ZoneAction{Water: &action.WaterAction{Duration: &pkg.Duration{Duration: time.Second}}},
			&struct{}{},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, status)

		history := waitForWaterHistory(t, gardenID, zoneID, pkg.WaterStatusCompleted)
		require.Len(t, history.History, 1)
		require.Equal(t, pkg.WaterStatusCompleted, history.History[0].Status)
	})

	t.Run("StopWaterAction", func(t *testing.T) {
		status, err := makeRequest(
			http.MethodPost,
			fmt.Sprintf("/gardens/%s/zones/%s/action", gardenID, zoneID),
			action.ZoneAction{Water: &action.WaterAction{Duration: &pkg.Duration{Duration: 30 * time.Second}}},
			&struct{}{},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, status)

		waitForWaterHistory(t, gardenID, zoneID, pkg.WaterStatusStarted)

		status, err = makeRequest(
			http.MethodPost,
			fmt.Sprintf("/gardens/%s/action", gardenID),
			action.GardenAction{Stop: &action.StopAction{}},
			&struct{}{},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, status)

		history := waitForWaterHistory(t, gardenID, zoneID, pkg.WaterStatusCancelled)
		require.Len(t, history.History, 2)
		require.Equal(t, pkg.WaterStatusCancelled, history.History[0].Status)
	})
}

func controllerIntegrationServerConfig(t *testing.T) server.Config {
	t.Helper()

	config := viper.New()
	config.SetConfigFile(controllerIntegrationConfigFile)
	require.NoError(t, config.ReadInConfig())

	var serverConfig server.Config
	require.NoError(t, config.Unmarshal(&serverConfig))
	return serverConfig
}

func controllerIntegrationMQTTServer(t *testing.T) string {
	t.Helper()

	if server := os.Getenv("CONTROLLER_INTEGRATION_MQTT_SERVER"); server != "" {
		return server
	}

	interfaces, err := net.Interfaces()
	require.NoError(t, err)
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		require.NoError(t, err)
		for _, address := range addresses {
			ipNet, ok := address.(*net.IPNet)
			if !ok {
				continue
			}
			if ip := ipNet.IP.To4(); ip != nil && !ip.IsLoopback() {
				return ip.String()
			}
		}
	}

	t.Fatal("unable to detect a LAN IPv4 address; set CONTROLLER_INTEGRATION_MQTT_SERVER")
	return ""
}

func createControllerIntegrationGarden(t *testing.T, topicPrefix string) string {
	t.Helper()

	var garden server.GardenResponse
	status, err := makeRequest(http.MethodPost, "/gardens", pkg.Garden{
		Name:        "Controller Integration Test",
		TopicPrefix: topicPrefix,
		MaxZones:    pointer(uint(4)),
		ControllerConfig: &pkg.ControllerConfig{
			ValvePins: []uint{16, 17, 5, 18},
			PumpPins:  []uint{16, 17, 5, 18},
			LightPin:  pointer(uint(32)),
			FanPin:    pointer(uint(23)),
			Sensors: []pkg.SensorConfig{
				{
					ID:       "14bc778bdefe4a708307",
					Name:     "Ambient",
					Type:     "DHT22",
					Pin:      4,
					Interval: pkg.Duration{Duration: time.Minute},
				},
				{
					ID:       "d93j94djtb6s73as8h20",
					Name:     "Aerogarden Water",
					Type:     "DS18B20",
					Pin:      19,
					Interval: pkg.Duration{Duration: time.Minute},
				},
			},
		},
	}, &garden)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, status)

	return garden.ID.String()
}

func waitForWaterHistory(t *testing.T, gardenID, zoneID string, expectedStatus pkg.WaterStatus) server.ZoneWaterHistoryResponse {
	t.Helper()

	var history server.ZoneWaterHistoryResponse
	require.Eventually(t, func() bool {
		status, err := makeRequest(
			http.MethodGet,
			fmt.Sprintf("/gardens/%s/zones/%s/history", gardenID, zoneID),
			http.NoBody,
			&history,
		)
		return err == nil && status == http.StatusOK && len(history.History) > 0 && history.History[0].Status == expectedStatus
	}, controllerIntegrationTimeout, 300*time.Millisecond, "water history did not contain a %s event", expectedStatus)

	return history
}
