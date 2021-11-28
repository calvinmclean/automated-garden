package pkg

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/rs/xid"
	"github.com/stretchr/testify/mock"
)

func TestHealth(t *testing.T) {
	tests := []struct {
		name            string
		lastContactTime time.Time
		err             error
		expectedStatus  string
	}{
		{
			"GardenIsUp",
			time.Now(),
			nil,
			"UP",
		},
		{
			"GardenIsDown",
			time.Now().Add(-5 * time.Minute),
			nil,
			"DOWN",
		},
		{
			"InfluxDBError",
			time.Time{},
			errors.New("influxdb error"),
			"N/A",
		},
		{
			"ZeroTime",
			time.Time{},
			nil,
			"DOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "garden").Return(tt.lastContactTime, tt.err)

			g := Garden{Name: "garden"}

			gardenHealth := g.Health(context.Background(), influxdbClient)
			if gardenHealth.Status != tt.expectedStatus {
				t.Errorf("Unexpected GardenHealth.Status: expected = %s, actual = %s", tt.expectedStatus, gardenHealth.Status)
			}
		})
	}
}

func TestGardenEndDated(t *testing.T) {
	pastDate := time.Now().Add(-1 * time.Minute)
	futureDate := time.Now().Add(time.Minute)
	tests := []struct {
		name     string
		endDate  *time.Time
		expected bool
	}{
		{"NilEndDateFalse", nil, false},
		{"EndDateFutureEndDateFalse", &futureDate, false},
		{"EndDatePastEndDateTrue", &pastDate, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Garden{EndDate: tt.endDate}
			if g.EndDated() != tt.expected {
				t.Errorf("Unexpected result: expected=%v, actual=%v", tt.expected, g.EndDated())
			}
		})
	}
}

func TestGardenPatch(t *testing.T) {
	now := time.Now()
	ten := uint(10)
	tests := []struct {
		name      string
		newGarden *Garden
	}{
		{
			"PatchName",
			&Garden{Name: "name"},
		},
		{
			"PatchMaxPlants",
			&Garden{MaxPlants: &ten},
		},
		{
			"PatchCreatedAt",
			&Garden{CreatedAt: &now},
		},
		{
			"PatchLightSchedule.Duration",
			&Garden{LightSchedule: &LightSchedule{
				Duration: "2h",
			}},
		},
		{
			"PatchLightSchedule.StartTime",
			&Garden{LightSchedule: &LightSchedule{
				StartTime: "start time",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Garden{}
			g.Patch(tt.newGarden)
			if g.LightSchedule != nil && *g.LightSchedule != *tt.newGarden.LightSchedule {
				t.Errorf("Unexpected result for LightSchedule: expected=%v, actual=%v", tt.newGarden.LightSchedule, g.LightSchedule)
			}
			if g.Name != tt.newGarden.Name {
				t.Errorf("Unexpected result for Name: expected=%v, actual=%v", tt.newGarden.Name, g.Name)
			}
			if g.CreatedAt != tt.newGarden.CreatedAt {
				t.Errorf("Unexpected result for CreatedAt: expected=%v, actual=%v", tt.newGarden.CreatedAt, g.CreatedAt)
			}
		})
	}

	t.Run("RemoveLightSchedule", func(t *testing.T) {
		g := &Garden{
			LightSchedule: &LightSchedule{
				StartTime: "START TIME",
				Duration:  "2h",
			},
		}
		g.Patch(&Garden{LightSchedule: &LightSchedule{}})

		if g.LightSchedule != nil {
			t.Errorf("Expected nil LightSchedule, but got: %v", g.LightSchedule)
		}
	})

	t.Run("PatchDoesNotAddEndDate", func(t *testing.T) {
		now := time.Now()
		g := &Garden{}

		g.Patch(&Garden{EndDate: &now})

		if g.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", g.EndDate)
		}
	})

	t.Run("PatchRemoveEndDate", func(t *testing.T) {
		now := time.Now()
		g := &Garden{
			EndDate: &now,
		}

		g.Patch(&Garden{})

		if g.EndDate != nil {
			t.Errorf("Expected nil EndDate, but got: %v", g.EndDate)
		}
	})
}

func TestGardenNumPlants(t *testing.T) {
	endDate := time.Now().Add(-1 * time.Minute)
	tests := []struct {
		name     string
		garden   *Garden
		expected uint
	}{
		{
			"NoPlants",
			&Garden{},
			0,
		},
		{
			"NoActivePlants",
			&Garden{
				Plants: map[xid.ID]*Plant{
					xid.New(): {EndDate: &endDate},
				},
			},
			0,
		},
		{
			"NoEndDatedPlants",
			&Garden{
				Plants: map[xid.ID]*Plant{
					xid.New(): {},
				},
			},
			1,
		},
		{
			"EndDatedAndActivePlants",
			&Garden{
				Plants: map[xid.ID]*Plant{
					xid.New(): {EndDate: &endDate},
					xid.New(): {},
				},
			},
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.garden.NumPlants() != tt.expected {
				t.Errorf("Unexpected result: expected=%v, actual=%v", tt.expected, tt.garden.NumPlants())
			}
		})
	}
}
