package server

import (
	"context"
	"net/http"

	"github.com/go-chi/render"
)

func get[T render.Renderer](getter func(context.Context) T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := getLoggerFromContext(r.Context())

		resource := getter(r.Context())
		logger.Debugf("responding with resource: %+v", resource)

		if err := render.Render(w, r, resource); err != nil {
			logger.WithError(err).Error("unable to render response")
			render.Render(w, r, ErrRender(err))
		}
	}
}
