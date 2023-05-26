package kv

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/madflojo/hord"
	"github.com/madflojo/hord/drivers/hashmap"
	"github.com/madflojo/hord/drivers/redis"
	"gopkg.in/yaml.v3"
)

type Client struct {
	db        hord.Database
	options   map[string]string
	unmarshal func([]byte, interface{}) error
	marshal   func(interface{}) ([]byte, error)
}

func NewClient(options map[string]string) (*Client, error) {
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

func newFileClient(options map[string]string) (*Client, error) {
	if _, ok := options["filename"]; !ok {
		return nil, fmt.Errorf("missing config key 'filename'")
	}

	db, err := hashmap.Dial(hashmap.Config{
		Filename: options["filename"],
	})
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

	switch filepath.Ext(options["filename"]) {
	case ".json":
		client.unmarshal = json.Unmarshal
		client.marshal = json.Marshal
	case ".yml", ".yaml":
		client.unmarshal = yaml.Unmarshal
		client.marshal = yaml.Marshal
	}

	return client, nil
}

func newRedisClient(options map[string]string) (*Client, error) {
	server, ok := options["server"]
	if !ok {
		return nil, fmt.Errorf("missing config key 'server'")
	}

	db, err := redis.Dial(redis.Config{
		Server:   server,
		Password: options["password"],
	})
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
		unmarshal: yaml.Unmarshal,
		marshal:   yaml.Marshal,
	}, nil
}
