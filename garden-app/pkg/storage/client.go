package storage

import (
	"encoding/json"
	"fmt"
	"path/filepath"

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

type BaseClient struct {
	db        hord.Database
	options   map[string]interface{}
	unmarshal func([]byte, interface{}) error
	marshal   func(interface{}) ([]byte, error)
}

// NewBaseClient will create a new DB connection for one of the supported hord backends:
//   - hashmap
//   - redis
func NewBaseClient(config Config) (*BaseClient, error) {
	client := &BaseClient{
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

func (c *BaseClient) initFileDB(options map[string]interface{}) error {
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

func (c *BaseClient) initRedisDB(options map[string]interface{}) error {
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

func FilterEndDated[T babyapi.EndDateable](getEndDated bool) babyapi.FilterFunc[T] {
	return func(item T) bool {
		return getEndDated || !item.EndDated()
	}
}
