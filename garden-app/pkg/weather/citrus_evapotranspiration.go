package weather

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/units"
	"github.com/rs/xid"
)

// Species represents citrus tree species for ET calculation
type Species string

const (
	SpeciesOrange     Species = "orange"
	SpeciesGrapefruit Species = "grapefruit"
	SpeciesLemon      Species = "lemon"
	SpeciesLime       Species = "lime"
	SpeciesMandarin   Species = "mandarin"
)

// Monthly Kc values for citrus (January-December)
var citrusKcValues = []float32{0.50, 0.50, 0.80, 0.80, 0.80, 0.85, 1.00, 1.00, 1.00, 0.85, 0.50, 0.50}

// Species multipliers for citrus trees
var speciesMultipliers = map[Species]float32{
	SpeciesOrange:     1.00,
	SpeciesGrapefruit: 1.20,
	SpeciesLemon:      1.20,
	SpeciesLime:       1.20,
	SpeciesMandarin:   0.90,
}

// EvapotranspirationScaler configures ET-based watering using citrus tree formula
type EvapotranspirationScaler struct {
	ClientID           xid.ID  `json:"client_id"`
	CanopyDiameterFeet float32 `json:"canopy_diameter_feet"`
	Species            Species `json:"species"`
	FlowRateGPH        float32 `json:"flow_rate_gph"`
}

// Validate checks that the EvapotranspirationScaler has valid configuration
func (e *EvapotranspirationScaler) Validate() error {
	if e.CanopyDiameterFeet <= 0 {
		return errors.New("canopy_diameter_feet must be greater than 0")
	}
	if e.FlowRateGPH <= 0 {
		return errors.New("flow_rate_gph must be greater than 0")
	}
	if e.ClientID.IsZero() {
		return errors.New("client_id is required")
	}
	return nil
}

// CalculateETDuration calculates watering duration using the citrus tree formula:
// G = Area × E × F, where E = ET₀ × Kc
// The ET value is expected in mm (from weather APIs) and is converted to inches for the formula
// Returns duration and nil error on success
func (e *EvapotranspirationScaler) CalculateETDuration(eto float32, interval time.Duration, now time.Time) (time.Duration, error) {
	const conversionFactor = 0.436 // F = 7.48/12 × 0.7

	// Validate FlowRateGPH to prevent division by zero
	if e.FlowRateGPH == 0 {
		return 0, fmt.Errorf("flow_rate_gph cannot be zero")
	}

	// Convert ET from mm to inches (formula expects inches/day)
	etoInches := units.MmToInches(eto)

	// Get current month for Kc (time.Month is 1-indexed, so subtract 1)
	currentMonth := now.Month() - 1
	kc := citrusKcValues[currentMonth]

	// Get species multiplier (default to 1.0 for unknown species)
	multiplier := speciesMultipliers[e.Species]
	if multiplier == 0 {
		multiplier = 1.00
	}

	// Calculate canopy area: π × r²
	radius := e.CanopyDiameterFeet / 2
	area := math.Pi * radius * radius

	// Calculate pan evaporation: E = ET₀ × Kc
	evaporation := etoInches * kc

	// Daily water requirement: G = Area × E × F × multiplier
	dailyGallons := float32(area) * evaporation * conversionFactor * multiplier

	// Total for interval period
	intervalDays := float32(interval.Hours() / 24)
	totalGallons := dailyGallons * intervalDays

	// Convert to duration: hours = gallons / GPH
	hoursNeeded := totalGallons / e.FlowRateGPH
	duration := time.Duration(hoursNeeded * float32(time.Hour))

	return duration, nil
}
