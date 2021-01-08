package actions

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/api"
)

// ActionExecutor ...
type ActionExecutor interface {
	Execute(api.Plant) error
}

// WaterAction ...
type WaterAction struct {
	Duration int `json:"duration"`
}

// Execute ...
func (action *WaterAction) Execute(p api.Plant) error {
	fmt.Printf("Watering plant %s for %dms\n", p.ID, action.Duration)
	return nil
}

// SkipAction ...
type SkipAction struct {
	Count int `json:"count"`
}

// Execute ...
func (action *SkipAction) Execute(p api.Plant) error {
	fmt.Printf("Skipping next %d waterings for plant %s\n", action.Count, p.ID)
	return nil
}

// AggregateAction ...
type AggregateAction struct {
	Water *WaterAction `json:"water"`
	Skip  *SkipAction  `json:"skip"`
}

// Execute ...
func (action *AggregateAction) Execute(p api.Plant) error {
	if action.Skip != nil {
		action.Skip.Execute(p)
	}
	if action.Water != nil {
		action.Water.Execute(p)
	}
	return nil
}
