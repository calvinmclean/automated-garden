/*
Package hashmap provides a Hord database driver for an in-memory hashmap.

The Hashmap driver is a simple, in-memory key-value store that stores data in a hashmap structure. To use this driver, import it as follows:

	import (
	    "github.com/madflojo/hord"
	    "github.com/madflojo/hord/hashmap"
	)

# Connecting to the Database

Use the Dial() function to create a new client for interacting with the hashmap driver.

	var db hord.Database
	db, err := hashmap.Dial(hashmap.Config{})
	if err != nil {
	    // Handle connection error
	}

# Initialize database

Hord provides a Setup() function for preparing a database. This function is safe to execute after every Dial().

	err := db.Setup()
	if err != nil {
	    // Handle setup error
	}

# Database Operations

Hord provides a simple abstraction for working with the hashmap driver, with easy-to-use methods such as Get() and Set() to read and write values.

	// Connect to the hashmap database
	db, err := hashmap.Dial(hashmap.Config{})
	if err != nil {
	    // Handle connection error
	}

	err := db.Setup()
	if err != nil {
	    // Handle setup error
	}

	// Set a value
	err = db.Set("key", []byte("value"))
	if err != nil {
	    // Handle error
	}

	// Retrieve a value
	value, err := db.Get("key")
	if err != nil {
	    // Handle error
	}
*/
package hashmap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/madflojo/hord"
	"gopkg.in/yaml.v3"
)

// Config represents the configuration for the hashmap database.
type Config struct {
	// Filename is an optional parameter that accepts the path to a YAML or JSON file to read/write data
	Filename string
}

// Database is an in-memory hashmap implementation of the hord.Database interface.
type Database struct {
	sync.RWMutex

	config Config

	// data is used to store data in a simple map
	data map[string]ByteSlice
}

// Dial initializes and returns a new hashmap database instance.
func Dial(conf Config) (*Database, error) {
	if conf.Filename != "" {
		switch filepath.Ext(conf.Filename) {
		case ".yaml", ".yml", ".json":
		default:
			return nil, errors.New("filename must have yaml, yml, or json extension")
		}
	}

	db := &Database{config: conf}
	db.data = make(map[string]ByteSlice)
	return db, nil
}

// Setup sets up the hashmap database. If file storage is enabled, this will load from the file or create it if it does not exist.
func (db *Database) Setup() error {
	if db.config.Filename == "" {
		return nil
	}

	db.Lock()
	defer db.Unlock()

	// check file and create if it does not exist
	file, err := os.OpenFile(db.config.Filename, os.O_RDONLY|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("error checking file %q: %w", db.config.Filename, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("unable to read local file: %w", err)
	}

	switch filepath.Ext(db.config.Filename) {
	case ".json":
		// json fails to read empty input
		if len(data) != 0 {
			err = json.Unmarshal(data, &db.data)
		}
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &db.data)
	}
	if err != nil {
		return fmt.Errorf("unable to unmarshal data from file: %w", err)
	}

	return nil
}

// Get retrieves data from the hashmap database based on the provided key.
// It returns the data associated with the key or an error if the key is invalid or the data does not exist.
func (db *Database) Get(key string) ([]byte, error) {
	if err := hord.ValidKey(key); err != nil {
		return []byte(""), err
	}

	db.RLock()
	defer db.RUnlock()
	if db.data == nil {
		return []byte(""), hord.ErrNoDial
	}

	v, ok := db.data[key]
	if ok {
		return v, nil
	}
	return []byte(""), hord.ErrNil
}

// Set inserts or updates data in the hashmap database based on the provided key.
// It returns an error if the key or data is invalid.
func (db *Database) Set(key string, data []byte) error {
	if err := hord.ValidKey(key); err != nil {
		return err
	}

	if err := hord.ValidData(data); err != nil {
		return err
	}

	db.Lock()
	defer db.Unlock()
	if db.data == nil {
		return hord.ErrNoDial
	}

	db.data[key] = data
	return db.saveToLocalFile()
}

// Delete removes data from the hashmap database based on the provided key.
// It returns an error if the key is invalid.
func (db *Database) Delete(key string) error {
	if err := hord.ValidKey(key); err != nil {
		return err
	}

	db.Lock()
	defer db.Unlock()
	if db.data == nil {
		return hord.ErrNoDial
	}

	delete(db.data, key)
	return db.saveToLocalFile()
}

// Keys retrieves a list of keys stored in the hashmap database.
func (db *Database) Keys() ([]string, error) {
	db.RLock()
	defer db.RUnlock()
	if db.data == nil {
		return []string{}, hord.ErrNoDial
	}

	var keys []string
	for k := range db.data {
		keys = append(keys, k)
	}
	return keys, nil
}

// HealthCheck performs a health check on the hashmap database.
// Since the hashmap database is an in-memory implementation, it always returns nil.
func (db *Database) HealthCheck() error {
	db.RLock()
	defer db.RUnlock()
	if db.data == nil {
		return hord.ErrNoDial
	}

	if db.config.Filename != "" {
		_, err := os.Stat(db.config.Filename)
		if err != nil {
			return fmt.Errorf("error checking if file exists: %w", err)
		}
	}

	return nil
}

// Close closes the hashmap database connection and clears all stored data from memory (file remains if used).
func (db *Database) Close() {
	db.Lock()
	defer db.Unlock()
	db.data = nil
}

// saveToLocalFile is a helper function for methods that change the data (Set, Delete) and should
// only be used after acquiring Write lock
func (db *Database) saveToLocalFile() error {
	if db.config.Filename == "" {
		return nil
	}

	var err error
	var content []byte
	switch filepath.Ext(db.config.Filename) {
	case ".json":
		content, err = json.Marshal(db.data)
	case ".yaml", ".yml":
		content, err = yaml.Marshal(db.data)
	}
	if err != nil {
		return fmt.Errorf("error marshalling data: %w", err)
	}

	err = os.WriteFile(db.config.Filename, content, 0755)
	if err != nil {
		return fmt.Errorf("error writing data to file %q: %w", db.config.Filename, err)
	}

	return nil
}
