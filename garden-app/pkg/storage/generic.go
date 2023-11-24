package storage

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/madflojo/hord"
)

type Client struct {
	Gardens              *TypedClient[*pkg.Garden]
	Zones                *TypedClient[*pkg.Zone]
	WaterSchedules       *TypedClient[*pkg.WaterSchedule]
	WeatherClientConfigs *TypedClient[*weather.Config]
}

func NewClient(config Config) (*Client, error) {
	bc, err := NewBaseClient(config)
	if err != nil {
		return nil, fmt.Errorf("error creating base client: %w", err)
	}

	return &Client{
		Gardens:              NewTypedClient[*pkg.Garden](bc, "Garden"),
		Zones:                NewTypedClient[*pkg.Zone](bc, "Zone"),
		WaterSchedules:       NewTypedClient[*pkg.WaterSchedule](bc, "WaterSchedule"),
		WeatherClientConfigs: NewTypedClient[*weather.Config](bc, "WeatherClient"),
	}, nil
}

type Resource interface {
	babyapi.EndDateable
	// babyapi.Resource
	GetID() string
}

// Client is a wrapper around hord.Database to allow for easy interactions with resources
type TypedClient[T Resource] struct {
	prefix string
	*BaseClient
}

func NewTypedClient[T Resource](bc *BaseClient, prefix string) *TypedClient[T] {
	return &TypedClient[T]{prefix, bc}
}

func (c *TypedClient[T]) key(id string) string {
	return fmt.Sprintf("%s_%s", c.prefix, id)
}

// TODO: end-date instead of delete!
func (c *TypedClient[T]) Delete(id string) error {
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
func (c *TypedClient[T]) Get(id string) (T, error) {
	return c.get(c.key(id))
}

func (c *TypedClient[T]) get(key string) (T, error) {
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
func (c *TypedClient[T]) GetAll(filter babyapi.FilterFunc[T]) ([]T, error) {
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
func (c *TypedClient[T]) Set(item T) error {
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
