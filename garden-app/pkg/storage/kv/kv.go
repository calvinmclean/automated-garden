package kv

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/madflojo/hord"
	"github.com/madflojo/hord/drivers/hashmap"
	"gopkg.in/yaml.v3"
)

type Client struct {
	db        hord.Database
	options   map[string]string
	unmarshal func([]byte, interface{}) error
	marshal   func(interface{}) ([]byte, error)
}

func NewClient(options map[string]string) (*Client, error) {
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
