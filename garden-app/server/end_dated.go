package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"
)

// endDateable is a simple interface that requires a method to determine if something is end-dated
type endDateable interface {
	EndDated() bool
}

// restrictEndDatedMiddleware will get an EedDateable resource from context and return an error if it is end-dated
func restrictEndDatedMiddleware(resourceName string, ctxKey contextKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resource := r.Context().Value(ctxKey).(endDateable)
			logger := getLoggerFromContext(r.Context())

			if resource.EndDated() {
				err := fmt.Errorf("resource not available for end-dated %s", resourceName)
				logger.WithError(err).Error("unable to complete request")
				render.Render(w, r, ErrInvalidRequest(err))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
