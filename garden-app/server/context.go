package server

import (
	"context"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/sirupsen/logrus"
)

type contextKey int

const (
	loggerCtxKey contextKey = iota
	gardenCtxKey
	plantCtxKey
	zoneCtxKey
	weatherClientCtxKey
	waterScheduleCtxKey
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

func newContextWithPlant(ctx context.Context, p *PlantResponse) context.Context {
	return context.WithValue(ctx, plantCtxKey, p)
}

func getPlantFromContext(ctx context.Context) *PlantResponse {
	return ctx.Value(plantCtxKey).(*PlantResponse)
}

func newContextWithWeatherClient(ctx context.Context, wc *weather.Config) context.Context {
	return context.WithValue(ctx, weatherClientCtxKey, wc)
}

func getWeatherClientFromContext(ctx context.Context) *weather.Config {
	return ctx.Value(weatherClientCtxKey).(*weather.Config)
}

func newContextWithWaterSchedule(ctx context.Context, ws *pkg.WaterSchedule) context.Context {
	return context.WithValue(ctx, waterScheduleCtxKey, ws)
}

func getWaterScheduleFromContext(ctx context.Context) *pkg.WaterSchedule {
	return ctx.Value(waterScheduleCtxKey).(*pkg.WaterSchedule)
}
