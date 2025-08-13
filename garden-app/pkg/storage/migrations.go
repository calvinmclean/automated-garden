package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/migrate"
	"github.com/calvinmclean/babyapi"
)

var (
	zoneMigrations = []migrate.Migration{
		migrate.NewMigration("InitializeVersion1", func(z *pkg.Zone) (*pkg.Zone, error) {
			return z, nil
		}),
	}

	gardenMigrations = []migrate.Migration{
		migrate.NewMigration("InitializeVersion1", func(g *pkg.Garden) (*pkg.Garden, error) {
			return g, nil
		}),
		migrate.NewMigration("EnableNotificationsIfClientIsSet", func(g *pkg.Garden) (*pkg.Garden, error) {
			if g.NotificationClientID != nil {
				g.NotificationSettings = &pkg.NotificationSettings{
					ControllerStartup: true,
					LightSchedule:     true,
				}
			}
			return g, nil
		}),
	}

	waterScheduleMigrations = []migrate.Migration{
		migrate.NewMigration("InitializeVersion1", func(ws *pkg.WaterSchedule) (*pkg.WaterSchedule, error) {
			return ws, nil
		}),
	}
)

func (c *Client) RunMigrations(ctx context.Context) error {
	err := c.RunGardenAndZoneMigrations(ctx)
	if err != nil {
		return fmt.Errorf("error running Garden migrations: %w", err)
	}

	err = c.RunWaterScheduleMigrations(ctx)
	if err != nil {
		return fmt.Errorf("error running WaterSchedule migrations: %w", err)
	}

	return nil
}

func (c *Client) RunZoneMigrations(ctx context.Context, g *pkg.Garden) error {
	zones, err := c.Zones.Search(ctx, g.GetID(), babyapi.EndDatedQueryParam(true))
	if err != nil {
		return fmt.Errorf("error getting all Zones: %w", err)
	}

	for zone, err := range migrate.Each[*pkg.Zone, *pkg.Zone](zoneMigrations, zones) {
		if err != nil {
			if errors.Is(err, migrate.ErrNotFound) {
				continue
			}
			return fmt.Errorf("error migrating Zone: %w", err)
		}

		err = c.Zones.Set(ctx, zone)
		if err != nil {
			return fmt.Errorf("error storing migrated Zone: %w", err)
		}
	}

	return nil
}

func (c *Client) RunGardenAndZoneMigrations(ctx context.Context) error {
	gardens, err := c.Gardens.Search(ctx, "", babyapi.EndDatedQueryParam(true))
	if err != nil {
		return fmt.Errorf("error getting all Gardens: %w", err)
	}

	for garden, err := range migrate.Each[*pkg.Garden, *pkg.Garden](gardenMigrations, gardens) {
		if err != nil {
			if errors.Is(err, migrate.ErrNotFound) {
				continue
			}
			return fmt.Errorf("error migrating Garden: %w", err)
		}

		err := c.RunZoneMigrations(ctx, garden)
		if err != nil {
			return fmt.Errorf("error running Zone migrations: %w", err)
		}

		err = c.Gardens.Set(ctx, garden)
		if err != nil {
			return fmt.Errorf("error storing migrated Garden: %w", err)
		}
	}

	return nil
}

func (c *Client) RunWaterScheduleMigrations(ctx context.Context) error {
	waterSchedules, err := c.WaterSchedules.Search(ctx, "", babyapi.EndDatedQueryParam(true))
	if err != nil {
		return fmt.Errorf("error getting all WaterSchedules: %w", err)
	}

	for waterSchedule, err := range migrate.Each[*pkg.WaterSchedule, *pkg.WaterSchedule](waterScheduleMigrations, waterSchedules) {
		if err != nil {
			if errors.Is(err, migrate.ErrNotFound) {
				continue
			}
			return fmt.Errorf("error migrating WaterSchedule: %w", err)
		}

		err = c.WaterSchedules.Set(ctx, waterSchedule)
		if err != nil {
			return fmt.Errorf("error storing migrated WaterSchedule: %w", err)
		}
	}

	return nil
}
