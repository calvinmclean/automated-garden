package storage

import (
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

func TestNewYAMLClient(t *testing.T) {
	gardenID, _ := xid.FromString("c22tmvucie6n6gdrpal0")
	tests := []struct {
		name             string
		config           Config
		expectedFilename string
		expectedGardens  map[xid.ID]*pkg.Garden
	}{
		{
			"FileNotExist",
			Config{
				Type:    "YAML",
				Options: map[string]string{"filename": "fake file"},
			},
			"fake file",
			map[xid.ID]*pkg.Garden{},
		},
		{
			"EmptyFile",
			Config{
				Type:    "YAML",
				Options: map[string]string{"filename": "testdata/gardens_empty.yaml"},
			},
			"testdata/gardens_empty.yaml",
			map[xid.ID]*pkg.Garden{},
		},
		{
			"RealFile",
			Config{
				Type:    "YAML",
				Options: map[string]string{"filename": "testdata/gardens_data.yaml"},
			},
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
			client, err := NewYAMLClient(tt.config)
			if err != nil {
				t.Errorf("Unexpected error from NewYAMLClient: %v", err)
			}
			if tt.expectedFilename != client.filename {
				t.Errorf("Unepected filename: expected=%v, actual=%v", tt.expectedFilename, client.filename)
			}
			if len(tt.expectedGardens) != len(client.gardens) {
				t.Errorf("Unexpected size of Gardens map: expected=%v, actual=%v", len(tt.expectedGardens), len(client.gardens))
			}
			for id, expectedGarden := range tt.expectedGardens {
				garden, ok := client.gardens[id]
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
		_, err := NewYAMLClient(Config{
			Type:    "YAML",
			Options: map[string]string{},
		})
		if err == nil {
			t.Error("Expected error but got nil")
		}
		if err.Error() != "missing config key 'filename'" {
			t.Errorf("Unexpected error message: %v", err)
		}

	})
}
