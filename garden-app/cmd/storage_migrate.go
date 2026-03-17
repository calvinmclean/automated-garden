package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
	"github.com/spf13/cobra"
)

var storageMigrateCommand = &cobra.Command{
	Use:   "storage-migrate",
	Short: "Migrate all resources from KV storage to SQL storage",
	RunE: func(cmd *cobra.Command, _ []string) error {
		slog.Info("Starting storage migration from KV to SQL")

		// Parse source (KV) storage config
		sourceDriver, _ := cmd.Flags().GetString("source-driver")
		sourceDSN, _ := cmd.Flags().GetString("source-dsn")
		sourceFilename, _ := cmd.Flags().GetString("source-filename")

		sourceConfig := storage.Config{
			Driver:  sourceDriver,
			Options: map[string]any{},
		}

		if sourceDSN != "" {
			sourceConfig.Options["data_source_name"] = sourceDSN
		}
		if sourceFilename != "" {
			sourceConfig.Options["filename"] = sourceFilename
		}

		// Parse destination (SQL) storage config
		destDSN, _ := cmd.Flags().GetString("dest-dsn")

		destConfig := storage.Config{
			Driver: "sqlite",
			Options: map[string]any{
				"data_source_name":   destDSN,
				"disable_migrations": false,
			},
		}

		// Create source (KV) client
		sourceClient, err := storage.NewClient(sourceConfig)
		if err != nil {
			return fmt.Errorf("error creating source storage client: %w", err)
		}

		// Create destination (SQL) client
		destClient, err := storage.NewClient(destConfig)
		if err != nil {
			return fmt.Errorf("error creating destination storage client: %w", err)
		}

		ctx := context.Background()

		// Migrate Gardens
		slog.Info("Migrating Gardens")
		gardens, err := sourceClient.Gardens.Search(ctx, "", babyapi.EndDatedQueryParam(true))
		if err != nil {
			return fmt.Errorf("error getting Gardens from source: %w", err)
		}
		slog.Info(fmt.Sprintf("Found %d Gardens to migrate", len(gardens)))
		for _, g := range gardens {
			if err := destClient.Gardens.Set(ctx, g); err != nil {
				return fmt.Errorf("error saving Garden to destination: %w", err)
			}
		}
		slog.Info(fmt.Sprintf("Successfully migrated %d Gardens", len(gardens)))

		// Migrate Zones (need to get per-garden since zones are nested under gardens)
		slog.Info("Migrating Zones")
		zoneCount := 0
		for _, g := range gardens {
			zones, err := sourceClient.Zones.Search(ctx, g.GetID(), babyapi.EndDatedQueryParam(true))
			if err != nil {
				return fmt.Errorf("error getting Zones for garden %s: %w", g.GetID(), err)
			}
			for _, z := range zones {
				if err := destClient.Zones.Set(ctx, z); err != nil {
					return fmt.Errorf("error saving Zone to destination: %w", err)
				}
			}
			zoneCount += len(zones)
		}
		slog.Info(fmt.Sprintf("Successfully migrated %d Zones", zoneCount))

		// Migrate WaterSchedules
		slog.Info("Migrating WaterSchedules")
		waterSchedules, err := sourceClient.WaterSchedules.Search(ctx, "", babyapi.EndDatedQueryParam(true))
		if err != nil {
			return fmt.Errorf("error getting WaterSchedules from source: %w", err)
		}
		slog.Info(fmt.Sprintf("Found %d WaterSchedules to migrate", len(waterSchedules)))
		for _, ws := range waterSchedules {
			if err := destClient.WaterSchedules.Set(ctx, ws); err != nil {
				return fmt.Errorf("error saving WaterSchedule to destination: %w", err)
			}
		}
		slog.Info(fmt.Sprintf("Successfully migrated %d WaterSchedules", len(waterSchedules)))

		// Migrate WeatherClientConfigs
		slog.Info("Migrating WeatherClientConfigs")
		weatherClients, err := sourceClient.WeatherClientConfigs.Search(ctx, "", babyapi.EndDatedQueryParam(true))
		if err != nil {
			return fmt.Errorf("error getting WeatherClientConfigs from source: %w", err)
		}
		slog.Info(fmt.Sprintf("Found %d WeatherClientConfigs to migrate", len(weatherClients)))
		for _, wc := range weatherClients {
			if err := destClient.WeatherClientConfigs.Set(ctx, wc); err != nil {
				return fmt.Errorf("error saving WeatherClientConfig to destination: %w", err)
			}
		}
		slog.Info(fmt.Sprintf("Successfully migrated %d WeatherClientConfigs", len(weatherClients)))

		// Migrate NotificationClientConfigs
		slog.Info("Migrating NotificationClientConfigs")
		notificationClients, err := sourceClient.NotificationClientConfigs.Search(ctx, "", babyapi.EndDatedQueryParam(true))
		if err != nil {
			return fmt.Errorf("error getting NotificationClientConfigs from source: %w", err)
		}
		slog.Info(fmt.Sprintf("Found %d NotificationClientConfigs to migrate", len(notificationClients)))
		for _, nc := range notificationClients {
			if err := destClient.NotificationClientConfigs.Set(ctx, nc); err != nil {
				return fmt.Errorf("error saving NotificationClientConfig to destination: %w", err)
			}
		}
		slog.Info(fmt.Sprintf("Successfully migrated %d NotificationClientConfigs", len(notificationClients)))

		// Migrate WaterRoutines
		slog.Info("Migrating WaterRoutines")
		waterRoutines, err := sourceClient.WaterRoutines.Search(ctx, "", babyapi.EndDatedQueryParam(true))
		if err != nil {
			return fmt.Errorf("error getting WaterRoutines from source: %w", err)
		}
		slog.Info(fmt.Sprintf("Found %d WaterRoutines to migrate", len(waterRoutines)))
		for _, wr := range waterRoutines {
			if err := destClient.WaterRoutines.Set(ctx, wr); err != nil {
				return fmt.Errorf("error saving WaterRoutine to destination: %w", err)
			}
		}
		slog.Info(fmt.Sprintf("Successfully migrated %d WaterRoutines", len(waterRoutines)))

		slog.Info("Storage migration completed successfully")
		return nil
	},
}

func init() {
	storageMigrateCommand.Flags().String("source-driver", "hashmap", "Source storage driver (hashmap or redis)")
	storageMigrateCommand.Flags().String("source-dsn", "", "Source Redis DSN (e.g., 'localhost:6379')")
	storageMigrateCommand.Flags().String("source-filename", "", "Source hashmap filename (e.g., 'storage.json')")
	storageMigrateCommand.Flags().String("dest-dsn", "garden.db", "Destination SQLite DSN")
}
