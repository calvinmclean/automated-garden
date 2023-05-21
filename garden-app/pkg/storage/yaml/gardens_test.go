package yaml

import (
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

func TestGetGarden(t *testing.T) {
	client, err := NewClient("yaml", map[string]string{"filename": "testdata/gardens_data.yaml"})
	if err != nil {
		t.Errorf("Unexpected error from NewClient: %v", err)
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
	client, err := NewClient("yaml", map[string]string{"filename": "testdata/gardens_end_dated.yaml"})
	if err != nil {
		t.Errorf("Unexpected error from NewClient: %v", err)
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
	client, err := NewClient("yaml", map[string]string{"filename": "testdata/gardens_data.yaml"})
	if err != nil {
		t.Errorf("Unexpected error from NewClient: %v", err)
	}

	backup := copyData(client.data)

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
		if len(client.data.Gardens) != 2 {
			t.Errorf("Expected 2 Gardens but found: %v", len(client.data.Gardens))
		}
		g, ok := client.data.Gardens[newGarden.ID]
		if !ok {
			t.Error("Unable to find newly-added Garden")
		}
		if g.Name != "NEW GARDEN" {
			t.Errorf("Unexpected name for newly-added Garden: %v", g.Name)
		}
	})
}

func TestDeleteGarden(t *testing.T) {
	client, err := NewClient("yaml", map[string]string{"filename": "testdata/gardens_data.yaml"})
	if err != nil {
		t.Errorf("Unexpected error from NewClient: %v", err)
	}

	backup := copyData(client.data)

	t.Run("Successful", func(t *testing.T) {
		defer client.update()
		defer resetFile("testdata/gardens_data.yaml", backup)

		err := client.DeleteGarden(gardenID)
		if err != nil {
			t.Errorf("Unexpected error from GetGardens: %v", err)
		}
		if len(client.data.Gardens) != 0 {
			t.Errorf("Expected empty Gardens, but found: %v", len(client.data.Gardens))
		}
	})

	t.Run("SuccessfulGardenNotExist", func(t *testing.T) {
		defer client.update()
		defer resetFile("testdata/gardens_data.yaml", backup)

		err := client.DeleteGarden(endDatedGardenID)
		if err != nil {
			t.Errorf("Unexpected error from GetGardens: %v", err)
		}
		if len(client.data.Gardens) != 1 {
			t.Errorf("Expected 1 Gardens, but found: %v", len(client.data.Gardens))
		}
	})
}
