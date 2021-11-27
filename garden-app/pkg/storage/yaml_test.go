package storage

import (
	"io/ioutil"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
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

func resetFile(filename string, gardens map[xid.ID]*pkg.Garden) error {
	content, err := yaml.Marshal(gardens)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, content, 0755)
}

func copyMap(gardens map[xid.ID]*pkg.Garden) map[xid.ID]*pkg.Garden {
	result := map[xid.ID]*pkg.Garden{}
	for k, v := range gardens {
		result[k] = v
	}
	return result
}

func TestNewYAMLClient(t *testing.T) {
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

func TestGetGarden(t *testing.T) {
	client, err := NewYAMLClient(Config{
		Type:    "YAML",
		Options: map[string]string{"filename": "testdata/gardens_data.yaml"},
	})
	if err != nil {
		t.Errorf("Unexpected error from NewYAMLClient: %v", err)
	}

	t.Run("Successful", func(t *testing.T) {
		garden, err := client.GetGarden(gardenID)
		if err != nil {
			t.Errorf("Unexpected error from GetGarden: %v", err)
		}
		if garden == nil {
			t.Error("Expected Garden but got nil")
		}
		if garden != nil && gardenID != garden.ID {
			t.Errorf("Unexpected Garden IDs: expected=%v, actual=%v", gardenID, garden.ID)
		}
		if garden.Name != "test-garden" {
			t.Errorf("Unexpected Garden Names: expected=%v, actual=%v", "test-garden", garden.Name)
		}
	})
	t.Run("GardenNotFound", func(t *testing.T) {
		garden, err := client.GetGarden(endDatedGardenID)
		if err != nil {
			t.Errorf("Unexpected error from GetGarden: %v", err)
		}
		if garden != nil {
			t.Errorf("Expected nil, but got: %v", garden)
		}
	})
}

func TestGetGardens(t *testing.T) {
	client, err := NewYAMLClient(Config{
		Type:    "YAML",
		Options: map[string]string{"filename": "testdata/gardens_end_dated.yaml"},
	})
	if err != nil {
		t.Errorf("Unexpected error from NewYAMLClient: %v", err)
	}

	t.Run("SuccessfulNoEndDated", func(t *testing.T) {
		gardens, err := client.GetGardens(false)
		if err != nil {
			t.Errorf("Unexpected error from GetGardens: %v", err)
		}
		if len(gardens) != 1 {
			t.Errorf("Expected 1 Garden but got: %v", len(gardens))
		}
		if gardens[0] != nil && gardenID != gardens[0].ID {
			t.Errorf("Unexpected Garden IDs: expected=%v, actual=%v", gardenID, gardens[0].ID)
		}
		if gardens[0].Name != "test-garden" {
			t.Errorf("Unexpected Garden Names: expected=%v, actual=%v", "test-garden", gardens[0].Name)
		}
	})
	t.Run("SuccessfulEndDated", func(t *testing.T) {
		gardens, err := client.GetGardens(true)
		if err != nil {
			t.Errorf("Unexpected error from GetGardens: %v", err)
		}
		if len(gardens) != 2 {
			t.Errorf("Expected 2 Gardens but got: %v", len(gardens))
		}
	})
}

func TestSaveGarden(t *testing.T) {
	client, err := NewYAMLClient(Config{
		Type:    "YAML",
		Options: map[string]string{"filename": "testdata/gardens_data.yaml"},
	})
	if err != nil {
		t.Errorf("Unexpected error from NewYAMLClient: %v", err)
	}

	backup := copyMap(client.gardens)

	t.Run("Successful", func(t *testing.T) {
		defer client.update()
		defer resetFile("testdata/gardens_data.yaml", backup)

		newGarden := &pkg.Garden{
			ID:   xid.New(),
			Name: "NEW GARDEN",
		}

		err := client.SaveGarden(newGarden)
		if err != nil {
			t.Errorf("Unexpected error from SaveGarden: %v", err)
		}
		if len(client.gardens) != 2 {
			t.Errorf("Expected 2 Gardens but found: %v", len(client.gardens))
		}
		g, ok := client.gardens[newGarden.ID]
		if !ok {
			t.Error("Unable to find newly-added Garden")
		}
		if g.Name != "NEW GARDEN" {
			t.Errorf("Unexpected name for newly-added Garden: %v", g.Name)
		}
	})
}

func TestDeleteGarden(t *testing.T) {
	client, err := NewYAMLClient(Config{
		Type:    "YAML",
		Options: map[string]string{"filename": "testdata/gardens_data.yaml"},
	})
	if err != nil {
		t.Errorf("Unexpected error from NewYAMLClient: %v", err)
	}

	backup := copyMap(client.gardens)

	t.Run("Successful", func(t *testing.T) {
		defer client.update()
		defer resetFile("testdata/gardens_data.yaml", backup)

		err := client.DeleteGarden(gardenID)
		if err != nil {
			t.Errorf("Unexpected error from GetGardens: %v", err)
		}
		if len(client.gardens) != 0 {
			t.Errorf("Expected empty Gardens, but found: %v", len(client.gardens))
		}
	})

	t.Run("SuccessfulGardenNotExist", func(t *testing.T) {
		defer client.update()
		defer resetFile("testdata/gardens_data.yaml", backup)

		err := client.DeleteGarden(endDatedGardenID)
		if err != nil {
			t.Errorf("Unexpected error from GetGardens: %v", err)
		}
		if len(client.gardens) != 1 {
			t.Errorf("Expected 1 Gardens, but found: %v", len(client.gardens))
		}
	})
}
