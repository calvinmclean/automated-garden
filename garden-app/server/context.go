package server

import (
	"context"

	"github.com/sirupsen/logrus"
)

type contextKey int

const (
	loggerCtxKey contextKey = iota
	gardenCtxKey
	zoneCtxKey
)

func newContextWithLogger(ctx context.Context, logger *logrus.Entry) context.Context {
	return context.WithValue(ctx, loggerCtxKey, logger)
}

func getLoggerFromContext(ctx context.Context) *logrus.Entry {
	if logger, ok := ctx.Value(loggerCtxKey).(*logrus.Entry); ok {
		return logger
	}
	logger := logrus.New().WithField("", "")
	return logger
}

func newContextWithGarden(ctx context.Context, g *GardenResponse) context.Context {
	return context.WithValue(ctx, gardenCtxKey, g)
}

func getGardenFromContext(ctx context.Context) *GardenResponse {
	return ctx.Value(gardenCtxKey).(*GardenResponse)
}

func newContextWithZone(ctx context.Context, z *ZoneResponse) context.Context {
	return context.WithValue(ctx, zoneCtxKey, z)
}

func getZoneFromContext(ctx context.Context) *ZoneResponse {
	return ctx.Value(zoneCtxKey).(*ZoneResponse)
}
