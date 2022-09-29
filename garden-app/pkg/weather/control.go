package weather

// Control defines certain parameters and behaviors to influence watering patterns based off weather data
type Control struct {
	Rain         *RainControl         `json:"rain_control,omitempty"`
	SoilMoisture *SoilMoistureControl `json:"moisture_control,omitempty"`
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
