// Package weather provides weather client interfaces and implementations
package weather

// Control defines certain parameters and behaviors to influence watering patterns based off weather data
type Control struct {
	Rain        *WeatherScaler `json:"rain_control,omitempty"`
	Temperature *WeatherScaler `json:"temperature_control,omitempty"`
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
}
