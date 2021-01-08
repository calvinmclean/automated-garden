package api

// Plant ...
type Plant struct {
	Name           string `json:"name"`
	ID             string `json:"id"`
	WateringAmount int    `json:"watering_amount"`
	Interval       string `json:"interval"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	ValvePin       int    `json:"valve_pin"`
	PumpPin        int    `json:"pump_pin"`
}
