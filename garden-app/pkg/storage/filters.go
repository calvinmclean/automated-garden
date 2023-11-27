package storage

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
)

func FilterEndDated[T pkg.EndDateable](getEndDated bool) babyapi.FilterFunc[T] {
	return func(item T) bool {
		return getEndDated || !item.EndDated()
	}
}
