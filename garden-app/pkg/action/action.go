package action

import "github.com/calvinmclean/automated-garden/garden-app/pkg"

// Action is an interface that wraps the Execute method and allows running various actions
type Action interface {
	Execute(*pkg.Garden, *pkg.Zone, Scheduler)
}
