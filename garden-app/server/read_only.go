package server

import (
	"net/http"

	"github.com/calvinmclean/babyapi"
)

func readOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			logger := babyapi.GetLoggerFromContext(r.Context())
			logger.Info("received non-get request to read-only API")
			return
		}
		next.ServeHTTP(w, r)
	})
}
