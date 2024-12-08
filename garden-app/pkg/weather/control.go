package weather

import "github.com/rs/xid"

// Control defines certain parameters and behaviors to influence watering patterns based off weather data
type Control struct {
	Rain        *ScaleControl `json:"rain_control,omitempty"`
	Temperature *ScaleControl `json:"temperature_control,omitempty"`
}

// Patch allows modifying the struct in-place with values from a different instance
func (wc *Control) Patch(newControl *Control) {
	if newControl.Rain != nil {
		if wc.Rain == nil {
			wc.Rain = &ScaleControl{}
		}
		wc.Rain.Patch(newControl.Rain)
	}
	if newControl.Temperature != nil {
		if wc.Temperature == nil {
			wc.Temperature = &ScaleControl{}
		}
		wc.Temperature.Patch(newControl.Temperature)
	}
}

// ScaleControl is a generic struct that enables scaling
// BaselineValue is the value that scaling starts at
// Range is the most extreme value that scaling will go to (used as max/min)
// Factor is the maximum amount that this will scale by. This must be between 0 and 1, where 0 is no scaling and 1 scale by the full proportion of the range
// When a measured value is equal to or greater than the BaselineValue, factor is used to scale up the current value based
// on the proportion of the difference between current value and BaselineValue to the Range
//
// For example:
// BaselineValue: 90, Factor: 0.5, Range: 30, WaterDuration: 30m
//   - Input 100 degrees: (100 - 90)/30 * 0.5 + 1 = 1.1666666667 => water 35m
//   - Input 120 degrees: (120 - 90)/30 * 0.5 + 1 = 1.5, max scaling
//   - Input 130 degrees: (130 - 90)/30 * 0.5 + 1 = 1.6666666667 => greater than factor of 0.5, so we cut off at 1.5 and water 45m
//   - Input  80 degrees: ( 80 - 90)/30 * 0.5 + 1 = 0.8333333333 => water 25m
//   - Input  60 degrees: ( 60 - 90)/30 * 0.5 + 1 = 0.5 => water 15m
//   - Input  50 degrees: ( 50 - 90)/30 * 0.5 + 1 = 0.3333333333 => less than factor of 0.5, so we round up to 0.5
//
// Basically, a Factor of 0.5 means that if watering is set at 30m, I want to water at most 45 min and at least 15 min
// This way, the control doesn't need to know anything about the durations and can just return a multiplier that
// makes this happen
type ScaleControl struct {
	BaselineValue *float32 `json:"baseline_value"`
	Factor        *float32 `json:"factor"`
	Range         *float32 `json:"range"`
	ClientID      xid.ID   `json:"client_id"`
}

// Patch allows modifying the struct in-place with values from a different instance
func (sc *ScaleControl) Patch(newScaleControl *ScaleControl) {
	if newScaleControl.BaselineValue != nil {
		sc.BaselineValue = newScaleControl.BaselineValue
	}
	if newScaleControl.Factor != nil {
		sc.Factor = newScaleControl.Factor
	}
	if newScaleControl.Range != nil {
		sc.Range = newScaleControl.Range
	}
	if !newScaleControl.ClientID.IsNil() {
		sc.ClientID = newScaleControl.ClientID
	}
}

// Scale calculates and returns the multiplier based on the input value
func (sc *ScaleControl) Scale(actualValue float32) float32 {
	diff := actualValue - *sc.BaselineValue
	r := *sc.Range
	if diff > r {
		diff = r
	}
	if diff < -r {
		diff = -r
	}
	return (diff/r)*(*sc.Factor) + 1
}

// InvertedScaleDownOnly calculates and returns the multiplier based on the input value, but is inverted
// so higher input values cause scaling < 1. Also it will only scale in this direction
func (sc *ScaleControl) InvertedScaleDownOnly(actualValue float32) float32 {
	// If the baseline is not reached, just scale 1
	if actualValue < *sc.BaselineValue {
		return 1
	}

	diff := actualValue - *sc.BaselineValue
	r := *sc.Range
	if diff > r {
		diff = r
	}
	return 1 - (diff/r)*(1-*sc.Factor)
}
