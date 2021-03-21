package api

type WateringStrategy struct {
	WateringAmount int    `json:"watering_amount" yaml:"watering_amount,omitempty"`
	Type           string `json:"type" yaml:"type"`
	Interval       string `json:"interval,omitempty" yaml:"interval,omitempty"`
}

func (strategy WateringStrategy) GetWateringAmount() int {
	return strategy.WateringAmount
}
