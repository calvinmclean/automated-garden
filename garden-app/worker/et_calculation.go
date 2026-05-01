package worker

import (
	"fmt"
	"math"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
)

// Monthly Kc values for citrus (January-December)
var citrusKcValues = []float32{0.50, 0.50, 0.80, 0.80, 0.80, 0.85, 1.00, 1.00, 1.00, 0.85, 0.50, 0.50}

// Species multipliers for citrus trees
var speciesMultipliers = map[weather.Species]float32{
	weather.SpeciesOrange:     1.00,
	weather.SpeciesGrapefruit: 1.20,
	weather.SpeciesLemon:      1.20,
	weather.SpeciesMandarin:   0.90,
}

// CalculateETDuration calculates watering duration using the citrus tree formula:
// G = Area × E × F, where E = ET₀ × Kc
// The ET value is expected in mm (from weather APIs) and is converted to inches for the formula
// Returns duration and nil error on success
func CalculateETDuration(config *weather.EvapotranspirationScaler, eto float32, interval time.Duration, now time.Time) (time.Duration, error) {
	const conversionFactor = 0.436 // F = 7.48/12 × 0.7
	const mmToInches = 1.0 / 25.4  // Convert mm to inches

	// Validate FlowRateGPH to prevent division by zero
	if config.FlowRateGPH == 0 {
		return 0, fmt.Errorf("flow_rate_gph cannot be zero")
	}

	// Convert ET from mm to inches (formula expects inches/day)
	etoInches := eto * mmToInches

	// Get current month for Kc (time.Month is 1-indexed, so subtract 1)
	currentMonth := now.Month() - 1
	kc := citrusKcValues[currentMonth]

	// Get species multiplier (default to 1.0 for unknown species)
	multiplier := speciesMultipliers[config.Species]
	if multiplier == 0 {
		multiplier = 1.00
	}

	// Calculate canopy area: π × r²
	radius := config.CanopyDiameterFeet / 2
	area := math.Pi * radius * radius

	// Calculate pan evaporation: E = ET₀ × Kc
	evaporation := etoInches * kc

	// Daily water requirement: G = Area × E × F × multiplier
	dailyGallons := float32(area) * evaporation * conversionFactor * multiplier

	// Total for interval period
	intervalDays := float32(interval.Hours() / 24)
	totalGallons := dailyGallons * intervalDays

	// Convert to duration: hours = gallons / GPH
	hoursNeeded := totalGallons / config.FlowRateGPH
	duration := time.Duration(hoursNeeded * float32(time.Hour))

	return duration, nil
}
