package storage

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"

	"github.com/madflojo/hord"
	"github.com/madflojo/hord/drivers/hashmap"
	"github.com/madflojo/hord/drivers/redis"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

// Config is used to identify and configure a storage client
type Config struct {
	Driver  string                 `mapstructure:"driver"`
	Options map[string]interface{} `mapstructure:"options"`
}

type Client struct {
	Gardens              babyapi.Storage[*pkg.Garden]
	Zones                babyapi.Storage[*pkg.Zone]
	WaterSchedules       babyapi.Storage[*pkg.WaterSchedule]
	WeatherClientConfigs babyapi.Storage[*weather.Config]
}

func NewClient(config Config) (*Client, error) {
	bc, err := newBaseClient(config)
	if err != nil {
		return nil, fmt.Errorf("error creating base client: %w", err)
	}

	return &Client{
		Gardens:              newGenericClient[*pkg.Garden](bc, "Garden"),
		Zones:                newGenericClient[*pkg.Zone](bc, "Zone"),
		WaterSchedules:       newGenericClient[*pkg.WaterSchedule](bc, "WaterSchedule"),
		WeatherClientConfigs: newGenericClient[*weather.Config](bc, "WeatherClient"),
	}, nil
}

type client struct {
	db        hord.Database
	options   map[string]interface{}
	unmarshal func([]byte, interface{}) error
	marshal   func(interface{}) ([]byte, error)
}

// newBaseClient will create a new DB connection for one of the supported hord backends:
//   - hashmap
//   - redis
func newBaseClient(config Config) (*client, error) {
	client := &client{
		options: config.Options,
	}
	var err error
	switch config.Driver {
	case "hashmap":
		err = client.initFileDB(config.Options)
	case "redis":
		err = client.initRedisDB(config.Options)
	default:
		return nil, fmt.Errorf("invalid KV driver: %q", config.Driver)
	}
	if err != nil {
		return nil, fmt.Errorf("error initializing DB: %w", err)
	}

	return client, nil
}

func (c *client) initFileDB(options map[string]interface{}) error {
	var cfg hashmap.Config
	err := mapstructure.Decode(options, &cfg)
	if err != nil {
		return fmt.Errorf("error decoding config: %w", err)
	}

	c.db, err = hashmap.Dial(cfg)
	if err != nil {
		return fmt.Errorf("error creating database connection: %w", err)
	}

	err = c.db.Setup()
	if err != nil {
		return fmt.Errorf("error setting up database: %w", err)
	}

	switch filepath.Ext(cfg.Filename) {
	case ".json", "":
		c.unmarshal = json.Unmarshal
		c.marshal = json.Marshal
	case ".yml", ".yaml":
		c.unmarshal = yaml.Unmarshal
		c.marshal = yaml.Marshal
	}

	return nil
}

func (c *client) initRedisDB(options map[string]interface{}) error {
	var cfg redis.Config
	err := mapstructure.Decode(options, &cfg)
	if err != nil {
		return fmt.Errorf("error decoding config: %w", err)
	}

	c.db, err = redis.Dial(cfg)
	if err != nil {
		return fmt.Errorf("error creating database connection: %w", err)
	}

	err = c.db.Setup()
	if err != nil {
		return fmt.Errorf("error setting up database: %w", err)
	}

	c.unmarshal = json.Unmarshal
	c.marshal = json.Marshal

	return nil
}
