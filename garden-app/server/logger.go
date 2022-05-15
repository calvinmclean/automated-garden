package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

const loggerCtxKey = contextKey("logger")

func loggerMiddleware(logger *logrus.Logger) func(next http.Handler) http.Handler {
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
				httpLogger.WithFields(logrus.Fields{
					"status":        ww.Status(),
					"bytes_written": ww.BytesWritten(),
					"time_elapsed":  time.Since(t1),
				}).Info("response completed")
			}()

			next.ServeHTTP(ww, r.WithContext(context.WithValue(ctx, loggerCtxKey, httpLogger)))
		}
		return http.HandlerFunc(fn)
	}
}

func contextLogger(ctx context.Context) *logrus.Entry {
	if ctx == nil {
		logger := logrus.New().WithField("", "")
		logger.Info("created new logger due to nil context")
		return logger
	}
	if logger, ok := ctx.Value(loggerCtxKey).(*logrus.Entry); ok {
		return logger
	}
	logger := logrus.New().WithField("", "")
	logger.Info("created new logger due to missing context key")
	return logger
}
