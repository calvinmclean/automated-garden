package weather

// Control defines certain parameters and behaviors to influence watering patterns based off weather data
type Control struct {
	Rain *RainControl `json:"rain_control"`
}

// RainControl defines parameters for delaying watering based on recently-recorded rain data. This will skip/delay watering if
// the rain threshold was exceeded between now and the most recent watering time
type RainControl struct {
	Threshold float32 `json:"threshold"`
}
