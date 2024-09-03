package storage

import (
	"context"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/require"
)

func TestMigrations(t *testing.T) {
	client, err := NewClient(Config{
		Driver: "hashmap",
	})
	require.NoError(t, err)

	t.Run("StoreAllZones", func(t *testing.T) {
		zones := []*pkg.Zone{
			&pkg.Zone{
				ID:   babyapi.NewID(),
				Name: "Zone1",
			},
			&pkg.Zone{
				ID:   babyapi.NewID(),
				Name: "Zone2",
			},
			&pkg.Zone{
				ID:   babyapi.NewID(),
				Name: "Zone3",
			},
			&pkg.Zone{
				ID:   babyapi.NewID(),
				Name: "Zone4",
			},
		}

		for _, z := range zones {
			err := client.Zones.Set(context.Background(), z)
			require.NoError(t, err)
		}
	})

	t.Run("StoreAllGardens", func(t *testing.T) {
		gardens := []*pkg.Garden{
			&pkg.Garden{
				ID:   babyapi.NewID(),
				Name: "Garden1",
			},
			&pkg.Garden{
				ID:   babyapi.NewID(),
				Name: "Garden2",
			},
			&pkg.Garden{
				ID:   babyapi.NewID(),
				Name: "Garden3",
			},
			&pkg.Garden{
				ID:   babyapi.NewID(),
				Name: "Garden4",
			},
		}

		for _, g := range gardens {
			err := client.Gardens.Set(context.Background(), g)
			require.NoError(t, err)
		}
	})

	t.Run("StoreAllWaterSchedules", func(t *testing.T) {
		waterSchedules := []*pkg.WaterSchedule{
			&pkg.WaterSchedule{
				ID:   babyapi.NewID(),
				Name: "WaterSchedule1",
			},
			&pkg.WaterSchedule{
				ID:   babyapi.NewID(),
				Name: "WaterSchedule2",
			},
			&pkg.WaterSchedule{
				ID:   babyapi.NewID(),
				Name: "WaterSchedule3",
			},
			&pkg.WaterSchedule{
				ID:   babyapi.NewID(),
				Name: "WaterSchedule4",
			},
		}

		for _, ws := range waterSchedules {
			err := client.WaterSchedules.Set(context.Background(), ws)
			require.NoError(t, err)
		}
	})

	t.Run("RunMigrations", func(t *testing.T) {
		err := client.RunMigrations(context.Background())
		require.NoError(t, err)
	})

	t.Run("CheckUpdatedZoneVersion", func(t *testing.T) {
		allZones, err := client.Zones.GetAll(context.Background(), nil)
		require.NoError(t, err)
		for _, z := range allZones {
			require.Equal(t, uint(1), z.GetVersion())
		}
	})

	t.Run("CheckUpdatedGardenVersion", func(t *testing.T) {
		allGardens, err := client.Gardens.GetAll(context.Background(), nil)
		require.NoError(t, err)
		for _, g := range allGardens {
			require.Equal(t, uint(1), g.GetVersion())
		}
	})

	t.Run("CheckUpdatedWaterScheduleVersion", func(t *testing.T) {
		allWaterSchedules, err := client.WaterSchedules.GetAll(context.Background(), nil)
		require.NoError(t, err)
		for _, ws := range allWaterSchedules {
			require.Equal(t, uint(1), ws.GetVersion())
		}
	})
}
