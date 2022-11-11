package weather

// Control defines certain parameters and behaviors to influence watering patterns based off weather data
type Control struct {
	Rain         *RainControl         `json:"rain_control,omitempty"`
	SoilMoisture *SoilMoistureControl `json:"moisture_control,omitempty"`
	Temperature  *ScaleControl        `json:"temperature_control,omitempty"`
}

// RainControl defines parameters for delaying watering based on recently-recorded rain data. This will skip/delay watering if
// the rain threshold was exceeded between now and the most recent watering time
type RainControl struct {
	Threshold float32 `json:"threshold"`
}

// SoilMoistureControl defines parameters for delaying watering based on soil moisture data. This will skip watering if the
// soil moisture is below the minimum
// soil moisture value is currently hard-coded as the average value over the last 15 minutes
type SoilMoistureControl struct {
	MinimumMoisture int `json:"minimum_moisture,omitempty"`
}

// ScaleControl is a generic struct that enables scaling
// BaselineTemperature is the value that scaling starts at
// Range is the most extreme value that scaling will go to (used as max/min)
// Factor is the maximum amount that this will scale by. This must be between 0 and 1, where 0 is no scaling and 1 scale by the full proportion of the range
// When a measured value is equal to or greater than the BaselineTemperature, factor is used to scale up the current value based
// on the proportion of the difference between current value and BaselineTemperature to the Range
//
// For example:
// BaselineTemperature: 90, Factor: 0.5, Range: 30, WaterDuration: 30m
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
	BaselineTemperature *float32 `json:"baseline_temperature"`
	Factor              *float32 `json:"factor"`
	Range               *float32 `json:"range"`
}

// Scale calculates and returns the multiplier based on the input temperature value
func (sc *ScaleControl) Scale(actualTemperature float32) float32 {
	diff := actualTemperature - *sc.BaselineTemperature
	r := *sc.Range
	if diff > r {
		diff = r
	}
	if diff < -r {
		diff = -r
	}
	return (diff/r)*(*sc.Factor) + 1
}
