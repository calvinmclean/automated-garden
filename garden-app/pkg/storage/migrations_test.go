package storage

import (
	"context"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
)

func TestMigrations(t *testing.T) {
	client, err := NewClient(Config{
		Driver: "hashmap",
	})
	require.NoError(t, err)

	gardenID := xid.New()

	t.Run("StoreAllZones", func(t *testing.T) {
		zones := []*pkg.Zone{
			{
				ID:       babyapi.NewID(),
				Name:     "Zone1",
				GardenID: gardenID,
			},
			{
				ID:       babyapi.NewID(),
				Name:     "Zone2",
				GardenID: gardenID,
			},
			{
				ID:       babyapi.NewID(),
				Name:     "Zone3",
				GardenID: gardenID,
			},
			{
				ID:       babyapi.NewID(),
				Name:     "Zone4",
				GardenID: gardenID,
			},
		}

		for _, z := range zones {
			err := client.Zones.Set(context.Background(), z)
			require.NoError(t, err)
		}
	})

	t.Run("StoreAllGardens", func(t *testing.T) {
		gardens := []*pkg.Garden{
			{
				ID:   babyapi.ID{ID: gardenID},
				Name: "Garden1",
			},
			{
				ID:   babyapi.NewID(),
				Name: "Garden2",
			},
			{
				ID:   babyapi.NewID(),
				Name: "Garden3",
			},
			{
				ID:                   babyapi.NewID(),
				Name:                 "GardenWithNotificationClient",
				NotificationClientID: pointer("client_id"),
			},
		}

		for _, g := range gardens {
			err := client.Gardens.Set(context.Background(), g)
			require.NoError(t, err)
		}
	})

	t.Run("StoreAllWaterSchedules", func(t *testing.T) {
		waterSchedules := []*pkg.WaterSchedule{
			{
				ID:   babyapi.NewID(),
				Name: "WaterSchedule1",
			},
			{
				ID:   babyapi.NewID(),
				Name: "WaterSchedule2",
			},
			{
				ID:   babyapi.NewID(),
				Name: "WaterSchedule3",
			},
			{
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
		allZones, err := client.Zones.Search(context.Background(), gardenID.String(), nil)
		require.NoError(t, err)
		for _, z := range allZones {
			require.Equal(t, uint(1), z.GetVersion())
		}
	})

	t.Run("CheckUpdatedGardenVersion", func(t *testing.T) {
		allGardens, err := client.Gardens.Search(context.Background(), "", nil)
		require.NoError(t, err)
		for _, g := range allGardens {
			require.Equal(t, uint(2), g.GetVersion())

			if g.Name == "GardenWithNotificationClient" {
				t.Run("GardenWithNotificationClientHasSettingsTrue_Migration2", func(t *testing.T) {
					require.True(t, g.GetNotificationSettings().ControllerStartup)
					require.True(t, g.GetNotificationSettings().LightSchedule)
				})
			}
		}
	})

	t.Run("CheckUpdatedWaterScheduleVersion", func(t *testing.T) {
		allWaterSchedules, err := client.WaterSchedules.Search(context.Background(), "", nil)
		require.NoError(t, err)
		for _, ws := range allWaterSchedules {
			require.Equal(t, uint(1), ws.GetVersion())
		}
	})
}

func pointer[T any](v T) *T {
	return &v
}
