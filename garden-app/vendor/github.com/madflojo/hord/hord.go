/*
Package hord provides a simple and extensible interface for interacting with various database systems in a uniform way.

# Overview

Hord is designed to be a database-agnostic library that provides a common interface for interacting with different database systems. It allows developers to write code that is decoupled from the underlying database technology, making it easier to switch between databases or support multiple databases in the same application.

# Usage

To use Hord, import it as follows:

	import "github.com/madflojo/hord"

# Creating a Database Client

To create a database client, you need to import and use the appropriate driver package along with the `hord` package.

For example, to use the Redis driver:

	import (
	    "github.com/madflojo/hord"
	    "github.com/madflojo/hord/redis"
	)

	func main() {
	    var db hord.Database
	    db, err := redis.Dial(redis.Config{})
	    if err != nil {
	        // Handle connection error
	    }

	    // Use the db client for database operations
	    // ...
	}

Each driver provides its own `Dial` function to establish a connection to the database. Refer to the specific driver documentation for more details.

# Database Operations

Once you have a database client, you can use it to perform various database operations. The API is consistent across different drivers.

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

Refer to the `hord.Database` interface documentation for a complete list of available methods.

# Error Handling

Hord provides common error types and constants for consistent error handling across drivers. Refer to the `hord` package documentation for more information on error handling.

# Contributing

Contributions to Hord are welcome! If you want to add support for a new database driver or improve the existing codebase, please refer to the contribution guidelines in the project's repository.
*/
package hord

import "fmt"

// Database is an interface that is used to create a unified database access object.
type Database interface {
	// Setup is used to setup and configure the underlying database.
	// This can include setting optimal cluster settings, creating a database or tablespace,
	// or even creating the database structure.
	// Setup is meant to allow users to start with a fresh database service and turn it into a production-ready datastore.
	Setup() error

	// HealthCheck performs a check against the underlying database.
	// If any errors are returned, this health check will return an error.
	// An error returned from HealthCheck should be treated as the database service being untrustworthy.
	HealthCheck() error

	// Get is used to fetch data with the provided key.
	Get(key string) ([]byte, error)

	// Set is used to insert and update the specified key.
	// This function can be used on existing keys, with the new data overwriting existing data.
	Set(key string, data []byte) error

	// Delete will delete the data for the specified key.
	Delete(key string) error

	// Keys will return a list of keys for the entire database.
	// This operation can be expensive, use with caution.
	Keys() ([]string, error)

	// Close will close the database connection.
	// After executing close, all other functions should return an error.
	Close()
}

// Common Errors Used by Hord Drivers
var (
	ErrInvalidKey  = fmt.Errorf("Key cannot be nil")
	ErrInvalidData = fmt.Errorf("Data cannot be empty")
	ErrNil         = fmt.Errorf("Nil value returned from database")
	ErrNoDial      = fmt.Errorf("No database connection defined, did you dial?")
)

// ValidKey checks if a key is valid.
// A valid key should have a length greater than 0.
// Returns nil if the key is valid, otherwise returns ErrInvalidKey.
func ValidKey(key string) error {
	if len(key) > 0 {
		return nil
	}
	return ErrInvalidKey
}

// ValidData checks if data is valid.
// Valid data should have a length greater than 0.
// Returns nil if the data is valid, otherwise returns ErrInvalidData.
func ValidData(data []byte) error {
	if len(data) > 0 {
		return nil
	}
	return ErrInvalidData
}
