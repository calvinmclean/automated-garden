package storage

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

// Config is used to identify and configure a storage client
type Config struct {
	Type    string            `mapstructure:"type"`
	Options map[string]string `mapstructure:"options"`
}

// Client is a "generic" interface used to interact with our storage backend (DB, file, etc)
type Client interface {
	GetGarden(xid.ID) (*pkg.Garden, error)
	GetGardens(bool) ([]*pkg.Garden, error)
	SaveGarden(*pkg.Garden) error

	GetPlant(xid.ID, xid.ID) (*pkg.Plant, error)
	GetPlants(xid.ID, bool) ([]*pkg.Plant, error)
	SavePlant(xid.ID, *pkg.Plant) error

	Save() error
}

// NewStorageClient will use the config to create and return the correct type of storage client
func NewStorageClient(config Config) (Client, error) {
	switch config.Type {
	case "YAML", "yaml":
		return NewYAMLClient(config)
	case "ConfigMap", "configmap":
		return NewConfigMapClient(config)
	default:
		return nil, fmt.Errorf("invalid type '%s'", config.Type)
	}
}
