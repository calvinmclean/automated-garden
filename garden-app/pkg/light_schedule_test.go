package pkg

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

func TestNextChange(t *testing.T) {
	tests := []struct {
		name          string
		ls            LightSchedule
		currentTime   time.Time
		expectedTime  time.Time
		expectedState LightState
	}{
		{
			name: "OnInOneHour",
			ls: LightSchedule{
				StartTime: &StartTime{Time: time.Date(0, 0, 0, 13, 0, 0, 0, time.UTC)},
				Duration:  &Duration{Duration: 12 * time.Hour},
			},
			currentTime:   time.Date(2025, time.November, 8, 12, 0, 0, 0, time.UTC),
			expectedTime:  time.Date(2025, time.November, 8, 13, 0, 0, 0, time.UTC),
			expectedState: LightStateOn,
		},
		{
			name: "OffInElevenHours",
			ls: LightSchedule{
				StartTime: &StartTime{Time: time.Date(0, 0, 0, 13, 0, 0, 0, time.UTC)},
				Duration:  &Duration{Duration: 12 * time.Hour},
			},
			currentTime:   time.Date(2025, time.November, 8, 14, 0, 0, 0, time.UTC),
			expectedTime:  time.Date(2025, time.November, 9, 1, 0, 0, 0, time.UTC),
			expectedState: LightStateOff,
		},
		{
			name: "TurnedOnYesterdayAndTurnsOffLater",
			ls: LightSchedule{
				StartTime: &StartTime{Time: time.Date(0, 0, 0, 20, 0, 0, 0, time.UTC)},
				Duration:  &Duration{Duration: 12 * time.Hour},
			},
			currentTime:   time.Date(2025, time.November, 8, 6, 0, 0, 0, time.UTC),
			expectedTime:  time.Date(2025, time.November, 8, 8, 0, 0, 0, time.UTC),
			expectedState: LightStateOff,
		},
		{
			// Light turns on at 7AM and off at 7PM. It is currently 10PM, so it will turn on tomorrow morning
			name: "TurnsOnAgainTomorrow",
			ls: LightSchedule{
				StartTime: &StartTime{Time: time.Date(0, 0, 0, 0o7, 0, 0, 0, time.UTC)},
				Duration:  &Duration{Duration: 12 * time.Hour},
			},
			currentTime:   time.Date(2023, time.November, 8, 22, 0, 0, 0, time.UTC),
			expectedTime:  time.Date(2023, time.November, 9, 0o7, 0, 0, 0, time.UTC),
			expectedState: LightStateOn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextTime, nextState := tt.ls.NextChange(tt.currentTime)
			assert.Equal(t, tt.expectedTime, nextTime)
			assert.Equal(t, tt.expectedState, nextState)
		})
	}
}

func TestExpectedStateAtTime(t *testing.T) {
	tests := []struct {
		name          string
		ls            LightSchedule
		currentTime   time.Time
		expectedState LightState
	}{
		{
			name: "OnInOneHour_CurrentlyOff",
			ls: LightSchedule{
				StartTime: &StartTime{Time: time.Date(0, 0, 0, 13, 0, 0, 0, time.UTC)},
				Duration:  &Duration{Duration: 12 * time.Hour},
			},
			currentTime:   time.Date(2025, time.November, 8, 12, 0, 0, 0, time.UTC),
			expectedState: LightStateOff,
		},
		{
			name: "OffInElevenHours_CurrentlyOn",
			ls: LightSchedule{
				StartTime: &StartTime{Time: time.Date(0, 0, 0, 13, 0, 0, 0, time.UTC)},
				Duration:  &Duration{Duration: 12 * time.Hour},
			},
			currentTime:   time.Date(2025, time.November, 8, 14, 0, 0, 0, time.UTC),
			expectedState: LightStateOn,
		},
		{
			name: "TurnedOnYesterdayAndTurnsOffLater_CurrentlyOn",
			ls: LightSchedule{
				StartTime: &StartTime{Time: time.Date(0, 0, 0, 20, 0, 0, 0, time.UTC)},
				Duration:  &Duration{Duration: 12 * time.Hour},
			},
			currentTime:   time.Date(2025, time.November, 8, 6, 0, 0, 0, time.UTC),
			expectedState: LightStateOn,
		},
		{
			// Light runs from 5PM to 5AM and it is currently 8AM
			name: "CurrentlyOff",
			ls: LightSchedule{
				StartTime: &StartTime{Time: time.Date(0, 0, 0, 17, 0, 0, 0, time.UTC)},
				Duration:  &Duration{Duration: 12 * time.Hour},
			},
			currentTime:   time.Date(2025, time.November, 9, 8, 0, 0, 0, time.UTC),
			expectedState: LightStateOff,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentState := tt.ls.ExpectedStateAtTime(tt.currentTime)
			assert.Equal(t, tt.expectedState, currentState)
		})
	}
}

// TestNextChange_Boundaries tests that NextChange and ExpectedStateAtTime handle
// exact ON/OFF boundary times correctly. These are regression tests for a bug where
// boundary times returned Toggle instead of the correct next state.
func TestNextChange_Boundaries(t *testing.T) {
	// Use a schedule similar to the user's: 19:00 UTC-7 ON, 14h duration
	// ON  = 02:00 UTC, OFF = 16:00 UTC
	tz := time.FixedZone("UTC-7", -7*60*60)
	ls := LightSchedule{
		StartTime: &StartTime{Time: time.Date(0, 0, 0, 19, 0, 0, 0, tz)},
		Duration:  &Duration{Duration: 14 * time.Hour},
	}

	tests := []struct {
		name            string
		now             time.Time
		expectedTime    time.Time
		expectedState   LightState
		expectedCurrent LightState
	}{
		{
			name:            "AtExactOnTime",
			now:             time.Date(2026, 5, 22, 2, 0, 0, 0, time.UTC),
			expectedTime:    time.Date(2026, 5, 22, 16, 0, 0, 0, time.UTC),
			expectedState:   LightStateOff,
			expectedCurrent: LightStateOn,
		},
		{
			name:            "AtExactOffTime",
			now:             time.Date(2026, 5, 22, 16, 0, 0, 0, time.UTC),
			expectedTime:    time.Date(2026, 5, 23, 2, 0, 0, 0, time.UTC),
			expectedState:   LightStateOn,
			expectedCurrent: LightStateOff,
		},
		{
			name:            "OneSecondBeforeOn",
			now:             time.Date(2026, 5, 22, 1, 59, 59, 0, time.UTC),
			expectedTime:    time.Date(2026, 5, 22, 2, 0, 0, 0, time.UTC),
			expectedState:   LightStateOn,
			expectedCurrent: LightStateOff,
		},
		{
			name:            "OneSecondAfterOn",
			now:             time.Date(2026, 5, 22, 2, 0, 1, 0, time.UTC),
			expectedTime:    time.Date(2026, 5, 22, 16, 0, 0, 0, time.UTC),
			expectedState:   LightStateOff,
			expectedCurrent: LightStateOn,
		},
		{
			name:            "OneSecondBeforeOff",
			now:             time.Date(2026, 5, 22, 15, 59, 59, 0, time.UTC),
			expectedTime:    time.Date(2026, 5, 22, 16, 0, 0, 0, time.UTC),
			expectedState:   LightStateOff,
			expectedCurrent: LightStateOn,
		},
		{
			name:            "OneSecondAfterOff",
			now:             time.Date(2026, 5, 22, 16, 0, 1, 0, time.UTC),
			expectedTime:    time.Date(2026, 5, 23, 2, 0, 0, 0, time.UTC),
			expectedState:   LightStateOn,
			expectedCurrent: LightStateOff,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextTime, nextState := ls.NextChange(tt.now)
			assert.True(t, tt.expectedTime.Equal(nextTime), "NextChange time mismatch: expected %v, got %v", tt.expectedTime, nextTime)
			assert.Equal(t, tt.expectedState, nextState, "NextChange state mismatch")

			currentState := ls.ExpectedStateAtTime(tt.now)
			assert.Equal(t, tt.expectedCurrent, currentState, "ExpectedStateAtTime mismatch")
		})
	}
}
