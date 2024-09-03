package storage

import (
	"context"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/migrate"
)

var (
	zoneMigrations = []migrate.Migration{
		migrate.NewMigration("InitializeVersion", func(z *pkg.Zone) (*pkg.Zone, error) {
			if z.V == 0 {
				z.V = 1
			}
			return z, nil
		}),
	}

	gardenMigrations = []migrate.Migration{
		migrate.NewMigration("InitializeVersion", func(g *pkg.Garden) (*pkg.Garden, error) {
			if g.V == 0 {
				g.V = 1
			}
			return g, nil
		}),
	}

	waterScheduleMigrations = []migrate.Migration{
		migrate.NewMigration("InitializeVersion", func(ws *pkg.WaterSchedule) (*pkg.WaterSchedule, error) {
			if ws.V == 0 {
				ws.V = 1
			}
			return ws, nil
		}),
	}
)

func (c *Client) RunMigrations(ctx context.Context) error {
	err := c.RunZoneMigrations(ctx)
	if err != nil {
		return fmt.Errorf("error running Zone migrations: %w", err)
	}

	err = c.RunGardenMigrations(ctx)
	if err != nil {
		return fmt.Errorf("error running Garden migrations: %w", err)
	}

	err = c.RunWaterScheduleMigrations(ctx)
	if err != nil {
		return fmt.Errorf("error running WaterSchedule migrations: %w", err)
	}

	return nil
}

func (c *Client) RunZoneMigrations(ctx context.Context) error {
	zones, err := c.Zones.GetAll(ctx, nil)
	if err != nil {
		return fmt.Errorf("error getting all Zones: %w", err)
	}

	for zone, err := range migrate.Each[*pkg.Zone, *pkg.Zone](zoneMigrations, zones) {
		if err != nil {
			return fmt.Errorf("error migrating Zone: %w", err)
		}

		err = c.Zones.Set(ctx, zone)
		if err != nil {
			return fmt.Errorf("error storing migrated Zone: %w", err)
		}
	}

	return nil
}

func (c *Client) RunGardenMigrations(ctx context.Context) error {
	gardens, err := c.Gardens.GetAll(ctx, nil)
	if err != nil {
		return fmt.Errorf("error getting all Gardens: %w", err)
	}

	for garden, err := range migrate.Each[*pkg.Garden, *pkg.Garden](gardenMigrations, gardens) {
		if err != nil {
			return fmt.Errorf("error migrating Garden: %w", err)
		}

		err = c.Gardens.Set(ctx, garden)
		if err != nil {
			return fmt.Errorf("error storing migrated Garden: %w", err)
		}
	}

	return nil
}

func (c *Client) RunWaterScheduleMigrations(ctx context.Context) error {
	waterSchedules, err := c.WaterSchedules.GetAll(ctx, nil)
	if err != nil {
		return fmt.Errorf("error getting all WaterSchedules: %w", err)
	}

	for waterSchedule, err := range migrate.Each[*pkg.WaterSchedule, *pkg.WaterSchedule](waterScheduleMigrations, waterSchedules) {
		if err != nil {
			return fmt.Errorf("error migrating WaterSchedule: %w", err)
		}

		err = c.WaterSchedules.Set(ctx, waterSchedule)
		if err != nil {
			return fmt.Errorf("error storing migrated WaterSchedule: %w", err)
		}
	}

	return nil
}
