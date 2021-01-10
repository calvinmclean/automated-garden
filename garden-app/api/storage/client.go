package storage

import "github.com/calvinmclean/automated-garden/garden-app/api"

// Client is a "generic" interface used to interact with our storage backend (DB, file, etc)
type Client interface {
	GetPlant(string) *api.Plant
	GetPlants() []*api.Plant
	SavePlant(*api.Plant) error
	EndDatePlant(*api.Plant) error
}
