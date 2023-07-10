package storage

import (
	"encoding/json"
	"fmt"
	"path/filepath"

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

// Client is a wrapper around hord.Database to allow for easy interactions with resources
type Client struct {
	db        hord.Database
	options   map[string]interface{}
	unmarshal func([]byte, interface{}) error
	marshal   func(interface{}) ([]byte, error)
}

// NewClient will create a new DB connection for one of the supported hord backends:
//   - hashmap
//   - redis
func NewClient(config Config) (*Client, error) {
	switch config.Driver {
	case "hashmap":
		return newFileClient(config.Options)
	case "redis":
		return newRedisClient(config.Options)
	default:
		return nil, fmt.Errorf("invalid KV driver: %q", config.Driver)
	}
}

func newFileClient(options map[string]interface{}) (*Client, error) {
	var cfg hashmap.Config
	err := mapstructure.Decode(options, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error decoding config: %w", err)
	}

	db, err := hashmap.Dial(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating database connection: %w", err)
	}

	err = db.Setup()
	if err != nil {
		return nil, fmt.Errorf("error setting up database: %w", err)
	}

	client := &Client{
		db:      db,
		options: options,
	}

	switch filepath.Ext(cfg.Filename) {
	case ".json", "":
		client.unmarshal = json.Unmarshal
		client.marshal = json.Marshal
	case ".yml", ".yaml":
		client.unmarshal = yaml.Unmarshal
		client.marshal = yaml.Marshal
	}

	return client, nil
}

func newRedisClient(options map[string]interface{}) (*Client, error) {
	var cfg redis.Config
	err := mapstructure.Decode(options, &cfg)
	if err != nil {
		return nil, fmt.Errorf("error decoding config: %w", err)
	}

	db, err := redis.Dial(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating database connection: %w", err)
	}

	err = db.Setup()
	if err != nil {
		return nil, fmt.Errorf("error setting up database: %w", err)
	}

	return &Client{
		db:        db,
		options:   options,
		unmarshal: json.Unmarshal,
		marshal:   json.Marshal,
	}, nil
}
