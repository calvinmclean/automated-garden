package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

func TestPlantRequest(t *testing.T) {
	tests := []struct {
		name string
		pr   *PlantRequest
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required Plant fields",
		},
		{
			"EmptyPlantError",
			&PlantRequest{},
			"missing required Plant fields",
		},
		{
			"EmptyNameError",
			&PlantRequest{Plant: &pkg.Plant{}},
			"missing required name field",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		pr := &PlantRequest{
			Plant: &pkg.Plant{
				Name: "plant",
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := pr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading PlantRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.pr.Bind(r)
			if err == nil {
				t.Error("Expected error reading PlantRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}

func TestUpdatePlantRequest(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		pr   *UpdatePlantRequest
		err  string
	}{
		{
			"EmptyRequest",
			nil,
			"missing required Plant fields",
		},
		{
			"EmptyPlantError",
			&UpdatePlantRequest{},
			"missing required Plant fields",
		},
		{
			"ManualSpecificationOfIDError",
			&UpdatePlantRequest{
				Plant: &pkg.Plant{ID: xid.New()},
			},
			"updating ID is not allowed",
		},
		{
			"EndDateError",
			&UpdatePlantRequest{
				Plant: &pkg.Plant{
					EndDate: &now,
				},
			},
			"to end-date a Plant, please use the DELETE endpoint",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		pr := &UpdatePlantRequest{
			Plant: &pkg.Plant{
				Name: "plant",
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := pr.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading PlantRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.pr.Bind(r)
			if err == nil {
				t.Error("Expected error reading PlantRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}
