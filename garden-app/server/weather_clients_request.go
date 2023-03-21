package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
)

// WeatherClientRequest wraps a Zone into a request so we can handle Bind/Render in this package
type WeatherClientRequest struct {
	*weather.Config
}

// Bind is used to make this struct compatible with the go-chi webserver for reading incoming
// JSON requests
func (wc *WeatherClientRequest) Bind(r *http.Request) error {
	if wc == nil || wc.Config == nil {
		return errors.New("missing required WeatherClient fields")
	}
	if wc.Config.Type == "" {
		return errors.New("missing required type field")
	}
	if wc.Config.Options == nil {
		return errors.New("missing required options field")
	}

	_, err := weather.NewClient(wc.Config)
	if err != nil {
		return fmt.Errorf("failed to create valid client using config: %w", err)
	}

	return nil
}
