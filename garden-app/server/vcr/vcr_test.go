package vcr_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/server"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
)

func TestReplay(t *testing.T) {
	dir := "testdata/vcr_server/fixtures"
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	t.Cleanup(server.DisableMock)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		cassetteName := strings.TrimSuffix(entry.Name(), ".yaml")
		t.Run(cassetteName, func(t *testing.T) {
			api := server.NewAPI()
			server.EnableMock()

			var config server.Config
			t.Run("SetupConfig", func(t *testing.T) {
				viper.SetConfigFile("./testdata/vcr_server/config.yaml")

				err := viper.ReadInConfig()
				require.NoError(t, err)

				err = viper.Unmarshal(&config)
				require.NoError(t, err)
			})

			err := api.Setup(config, true)
			require.NoError(t, err)

			r, err := api.Router()
			require.NoError(t, err)

			cassette.TestServerReplay(t, filepath.Join(dir, cassetteName), r)
		})
	}
}
