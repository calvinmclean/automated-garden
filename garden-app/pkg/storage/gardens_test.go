package storage

import (
	"context"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGardenFanScheduleRoundTrip(t *testing.T) {
	dbPath := t.TempDir() + "/gardens_test.db"
	client, err := NewClient(Config{ConnectionString: dbPath})
	require.NoError(t, err)

	two := uint(2)
	power := uint(50)
	now := time.Now()
	g := &pkg.Garden{
		Name:        "test",
		TopicPrefix: "test",
		MaxZones:    &two,
		ID:          babyapi.NewID(),
		CreatedAt:   &now,
		FanSchedule: &pkg.FanSchedule{
			Duration: &pkg.Duration{Duration: 30 * time.Minute},
			Interval: &pkg.Duration{Duration: 2 * time.Hour},
			Power:    &power,
		},
	}

	err = client.Gardens.Set(context.Background(), g)
	require.NoError(t, err)

	g2, err := client.Gardens.Get(context.Background(), g.ID.String())
	require.NoError(t, err)

	require.NotNil(t, g2.FanSchedule, "FanSchedule should be persisted and loaded")
	assert.Equal(t, 30*time.Minute, g2.FanSchedule.Duration.Duration)
	assert.Equal(t, 2*time.Hour, g2.FanSchedule.Interval.Duration)
	assert.Equal(t, power, *g2.FanSchedule.Power)
}
