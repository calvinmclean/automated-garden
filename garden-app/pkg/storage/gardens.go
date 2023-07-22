package storage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/madflojo/hord"
	"github.com/rs/xid"
)

const gardenPrefix = "Garden_"

// GetGarden ...
func (c *Client) GetGarden(id xid.ID) (*pkg.Garden, error) {
	return c.getGarden(gardenPrefix + id.String())
}

// GetGardens ...
func (c *Client) GetGardens(getEndDated bool) ([]*pkg.Garden, error) {
	keys, err := c.db.Keys()
	if err != nil {
		return nil, fmt.Errorf("error getting keys: %w", err)
	}

	results := []*pkg.Garden{}
	for _, key := range keys {
		if !strings.HasPrefix(key, gardenPrefix) {
			continue
		}

		result, err := c.getGarden(key)
		if err != nil {
			return nil, fmt.Errorf("error getting Garden: %w", err)
		}

		if getEndDated || !result.EndDated() {
			results = append(results, result)
		}
	}

	return results, nil
}

// SaveGarden ...
func (c *Client) SaveGarden(g *pkg.Garden) error {
	asBytes, err := c.marshal(g)
	if err != nil {
		return fmt.Errorf("error marshalling Garden: %w", err)
	}

	err = c.db.Set(gardenPrefix+g.ID.String(), asBytes)
	if err != nil {
		return fmt.Errorf("error writing Garden to database: %w", err)
	}

	return nil
}

// DeleteGarden ...
func (c *Client) DeleteGarden(id xid.ID) error {
	return c.db.Delete(gardenPrefix + id.String())
}

func (c *Client) getGarden(key string) (*pkg.Garden, error) {
	dataBytes, err := c.db.Get(key)
	if err != nil {
		if errors.Is(hord.ErrNil, err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting Garden: %w", err)
	}

	var result pkg.Garden
	err = c.unmarshal(dataBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("error parsing Garden data: %w", err)
	}

	return &result, nil
}
