package weather

import (
	"errors"
	"fmt"
)

// WeatherScaler defines the configuration for weather-based duration scaling
// using min/max input-output pairs with configurable interpolation
type WeatherScaler struct {
	Enabled       bool              `json:"enabled" yaml:"enabled"`
	ClientID      string            `json:"client_id" yaml:"client_id"`
	Interpolation InterpolationMode `json:"interpolation" yaml:"interpolation"`
	InputMin      float64           `json:"input_min" yaml:"input_min"`
	InputMax      float64           `json:"input_max" yaml:"input_max"`
	FactorMin     float64           `json:"factor_min" yaml:"factor_min"`
	FactorMax     float64           `json:"factor_max" yaml:"factor_max"`
}

// Validate checks that the WeatherScaler configuration is valid
func (ws *WeatherScaler) Validate() error {
	if !ws.Enabled {
		return nil
	}
	if ws.InputMax <= ws.InputMin {
		return errors.New("input_max must be greater than input_min")
	}
	if ws.FactorMin < 0 || ws.FactorMax < 0 {
		return errors.New("factors must be non-negative")
	}
	if !ws.Interpolation.IsValid() {
		return fmt.Errorf("invalid interpolation mode: %s", ws.Interpolation)
	}
	return nil
}

// Scale calculates the scale factor for the given input value.
// It applies clamping for values outside the input range and
// interpolates values within the range.
func (ws *WeatherScaler) Scale(input float64) float64 {
	if !ws.Enabled {
		return 1.0
	}

	// Clamp to lower boundary
	if input <= ws.InputMin {
		return ws.FactorMin
	}

	// Clamp to upper boundary
	if input >= ws.InputMax {
		return ws.FactorMax
	}

	// Normalize input to 0-1 range
	t := (input - ws.InputMin) / (ws.InputMax - ws.InputMin)

	// Apply interpolation
	interpolator := GetInterpolator(ws.Interpolation)
	interpolated := interpolator.Interpolate(t)

	// Map interpolated value to output range
	return ws.FactorMin + (ws.FactorMax-ws.FactorMin)*interpolated
}

// ScaleMulti multiplies multiple scaler factors together.
// Disabled scalers contribute a factor of 1.0.
func ScaleMulti(scalers []*WeatherScaler, inputs []float64) float64 {
	if len(scalers) != len(inputs) {
		return 1.0
	}

	result := 1.0
	for i, scaler := range scalers {
		result *= scaler.Scale(inputs[i])
	}
	return result
}
