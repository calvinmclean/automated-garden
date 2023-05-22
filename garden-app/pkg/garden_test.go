package pkg

import (
	"context"
	"encoding/json"
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

			g := Garden{TopicPrefix: "garden"}

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
			"PatchTopicPrefix",
			&Garden{TopicPrefix: "topic"},
		},
		{
			"PatchMaxPlants",
			&Garden{MaxZones: &ten},
		},
		{
			"PatchCreatedAt",
			&Garden{CreatedAt: &now},
		},
		{
			"PatchLightSchedule.Duration",
			&Garden{LightSchedule: &LightSchedule{
				Duration: &Duration{2 * time.Hour, ""},
			}},
		},
		{
			"PatchLightSchedule.StartTime",
			&Garden{LightSchedule: &LightSchedule{
				StartTime: "start time",
			}},
		},
		{
			"PatchLightSchedule.AdhocOnTime",
			&Garden{LightSchedule: &LightSchedule{
				AdhocOnTime: nil,
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
				Duration:  &Duration{2 * time.Hour, ""},
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

func TestLightStateString(t *testing.T) {
	tests := []struct {
		name     string
		input    LightState
		expected string
	}{
		{
			"ON",
			LightStateOn,
			"ON",
		},
		{
			"OFF",
			LightStateOff,
			"OFF",
		},
		{
			"Toggle",
			LightStateToggle,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.String() != tt.expected {
				t.Errorf("Expected %v, but got: %v", tt.expected, tt.input)
			}
		})
	}
}

func TestLightStateUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected LightState
	}{
		{
			"ON",
			`"ON"`,
			LightStateOn,
		},
		{
			"on",
			`"on"`,
			LightStateOn,
		},
		{
			"OFF",
			`"OFF"`,
			LightStateOff,
		},
		{
			"off",
			`"off"`,
			LightStateOff,
		},
		{
			"Toggle",
			`""`,
			LightStateToggle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var l LightState
			err := json.Unmarshal([]byte(tt.input), &l)
			if err != nil {
				t.Errorf("Unexpected error when unmarshaling JSON: %v", err)
			}
			if l != tt.expected {
				t.Errorf("Expected %v, but got: %v", tt.expected, l.String())
			}
		})
	}

	t.Run("InvalidInput", func(t *testing.T) {
		var l LightState
		err := json.Unmarshal([]byte(`"invalid"`), &l)
		if err == nil {
			t.Error("Expected error but got nil")
		}
		if err.Error() != `cannot unmarshal "invalid" into Go value of type *pkg.LightState` {
			t.Errorf("Unexpected error string: %v", err)
		}
	})
}

func TestLightStateMarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    LightState
		expected string
	}{
		{
			"ON",
			LightStateOn,
			`"ON"`,
		},
		{
			"OFF",
			LightStateOff,
			`"OFF"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := json.Marshal(tt.input)
			if err != nil {
				t.Errorf("Unexpected error when marshaling JSON: %v", err)
			}
			if string(result) != tt.expected {
				t.Errorf("Expected %v, but got: %s", tt.expected, string(result))
			}
		})
	}

	t.Run("InvalidLightState", func(t *testing.T) {
		result, err := json.Marshal(LightState(3))
		if err == nil {
			t.Error("Expected error but got nil")
		}
		if err.Error() != `json: error calling MarshalJSON for type pkg.LightState: cannot convert 3 to pkg.LightState` {
			t.Errorf("Unexpected error string: %v", err)
		}
		if string(result) != "" {
			t.Errorf("Expected empty string but got: %v", string(result))
		}
	})
}

func TestNumZones(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	tests := []struct {
		name     string
		garden   *Garden
		expected uint
	}{
		{
			"Zero",
			&Garden{},
			0,
		},
		{
			"One",
			&Garden{
				Zones: map[xid.ID]*Zone{xid.New(): {}},
			},
			1,
		},
		{
			"OneWithEndDated",
			&Garden{
				Zones: map[xid.ID]*Zone{
					xid.New(): {},
					xid.New(): {EndDate: &past},
				},
			},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.garden.NumZones()
			if result != tt.expected {
				t.Errorf("Expected %d but got %d", tt.expected, result)
			}
		})
	}
}

func TestPlantsByZone(t *testing.T) {
	zone := &Zone{ID: xid.New()}
	plant := &Plant{ID: xid.New(), ZoneID: zone.ID}
	past := time.Now().Add(-1 * time.Hour)
	endDatedPlant := &Plant{ID: xid.New(), EndDate: &past, ZoneID: zone.ID}

	tests := []struct {
		name        string
		garden      *Garden
		zoneID      xid.ID
		getEndDated bool
		expectedLen int
	}{
		{
			"Zero",
			&Garden{
				Zones: map[xid.ID]*Zone{zone.ID: zone},
			},
			zone.ID,
			true,
			0,
		},
		{
			"NoEndDated",
			&Garden{
				Zones:  map[xid.ID]*Zone{zone.ID: zone},
				Plants: map[xid.ID]*Plant{plant.ID: plant, endDatedPlant.ID: endDatedPlant},
			},
			zone.ID,
			false,
			1,
		},
		{
			"GetEndDated",
			&Garden{
				Zones:  map[xid.ID]*Zone{zone.ID: zone},
				Plants: map[xid.ID]*Plant{plant.ID: plant, endDatedPlant.ID: endDatedPlant},
			},
			zone.ID,
			true,
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plants := tt.garden.PlantsByZone(tt.zoneID, tt.getEndDated)
			if len(plants) != tt.expectedLen {
				t.Errorf("Expected %d Plants but got %d", tt.expectedLen, len(plants))
			}
		})
	}
}
