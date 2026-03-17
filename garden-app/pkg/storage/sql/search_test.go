package sql

import (
	"context"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGardenStorageSearchWithEndDated(t *testing.T) {
	ctx := context.Background()

	sqlClient, err := NewClient(Config{
		DataSourceName: ":memory:",
	})
	require.NoError(t, err)

	gardenStorage := sqlClient.Gardens

	// Create an active garden (no end date)
	activeGarden := &pkg.Garden{
		ID:          babyapi.NewID(),
		Name:        "active-garden",
		TopicPrefix: "active",
	}
	err = gardenStorage.Set(ctx, activeGarden)
	require.NoError(t, err)

	// Create an end-dated garden
	endDate := time.Now().Add(-24 * time.Hour)
	endDatedGarden := &pkg.Garden{
		ID:          babyapi.NewID(),
		Name:        "end-dated-garden",
		TopicPrefix: "end-dated",
		EndDate:     &endDate,
	}
	err = gardenStorage.Set(ctx, endDatedGarden)
	require.NoError(t, err)

	t.Run("SearchWithoutEndDated", func(t *testing.T) {
		// Search without end_dated flag should only return active gardens
		gardens, err := gardenStorage.Search(ctx, "", nil)
		require.NoError(t, err)
		assert.Len(t, gardens, 1)
		assert.Equal(t, "active-garden", gardens[0].Name)
	})

	t.Run("SearchWithEndDated", func(t *testing.T) {
		// Search with end_dated=true should return all gardens including end-dated ones
		gardens, err := gardenStorage.Search(ctx, "", babyapi.EndDatedQueryParam(true))
		require.NoError(t, err)
		assert.Len(t, gardens, 2)

		// Verify we got both gardens
		names := make([]string, len(gardens))
		for i, g := range gardens {
			names[i] = g.Name
		}
		assert.Contains(t, names, "active-garden")
		assert.Contains(t, names, "end-dated-garden")
	})
}

func TestZoneStorageSearchWithEndDated(t *testing.T) {
	ctx := context.Background()

	sqlClient, err := NewClient(Config{
		DataSourceName: ":memory:",
	})
	require.NoError(t, err)

	gardenStorage := sqlClient.Gardens
	zoneStorage := sqlClient.Zones

	// Create a parent garden
	gardenID := babyapi.NewID()
	garden := &pkg.Garden{
		ID:          gardenID,
		Name:        "test-garden",
		TopicPrefix: "test",
	}
	err = gardenStorage.Set(ctx, garden)
	require.NoError(t, err)

	// Create an active zone (no end date)
	activeZone := &pkg.Zone{
		ID:       babyapi.NewID(),
		Name:     "active-zone",
		GardenID: gardenID.ID,
	}
	err = zoneStorage.Set(ctx, activeZone)
	require.NoError(t, err)

	// Create an end-dated zone
	endDate := time.Now().Add(-24 * time.Hour)
	endDatedZone := &pkg.Zone{
		ID:       babyapi.NewID(),
		Name:     "end-dated-zone",
		GardenID: gardenID.ID,
		EndDate:  &endDate,
	}
	err = zoneStorage.Set(ctx, endDatedZone)
	require.NoError(t, err)

	t.Run("SearchWithoutEndDated", func(t *testing.T) {
		// Search without end_dated flag should only return active zones
		zones, err := zoneStorage.Search(ctx, gardenID.String(), nil)
		require.NoError(t, err)
		assert.Len(t, zones, 1)
		assert.Equal(t, "active-zone", zones[0].Name)
	})

	t.Run("SearchWithEndDated", func(t *testing.T) {
		// Search with end_dated=true should return all zones including end-dated ones
		zones, err := zoneStorage.Search(ctx, gardenID.String(), babyapi.EndDatedQueryParam(true))
		require.NoError(t, err)
		assert.Len(t, zones, 2)

		// Verify we got both zones
		names := make([]string, len(zones))
		for i, z := range zones {
			names[i] = z.Name
		}
		assert.Contains(t, names, "active-zone")
		assert.Contains(t, names, "end-dated-zone")
	})
}

func TestWaterScheduleStorageSearchWithEndDated(t *testing.T) {
	ctx := context.Background()

	sqlClient, err := NewClient(Config{
		DataSourceName: ":memory:",
	})
	require.NoError(t, err)

	waterScheduleStorage := sqlClient.WaterSchedules

	// Create an active water schedule (no end date)
	duration := pkg.Duration{Duration: time.Hour}
	interval := pkg.Duration{Duration: 24 * time.Hour}
	activeWS := &pkg.WaterSchedule{
		ID:        babyapi.NewID(),
		Name:      "active-schedule",
		Duration:  &duration,
		Interval:  &interval,
		StartTime: pkg.NewStartTime(time.Now()),
	}
	err = waterScheduleStorage.Set(ctx, activeWS)
	require.NoError(t, err)

	// Create an end-dated water schedule
	endDate := time.Now().Add(-24 * time.Hour)
	endDatedWS := &pkg.WaterSchedule{
		ID:        babyapi.NewID(),
		Name:      "end-dated-schedule",
		Duration:  &duration,
		Interval:  &interval,
		StartTime: pkg.NewStartTime(time.Now()),
		EndDate:   &endDate,
	}
	err = waterScheduleStorage.Set(ctx, endDatedWS)
	require.NoError(t, err)

	t.Run("SearchWithoutEndDated", func(t *testing.T) {
		// Search without end_dated flag should only return active water schedules
		schedules, err := waterScheduleStorage.Search(ctx, "", nil)
		require.NoError(t, err)
		assert.Len(t, schedules, 1)
		assert.Equal(t, "active-schedule", schedules[0].Name)
	})

	t.Run("SearchWithEndDated", func(t *testing.T) {
		// Search with end_dated=true should return all water schedules including end-dated ones
		schedules, err := waterScheduleStorage.Search(ctx, "", babyapi.EndDatedQueryParam(true))
		require.NoError(t, err)
		assert.Len(t, schedules, 2)

		// Verify we got both water schedules
		names := make([]string, len(schedules))
		for i, ws := range schedules {
			names[i] = ws.Name
		}
		assert.Contains(t, names, "active-schedule")
		assert.Contains(t, names, "end-dated-schedule")
	})
}

func TestSearchEndDatedWithRFC3339Format(t *testing.T) {
	ctx := context.Background()

	sqlClient, err := NewClient(Config{
		DataSourceName: ":memory:",
	})
	require.NoError(t, err)

	gardenStorage := sqlClient.Gardens

	// Create gardens with specific end dates to test RFC3339 format parsing
	now := time.Now()

	// Garden ending in the past (should be excluded from active search)
	pastEndDate := now.Add(-48 * time.Hour)
	pastGarden := &pkg.Garden{
		ID:          babyapi.NewID(),
		Name:        "past-garden",
		TopicPrefix: "past",
		EndDate:     &pastEndDate,
	}
	err = gardenStorage.Set(ctx, pastGarden)
	require.NoError(t, err)

	// Garden ending in the future (should be included in active search)
	futureEndDate := now.Add(48 * time.Hour)
	futureGarden := &pkg.Garden{
		ID:          babyapi.NewID(),
		Name:        "future-garden",
		TopicPrefix: "future",
		EndDate:     &futureEndDate,
	}
	err = gardenStorage.Set(ctx, futureGarden)
	require.NoError(t, err)

	// Active garden (no end date)
	activeGarden := &pkg.Garden{
		ID:          babyapi.NewID(),
		Name:        "active-garden",
		TopicPrefix: "active",
	}
	err = gardenStorage.Set(ctx, activeGarden)
	require.NoError(t, err)

	t.Run("ActiveSearchExcludesPastEndDate", func(t *testing.T) {
		// Search without end_dated flag should return active garden and future-ended garden
		gardens, err := gardenStorage.Search(ctx, "", nil)
		require.NoError(t, err)
		assert.Len(t, gardens, 2)

		names := make([]string, len(gardens))
		for i, g := range gardens {
			names[i] = g.Name
		}
		assert.Contains(t, names, "active-garden")
		assert.Contains(t, names, "future-garden")
		assert.NotContains(t, names, "past-garden")
	})

	t.Run("EndDatedSearchIncludesAll", func(t *testing.T) {
		// Search with end_dated=true should return all gardens
		gardens, err := gardenStorage.Search(ctx, "", babyapi.EndDatedQueryParam(true))
		require.NoError(t, err)
		assert.Len(t, gardens, 3)

		names := make([]string, len(gardens))
		for i, g := range gardens {
			names[i] = g.Name
		}
		assert.Contains(t, names, "active-garden")
		assert.Contains(t, names, "future-garden")
		assert.Contains(t, names, "past-garden")
	})
}
