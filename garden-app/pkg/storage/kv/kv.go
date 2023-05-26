package kv

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

type Client struct {
	db        hord.Database
	options   map[string]interface{}
	unmarshal func([]byte, interface{}) error
	marshal   func(interface{}) ([]byte, error)
}

func NewClient(options map[string]interface{}) (*Client, error) {
	driver, ok := options["driver"]
	if !ok {
		return nil, fmt.Errorf("missing config key 'driver'")
	}

	switch driver {
	case "file":
		return newFileClient(options)
	case "redis":
		return newRedisClient(options)
	default:
		return nil, fmt.Errorf("invalid KV driver: %s", driver)
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
