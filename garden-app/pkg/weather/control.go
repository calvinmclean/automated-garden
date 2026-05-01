// Package weather provides weather client interfaces and implementations
package weather

import (
	"errors"

	"github.com/rs/xid"
)

// Species represents citrus tree species for ET calculation
type Species string

const (
	SpeciesOrange     Species = "orange"
	SpeciesGrapefruit Species = "grapefruit"
	SpeciesLemon      Species = "lemon"
	SpeciesMandarin   Species = "mandarin"
)

// Control defines certain parameters and behaviors to influence watering patterns based off weather data
type Control struct {
	Rain               *WeatherScaler            `json:"rain_control,omitempty"`
	Temperature        *WeatherScaler            `json:"temperature_control,omitempty"`
	Evapotranspiration *EvapotranspirationScaler `json:"evapotranspiration_control,omitempty"`
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

// Patch allows modifying the struct in-place with values from a different instance
func (wc *Control) Patch(newControl *Control) {
	if newControl.Rain != nil {
		if wc.Rain == nil {
			wc.Rain = &WeatherScaler{}
		}
		wc.Rain.Patch(newControl.Rain)
	}
	if newControl.Temperature != nil {
		if wc.Temperature == nil {
			wc.Temperature = &WeatherScaler{}
		}
		wc.Temperature.Patch(newControl.Temperature)
	}
	if newControl.Evapotranspiration != nil {
		wc.Evapotranspiration = newControl.Evapotranspiration
	}
}
