package weather

import (
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather/netatmo"
)

// Config is used to identify and configure a client type
type Config struct {
	Type    string                 `mapstructure:"type"`
	Options map[string]interface{} `mapstructure:"options"`
}

// Client is an interface defining the possible methods used to interact with the weather client APIs
type Client interface {
	GetTotalRain(since time.Duration) (float32, error)
	GetAverageHighTemperature(since time.Duration) (float32, error)
}

// NewClient will use the config to create and return the correct type of weather client. If no type is provided, this will
// return a nil client rather than an error since Weather client is not required
func NewClient(config Config) (Client, error) {
	switch config.Type {
	case "netatmo":
		return netatmo.NewClient(config.Options)
	case "":
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid type '%s'", config.Type)
	}
}
