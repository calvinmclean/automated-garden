package yaml

import (
	"fmt"
	"sync"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Client implements the Client interface to use a YAML file as a storage mechanism
type Client struct {
	data    clientData
	Options map[string]string
	m       *sync.Mutex

	// yaml file source
	filename string

	// configmap source
	configMapName string
	keyName       string
	k8sClient     v1.ConfigMapInterface

	save   func() error
	update func() error
}

type clientData struct {
	Gardens              map[xid.ID]*pkg.Garden     `yaml:"gardens"`
	WeatherClientConfigs map[xid.ID]*weather.Config `yaml:"weather_clients"`
}

// NewClient creates a new storage backend using YAML format. It has options to store to a local YAML
// file or a K8s ConfigMap
func NewClient(storageType string, options map[string]string) (*Client, error) {
	switch storageType {
	case "YAML", "yaml":
		return newYAMLStorage(options)
	case "ConfigMap", "configmap":
		return newConfigMapStorage(options)
	default:
		return nil, fmt.Errorf("invalid type '%s'", storageType)
	}
}
