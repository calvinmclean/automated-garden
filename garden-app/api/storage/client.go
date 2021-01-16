package storage

import (
	"github.com/calvinmclean/automated-garden/garden-app/api"
	"github.com/rs/xid"
)

// Client is a "generic" interface used to interact with our storage backend (DB, file, etc)
type Client interface {
	GetPlant(xid.ID) (*api.Plant, error)
	GetPlants(bool) []*api.Plant
	SavePlant(*api.Plant) error
}
