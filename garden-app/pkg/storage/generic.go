package storage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/madflojo/hord"
)

// getOne will use the provided key to read data from the data source. Then, it will Unmarshal
// into the generic type
func getOne[T any](c *Client, key string) (*T, error) {
	dataBytes, err := c.db.Get(key)
	if err != nil {
		if errors.Is(hord.ErrNil, err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting data: %w", err)
	}

	var result T
	err = c.unmarshal(dataBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing data: %w", err)
	}

	return &result, nil
}

// getMultiple will use the provided prefix to read data from the data source. Then, it will use getOne
// to read each element into the correct type. These types must support `pkg.EndDateable` to allow
// excluding end-dated resources
func getMultiple[T pkg.EndDateable](c *Client, getEndDated bool, prefix string) ([]T, error) {
	keys, err := c.db.Keys()
	if err != nil {
		return nil, fmt.Errorf("error getting keys: %w", err)
	}

	results := []T{}
	for _, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		result, err := getOne[T](c, key)
		if err != nil {
			return nil, fmt.Errorf("error getting data: %w", err)
		}
		if result == nil {
			continue
		}

		if getEndDated || !(*result).EndDated() {
			results = append(results, *result)
		}
	}

	return results, nil
}

// save marshals the provided item and writes it to the database
func save[T any](c *Client, item T, key string) error {
	asBytes, err := c.marshal(item)
	if err != nil {
		return fmt.Errorf("error marshalling data: %w", err)
	}

	err = c.db.Set(key, asBytes)
	if err != nil {
		return fmt.Errorf("error writing data to database: %w", err)
	}

	return nil
}
