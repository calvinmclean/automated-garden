package actions

import (
	"github.com/calvinmclean/automated-garden/garden-app/api"
)

// ActionExecutor ...
type ActionExecutor interface {
	Execute(api.Plant) error
}

// AggregateAction ...
type AggregateAction struct {
	Water *WaterAction `json:"water"`
	Skip  *SkipAction  `json:"skip"`
	Stop  *StopAction  `json:"stop"`
}

// Execute ...
func (action *AggregateAction) Execute(p api.Plant) error {
	if action.Skip != nil {
		action.Skip.Execute(p)
	}
	if action.Water != nil {
		action.Water.Execute(p)
	}
	if action.Stop != nil {
		action.Stop.Execute(p)
	}
	return nil
}
