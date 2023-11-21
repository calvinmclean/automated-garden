package server

import (
	"context"

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

func newContextWithWaterSchedule(ctx context.Context, ws *WaterScheduleResponse) context.Context {
	return context.WithValue(ctx, waterScheduleCtxKey, ws)
}

func getWaterScheduleFromContext(ctx context.Context) *WaterScheduleResponse {
	return ctx.Value(waterScheduleCtxKey).(*WaterScheduleResponse)
}
