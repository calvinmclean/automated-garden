package storage

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
	"github.com/madflojo/hord"
)

type endDateableResource interface {
	pkg.EndDateable
	babyapi.Resource
}

// genericClient is a wrapper around the base client that includes the expected type
// that it works with and has the internal key prefix used when storing
type genericClient[T endDateableResource] struct {
	prefix string
	*client
}

func newGenericClient[T endDateableResource](bc *client, prefix string) *genericClient[T] {
	return &genericClient[T]{prefix, bc}
}

func (c *genericClient[T]) key(id string) string {
	return fmt.Sprintf("%s_%s", c.prefix, id)
}

func (c *genericClient[T]) Delete(id string) error {
	key := c.key(id)

	result, err := c.get(key)
	if err != nil {
		return fmt.Errorf("error getting resource before deleting: %w", err)
	}
	if result.EndDated() {
		return c.db.Delete(key)
	}

	result.SetEndDate(time.Now())

	return c.Set(result)
}

// Get will use the provided key to read data from the data source. Then, it will Unmarshal
// into the generic type
func (c *genericClient[T]) Get(id string) (T, error) {
	return c.get(c.key(id))
}

func (c *genericClient[T]) get(key string) (T, error) {
	if c.db == nil {
		return *new(T), fmt.Errorf("error missing database connection")
	}

	dataBytes, err := c.db.Get(key)
	if err != nil {
		if errors.Is(hord.ErrNil, err) {
			return *new(T), babyapi.ErrNotFound
		}
		return *new(T), fmt.Errorf("error getting data: %w", err)
	}

	var result T
	err = c.unmarshal(dataBytes, &result)
	if err != nil {
		return *new(T), fmt.Errorf("error parsing data: %w", err)
	}

	return result, nil
}

// GetAll will use the provided prefix to read data from the data source. Then, it will use getOne
// to read each element into the correct type. These types must support `pkg.EndDateable` to allow
// excluding end-dated resources
func (c *genericClient[T]) GetAll(filter babyapi.FilterFunc[T]) ([]T, error) {
	keys, err := c.db.Keys()
	if err != nil {
		return nil, fmt.Errorf("error getting keys: %w", err)
	}

	results := []T{}
	for _, key := range keys {
		if !strings.HasPrefix(key, c.prefix) {
			continue
		}

		result, err := c.get(key)
		if err != nil {
			return nil, fmt.Errorf("error getting data: %w", err)
		}

		if filter == nil || filter(result) {
			results = append(results, result)
		}
	}

	return results, nil
}

// Set marshals the provided item and writes it to the database
func (c *genericClient[T]) Set(item T) error {
	asBytes, err := c.marshal(item)
	if err != nil {
		return fmt.Errorf("error marshalling data: %w", err)
	}

	err = c.db.Set(c.key(item.GetID()), asBytes)
	if err != nil {
		return fmt.Errorf("error writing data to database: %w", err)
	}

	return nil
}
