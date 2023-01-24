package server

import (
	"context"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/sirupsen/logrus"
)

type contextKey int

const (
	loggerCtxKey contextKey = iota
	gardenCtxKey
	plantCtxKey
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

func newContextWithGarden(ctx context.Context, g *pkg.Garden) context.Context {
	return context.WithValue(ctx, gardenCtxKey, g)
}

func getGardenFromContext(ctx context.Context) *pkg.Garden {
	return ctx.Value(gardenCtxKey).(*pkg.Garden)
}

func newContextWithZone(ctx context.Context, z *pkg.Zone) context.Context {
	return context.WithValue(ctx, zoneCtxKey, z)
}

func getZoneFromContext(ctx context.Context) *pkg.Zone {
	return ctx.Value(zoneCtxKey).(*pkg.Zone)
}

func newContextWithPlant(ctx context.Context, p *pkg.Plant) context.Context {
	return context.WithValue(ctx, plantCtxKey, p)
}

func getPlantFromContext(ctx context.Context) *pkg.Plant {
	return ctx.Value(plantCtxKey).(*pkg.Plant)
}
