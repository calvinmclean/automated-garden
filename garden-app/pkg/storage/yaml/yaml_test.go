package yaml

import (
	"os"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
	"gopkg.in/yaml.v3"
)

var (
	gardenID         xid.ID
	endDatedGardenID xid.ID
	plantID          xid.ID
)

func init() {
	gardenID, _ = xid.FromString("c22tmvucie6n6gdrpal0")
	endDatedGardenID, _ = xid.FromString("c6gknrvphd1d7b5nfck0")
	plantID, _ = xid.FromString("c3ucvu06n88pt1dom670")
}

func resetFile(filename string, data clientData) error {
	content, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, content, 0755)
}

func copyData(data clientData) clientData {
	gardens := map[xid.ID]*pkg.Garden{}
	for k, v := range data.Gardens {
		gardens[k] = v
	}
	weatherClients := map[xid.ID]*weather.Config{}
	for k, v := range data.WeatherClientConfigs {
		weatherClients[k] = v
	}
	return clientData{Gardens: gardens, WeatherClientConfigs: weatherClients}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name             string
		options          map[string]interface{}
		expectedFilename string
		expectedGardens  map[xid.ID]*pkg.Garden
	}{
		{
			"FileNotExist",
			map[string]interface{}{"filename": "fake file"},
			"fake file",
			map[xid.ID]*pkg.Garden{},
		},
		{
			"EmptyFile",
			map[string]interface{}{"filename": "testdata/gardens_empty.yaml"},
			"testdata/gardens_empty.yaml",
			map[xid.ID]*pkg.Garden{},
		},
		{
			"RealFile",
			map[string]interface{}{"filename": "testdata/gardens_data.yaml"},
			"testdata/gardens_data.yaml",
			map[xid.ID]*pkg.Garden{
				gardenID: {
					Name: "test-garden",
					ID:   gardenID,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient("yaml", tt.options)
			if err != nil {
				t.Errorf("Unexpected error from NewClient: %v", err)
			}
			if tt.expectedFilename != client.filename {
				t.Errorf("Unepected filename: expected=%v, actual=%v", tt.expectedFilename, client.filename)
			}
			if len(tt.expectedGardens) != len(client.data.Gardens) {
				t.Errorf("Unexpected size of Gardens map: expected=%v, actual=%v", len(tt.expectedGardens), len(client.data.Gardens))
			}
			for id, expectedGarden := range tt.expectedGardens {
				garden, ok := client.data.Gardens[id]
				if !ok {
					t.Errorf("Expected Garden does not exist. ID=%v", id)
				}
				if expectedGarden.ID != garden.ID {
					t.Errorf("Unexpected Garden IDs: expected=%v, actual=%v", expectedGarden.ID, garden.ID)
				}
				if expectedGarden.Name != garden.Name {
					t.Errorf("Unexpected Garden Names: expected=%v, actual=%v", expectedGarden.Name, garden.Name)
				}
			}
		})
	}

	t.Run("ErrorMissingFilename", func(t *testing.T) {
		_, err := NewClient("yaml", map[string]interface{}{})
		if err == nil {
			t.Error("Expected error but got nil")
		}
		if err.Error() != "missing config key 'filename'" {
			t.Errorf("Unexpected error message: %v", err)
		}

	})
}
