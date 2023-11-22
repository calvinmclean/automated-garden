package server

import (
	"fmt"
	"net/http"

	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
)

// restrictEndDatedMiddleware will get an EedDateable resource from context and return an error if it is end-dated
func restrictEndDatedMiddleware(resourceName string, ctxKey contextKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resource := r.Context().Value(ctxKey).(babyapi.EndDateable)
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
