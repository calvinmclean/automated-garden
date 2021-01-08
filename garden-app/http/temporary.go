package http

import "github.com/calvinmclean/automated-garden/garden-app/api"

var temporaryPlantsMap = map[string]api.Plant{
	"D89406D6-884D-48D3-98A8-7A282CD210EB": {
		Name:           "Cherry Tomato",
		ID:             "D89406D6-884D-48D3-98A8-7A282CD210EB",
		WateringAmount: 15000,
		Interval:       "24h",
		StartDate:      "2021-01-15",
		ValvePin:       3,
		PumpPin:        5,
	},
}
