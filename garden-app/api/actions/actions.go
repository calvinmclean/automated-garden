package actions

import (
	"errors"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/api"
)

// ActionExecutor is an interface used to create generic actions that the CLI or webserver
// can execute without knowing much detail about what the action is really doing
type ActionExecutor interface {
	Execute(*api.Plant) error
}

// AggregateAction collects all the possible actions into a single struct/request so one
// or more action can be performed from a single request
type AggregateAction struct {
	Water *WaterAction `json:"water"`
	Stop  *StopAction  `json:"stop"`
}

// Bind is used to make this struct compatible with our REST API implemented with go-chi.
// It will verify that the request is valid
func (action *AggregateAction) Bind(r *http.Request) error {
	// a.AggregateAction is nil if no AggregateAction fields are sent in the request. Return an
	// error to avoid a nil pointer dereference.
	if action == nil || (action.Water == nil && action.Stop == nil) {
		return errors.New("missing required action fields")
	}

	return nil
}

// Execute is responsible for performing the actual individual actions in this aggregate.
// The actions are executed in a deliberate order to be most intuitive for a user that wants
// to perform multiple actions with one request
func (action *AggregateAction) Execute(p *api.Plant) error {
	if action.Stop != nil {
		if err := action.Stop.Execute(p); err != nil {
			return err
		}
	}
	if action.Water != nil {
		if err := action.Water.Execute(p); err != nil {
			return err
		}
	}
	return nil
}
