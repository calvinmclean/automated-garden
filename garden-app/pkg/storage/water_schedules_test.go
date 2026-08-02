package storage

import (
	"database/sql"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/db"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/require"
)

func TestDBWaterScheduleToWaterScheduleLegacyNotificationSettings(t *testing.T) {
	waterSchedule, err := dbWaterScheduleToWaterSchedule(db.WaterSchedule{
		ID: babyapi.NewID().String(),
		NotificationSettings: sql.NullString{
			String: `{"watering_reminder":1,"watering_errors":0}`,
			Valid:  true,
		},
	})
	require.NoError(t, err)
	require.True(t, waterSchedule.NotificationSettings.WateringReminder)
	require.False(t, waterSchedule.NotificationSettings.WateringErrors)
}
