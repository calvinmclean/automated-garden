package http

import "github.com/calvinmclean/automated-garden/garden-app/api"

var temporaryPlantsMap = map[string]api.Plant{
	"9m4e2mr0ui3e8a215n4g": {
		Name:           "Cherry Tomato",
		ID:             "9m4e2mr0ui3e8a215n4g",
		WateringAmount: 15000,
		Interval:       "24h",
		StartDate:      "2021-01-15",
		ValvePin:       3,
		PumpPin:        5,
	},
}
