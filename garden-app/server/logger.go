package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

func loggerMiddleware(logger *logrus.Entry) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			ctx := r.Context()

			httpLogger := logger.WithFields(logrus.Fields{
				"method":     r.Method,
				"path":       r.RequestURI,
				"host":       r.Host,
				"from":       r.RemoteAddr,
				"request_id": middleware.GetReqID(ctx),
			})

			t1 := time.Now()
			defer func() {
				if r.URL.Path == "/metrics" {
					return
				}
				httpLogger.WithFields(logrus.Fields{
					"status":        ww.Status(),
					"bytes_written": ww.BytesWritten(),
					"time_elapsed":  time.Since(t1),
				}).Info("response completed")
			}()

			next.ServeHTTP(ww, r.WithContext(newContextWithLogger(ctx, httpLogger)))
		}
		return http.HandlerFunc(fn)
	}
}
