package server

import (
	"net/http"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
)

func EndDatedFilter[T pkg.EndDateable](r *http.Request) babyapi.FilterFunc[T] {
	getEndDated := r.URL.Query().Get("end_dated") == "true"
	return storage.FilterEndDated[T](getEndDated)
}
