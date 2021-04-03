package server

import (
	"errors"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/api"
)

// PlantRequest wraps a Plant into a request so we can handle Bind/Render in this package
type PlantRequest struct {
	*api.Plant
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (p *PlantRequest) Bind(r *http.Request) error {
	if p == nil {
		return errors.New("missing required Plant fields")
	}

	if p.WateringStrategy == (api.WateringStrategy{}) {
		return errors.New("missing required watering_strategy field")
	}
	if p.WateringStrategy.Interval == "" {
		return errors.New("missing required watering_strategy.interval field")
	}
	if p.WateringStrategy.WateringAmount == 0 {
		return errors.New("missing required watering_strategy.watering_amount field")
	}
	if p.Name == "" {
		return errors.New("missing required name field")
	}
	if p.Garden == "" {
		return errors.New("missing required garden field")
	}

	return nil
}

// AggregateActionRequest wraps a AggregateAction into a request so we can handle Bind/Render in this package
type AggregateActionRequest struct {
	*api.AggregateAction
}

// Bind is used to make this struct compatible with our REST API implemented with go-chi.
// It will verify that the request is valid
func (action *AggregateActionRequest) Bind(r *http.Request) error {
	// a.AggregateAction is nil if no AggregateAction fields are sent in the request. Return an
	// error to avoid a nil pointer dereference.
	if action == nil || (action.Water == nil && action.Stop == nil) {
		return errors.New("missing required action fields")
	}

	return nil
}
