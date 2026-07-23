//go:build manual

package integrationtests

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/controller"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/server"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// TestSeedManualData starts the integration-test stack (InfluxDB/MQTT), creates a Garden,
// WaterSchedule, and Zone, triggers a water action, and waits for the data to appear in
// both the Zone and Garden water-history endpoints. After the test exits it leaves
// manual_test.db populated so the server can be started manually. The manual docker-compose
// override uses ephemeral InfluxDB storage, so logs/metrics are reseeded each time.
func TestSeedManualData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping testing in short mode")
	}

	viper.SetConfigFile("testdata/manual_config.yml")
	err := viper.ReadInConfig()
	require.NoError(t, err)

	var serverConfig server.Config
	err = viper.Unmarshal(&serverConfig)
	require.NoError(t, err)

	// The seed test runs from ./integration_tests, but the server is started from ./garden-app
	// via the Taskfile. Use an absolute path so both contexts point to the same file.
	dbPath, err := filepath.Abs("../manual_test.db")
	require.NoError(t, err)

	if _, statErr := os.Stat(dbPath); statErr == nil {
		t.Skipf("manual test DB already exists at %s; skipping seeding", dbPath)
	}

	serverConfig.StorageConfig.ConnectionString = dbPath

	var controllerConfig controller.Config
	err = viper.Unmarshal(&controllerConfig)
	require.NoError(t, err)

	api := server.NewAPI()
	err = api.Setup(serverConfig, true)
	require.NoError(t, err)

	seedController, err := controller.NewController(controllerConfig)
	require.NoError(t, err)

	go seedController.Start()
	go func() {
		serveErr := api.SetAddress(":8080").Serve()
		if serveErr != nil {
			panic(serveErr.Error())
		}
	}()

	defer seedController.Stop()
	defer api.Stop()

	time.Sleep(500 * time.Millisecond)

	gardenID := CreateGardenTest(t)
	waterScheduleID := CreateWaterScheduleTest(t)
	zoneID := CreateZoneTest(t, gardenID, waterScheduleID)

	t.Run("WaterActionAndWaterHistory", func(t *testing.T) {
		duration := &pkg.Duration{Duration: 3 * time.Second}
		status, err := makeRequest(
			http.MethodPost,
			fmt.Sprintf("/gardens/%s/zones/%s/action", gardenID, zoneID),
			action.ZoneAction{Water: &action.WaterAction{Duration: duration}},
			&struct{}{},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, status)

		// Wait for the watering to complete and appear in the garden-wide history endpoint.
		var gardenHistory server.GardenWaterHistoryResponse
		require.Eventually(t, func() bool {
			status, err = makeRequest(
				http.MethodGet,
				fmt.Sprintf("/gardens/%s/water_history", gardenID),
				http.NoBody,
				&gardenHistory,
			)
			return err == nil && status == http.StatusOK && gardenHistory.Count >= 1
		}, 30*time.Second, 300*time.Millisecond, "garden water history did not populate")

		require.GreaterOrEqual(t, len(gardenHistory.History), 1)
		require.NotEmpty(t, gardenHistory.History[0].ZoneID, "history should include zone_id")
	})

	t.Run("ControllerLogs", func(t *testing.T) {
		// Publish a few controller log entries so the controller-logs endpoint has data.
		logs := [][]byte{
			[]byte(`logs,level=error,source=wifi_manager message="failed to connect to WiFi",device="esp32"`),
			[]byte(`logs,level=info,source=startup message="garden-controller setup complete",reset_reason="Reset due to power-on event."`),
			[]byte(`logs,level=warn,source=main message="low memory",free_heap=1024`),
		}
		for _, logMsg := range logs {
			err = seedController.PublishLog("test", logMsg)
			require.NoError(t, err)
		}

		expectedLogs := []pkg.ControllerLog{
			{
				Level:   "error",
				Source:  "wifi_manager",
				Message: "failed to connect to WiFi",
				Details: map[string]string{"device": "esp32"},
			},
			{
				Level:   "info",
				Source:  "startup",
				Message: "garden-controller setup complete",
				Details: map[string]string{"reset_reason": "Reset due to power-on event."},
			},
			{
				Level:   "warn",
				Source:  "main",
				Message: "low memory",
				Details: map[string]string{"free_heap": "1024"},
			},
		}

		matchesExpectedLog := func(actual, expected pkg.ControllerLog) bool {
			if actual.Level != expected.Level || actual.Source != expected.Source || actual.Message != expected.Message {
				return false
			}
			for k, v := range expected.Details {
				if actual.Details[k] != v {
					return false
				}
			}
			return true
		}

		// Wait for the controller logs to appear in InfluxDB.
		var controllerLogs server.ControllerLogsResponse
		require.Eventually(t, func() bool {
			status, err := makeRequest(
				http.MethodGet,
				fmt.Sprintf("/gardens/%s/controller-logs", gardenID),
				http.NoBody,
				&controllerLogs,
			)
			slices.SortFunc(controllerLogs.Logs, func(a, b pkg.ControllerLog) int {
				return strings.Compare(a.Message, b.Message)
			})
			return err == nil && status == http.StatusOK && slices.EqualFunc(controllerLogs.Logs, expectedLogs, matchesExpectedLog)
		}, 30*time.Second, 300*time.Millisecond, "controller logs did not populate")

		for _, expected := range expectedLogs {
			require.Condition(t, func() bool {
				for _, log := range controllerLogs.Logs {
					if matchesExpectedLog(log, expected) {
						return true
					}
				}
				return false
			})
		}
	})

	t.Run("ControllerInfo", func(t *testing.T) {
		err = seedController.PublishControllerInfo()
		require.NoError(t, err)

		var gardenWithInfo server.GardenResponse
		require.Eventually(t, func() bool {
			status, err := makeRequest(
				http.MethodGet,
				fmt.Sprintf("/gardens/%s", gardenID),
				http.NoBody,
				&gardenWithInfo,
			)
			if err != nil || status != http.StatusOK {
				return false
			}
			return gardenWithInfo.ControllerInfo != nil && gardenWithInfo.ControllerInfo.MACAddress != ""
		}, 30*time.Second, 300*time.Millisecond, "controller info did not populate")

		require.Equal(t, "AA:BB:CC:DD:EE:FF", gardenWithInfo.ControllerInfo.MACAddress)
		require.Equal(t, "192.168.1.42", gardenWithInfo.ControllerInfo.IPAddress)
		require.Equal(t, "test-version", gardenWithInfo.ControllerInfo.FirmwareVersion)
		require.NotNil(t, gardenWithInfo.ControllerInfo.UpdatedAt)
	})

	t.Run("SensorData", func(t *testing.T) {
		// Wait for sensor data from both configured sensors to appear in the garden response.
		var garden server.GardenResponse
		require.Eventually(t, func() bool {
			status, err := makeRequest(
				http.MethodGet,
				fmt.Sprintf("/gardens/%s", gardenID),
				http.NoBody,
				&garden,
			)
			if err != nil || status != http.StatusOK {
				return false
			}
			if len(garden.SensorsData) < 2 {
				return false
			}
			for _, s := range garden.SensorsData {
				if s.TemperatureCelsius == 0 {
					return false
				}
			}
			return true
		}, 30*time.Second, 300*time.Millisecond, "sensor data did not populate")

		require.Equal(t, "Water", garden.SensorsData[1].Name)
		require.Equal(t, "DS18B20", garden.SensorsData[1].Type)
		require.NotZero(t, garden.SensorsData[1].TemperatureCelsius)
		require.Zero(t, garden.SensorsData[1].HumidityPercentage)

		require.Len(t, garden.SensorsData, 2)
		require.Equal(t, "Ambient", garden.SensorsData[0].Name)
		require.Equal(t, "DHT22", garden.SensorsData[0].Type)
		require.NotZero(t, garden.SensorsData[0].TemperatureCelsius)
		require.NotZero(t, garden.SensorsData[0].HumidityPercentage)
	})
}
