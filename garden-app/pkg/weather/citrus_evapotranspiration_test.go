package weather

import (
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func TestCalculateETDuration(t *testing.T) {
	tests := []struct {
		name          string
		config        *EvapotranspirationScaler
		eto           float32
		interval      time.Duration
		now           time.Time
		expectedMin   time.Duration
		expectedMax   time.Duration
		expectError   bool
		errorContains string
	}{
		{
			name: "Orange_16ft_August_10.16mmET_14days",
			config: &EvapotranspirationScaler{
				ClientID:           xid.New(),
				CanopyDiameterFeet: 16,
				Species:            SpeciesOrange,
				FlowRateGPH:        10,
			},
			eto:         10.16,                                        // mm/day (0.4 inches)
			interval:    14 * 24 * time.Hour,                          // 14 days
			now:         time.Date(2024, 8, 15, 0, 0, 0, 0, time.UTC), // August (Kc = 1.00)
			expectError: false,
			// Area = π × 8² = 201 ft²
			// E = 0.4 × 1.00 (August Kc) = 0.4 inches
			// G_daily = 201 × 0.4 × 0.436 = 35.05 gallons/day
			// G_total = 35.05 × 14 = 490.7 gallons
			// Duration = 490.7 / 10 = 49.07 hours
			expectedMin: 48 * time.Hour,
			expectedMax: 50 * time.Hour,
		},
		{
			name: "Grapefruit_20ft_July_12.7mmET_10days",
			config: &EvapotranspirationScaler{
				ClientID:           xid.New(),
				CanopyDiameterFeet: 20,
				Species:            SpeciesGrapefruit,
				FlowRateGPH:        15,
			},
			eto:         12.7, // mm/day (0.5 inches)
			interval:    10 * 24 * time.Hour,
			now:         time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC), // July (Kc = 1.00)
			expectError: false,
			// Area = π × 10² = 314 ft²
			// E = 0.5 × 1.00 (July Kc) = 0.5 inches
			// G_daily = 314 × 0.5 × 0.436 × 1.2 (grapefruit multiplier) = 82.15 gallons/day
			// G_total = 82.15 × 10 = 821.5 gallons
			// Duration = 821.5 / 15 = 54.77 hours
			expectedMin: 54 * time.Hour,
			expectedMax: 56 * time.Hour,
		},
		{
			name: "Lemon_12ft_March_7.62mmET_21days",
			config: &EvapotranspirationScaler{
				ClientID:           xid.New(),
				CanopyDiameterFeet: 12,
				Species:            SpeciesLemon,
				FlowRateGPH:        8,
			},
			eto:         7.62, // mm/day (0.3 inches)
			interval:    21 * 24 * time.Hour,
			now:         time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), // March (Kc = 0.80)
			expectError: false,
			// Area = π × 6² = 113 ft²
			// E = 0.3 × 0.80 (March Kc) = 0.24 inches
			// G_daily = 113 × 0.24 × 0.436 × 1.2 = 14.17 gallons/day
			// G_total = 14.17 × 21 = 297.6 gallons
			// Duration = 297.6 / 8 = 37.2 hours
			expectedMin: 36 * time.Hour,
			expectedMax: 38 * time.Hour,
		},
		{
			name: "Mandarin_10ft_June_8.89mmET_14days",
			config: &EvapotranspirationScaler{
				ClientID:           xid.New(),
				CanopyDiameterFeet: 10,
				Species:            SpeciesMandarin,
				FlowRateGPH:        5,
			},
			eto:         8.89, // mm/day (0.35 inches)
			interval:    14 * 24 * time.Hour,
			now:         time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), // June (Kc = 0.85)
			expectError: false,
			// Area = π × 5² = 78.5 ft²
			// E = 0.35 × 0.85 (June Kc) = 0.2975 inches
			// G_daily = 78.5 × 0.2975 × 0.436 × 0.9 (mandarin multiplier) = 9.15 gallons/day
			// G_total = 9.15 × 14 = 128.1 gallons
			// Duration = 128.1 / 5 = 25.62 hours
			expectedMin: 25 * time.Hour,
			expectedMax: 26 * time.Hour,
		},
		{
			name: "ZeroFlowRate_Error",
			config: &EvapotranspirationScaler{
				ClientID:           xid.New(),
				CanopyDiameterFeet: 16,
				Species:            SpeciesOrange,
				FlowRateGPH:        0,
			},
			eto:           10.16, // mm/day (0.4 inches)
			interval:      14 * 24 * time.Hour,
			now:           time.Date(2024, 8, 15, 0, 0, 0, 0, time.UTC),
			expectError:   true,
			errorContains: "flow_rate_gph cannot be zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := tt.config.CalculateETDuration(tt.eto, tt.interval, tt.now)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			assert.NoError(t, err)
			assert.True(t, duration >= tt.expectedMin && duration <= tt.expectedMax,
				"duration %v should be between %v and %v", duration, tt.expectedMin, tt.expectedMax)
		})
	}
}

func TestSpeciesMultipliers(t *testing.T) {
	// Test that all species constants have multipliers
	speciesList := []Species{
		SpeciesOrange,
		SpeciesGrapefruit,
		SpeciesLemon,
		SpeciesMandarin,
	}

	for _, species := range speciesList {
		multiplier := speciesMultipliers[species]
		assert.True(t, multiplier > 0, "species %s should have a multiplier", species)
	}
}

func TestCitrusKcValues(t *testing.T) {
	// Verify Kc values are defined for all 12 months
	assert.Equal(t, 12, len(citrusKcValues))

	// Verify expected ranges (should be between 0.3 and 1.2)
	for i, kc := range citrusKcValues {
		assert.True(t, kc >= 0.3 && kc <= 1.2,
			"Kc value for month %d (%.2f) should be between 0.3 and 1.2", i+1, kc)
	}
}
