package weather

import (
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather/netatmo"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather/units"
)

type Config struct {
	Type    string                 `mapstructure:"type"`
	Options map[string]interface{} `mapstructure:"options"`
}

type Client interface {
	GetTotalRain(since time.Time, unit units.RainUnit) (float32, error)
}

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
