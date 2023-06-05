package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
)

// WeatherClientRequest wraps a WeatherClient config into a request so we can handle Bind/Render in this package
type WeatherClientRequest struct {
	*weather.Config
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (wc *WeatherClientRequest) Bind(_ *http.Request) error {
	if wc == nil || wc.Config == nil {
		return errors.New("missing required WeatherClient fields")
	}
	if wc.Config.Type == "" {
		return errors.New("missing required type field")
	}
	if wc.Config.Options == nil {
		return errors.New("missing required options field")
	}

	_, err := weather.NewClient(wc.Config, func(map[string]interface{}) error { return nil })
	if err != nil {
		return fmt.Errorf("failed to create valid client using config: %w", err)
	}

	return nil
}

// UpdateWeatherClientRequest wraps a WeatherClient config into a request so we can handle Bind/Render in this package
// It has different validation than the WeatherClientRequest
type UpdateWeatherClientRequest struct {
	*weather.Config
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (wc *UpdateWeatherClientRequest) Bind(_ *http.Request) error {
	if wc == nil || wc.Config == nil {
		return errors.New("missing required WeatherClient fields")
	}
	if wc.ID != xid.NilID() {
		return errors.New("updating ID is not allowed")
	}

	return nil
}
