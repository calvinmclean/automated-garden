//go:build manual

package integrationtests

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
// manual_test.db and the InfluxDB volume populated so the server can be started manually.
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

	t.Logf("Seeded garden %s with %d water history event(s)", gardenID, len(gardenHistory.History))
}
