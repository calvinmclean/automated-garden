package server

import (
	"context"

	"github.com/sirupsen/logrus"
)

type contextKey int

const (
	loggerCtxKey contextKey = iota
	gardenCtxKey
)

func newContextWithLogger(ctx context.Context, logger *logrus.Entry) context.Context {
	return context.WithValue(ctx, loggerCtxKey, logger)
}

// TODO: REMOVE
func getLoggerFromContext(ctx context.Context) *logrus.Entry {
	if logger, ok := ctx.Value(loggerCtxKey).(*logrus.Entry); ok {
		return logger
	}
	logger := logrus.New().WithField("", "")
	return logger
}
