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
