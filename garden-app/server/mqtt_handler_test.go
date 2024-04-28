package server

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/require"
)

func TestParseWaterMessage(t *testing.T) {
	tests := []struct {
		in            string
		expectedPos   int
		waterDuration time.Duration
	}{
		{
			"water,zone=1 millis=6000",
			1, 6000 * time.Millisecond,
		},
		{
			"water,zone=100 millis=1",
			100, 1 * time.Millisecond,
		},
		{
			"water,zone=0 millis=0",
			0, 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			zonePosition, waterDuration, err := parseWaterMessage([]byte(tt.in))
			require.NoError(t, err)
			require.Equal(t, tt.expectedPos, zonePosition)
			require.Equal(t, tt.waterDuration, waterDuration)
		})
	}
}

func TestHandle(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	require.NoError(t, err)

	handler := NewMQTTHandler(storageClient, slog.Default())

	t.Run("ErrorParsingMessage", func(t *testing.T) {
		err = handler.handle("garden/data/water", []byte{})
		require.Error(t, err)
		require.Equal(t, `error parsing message: error parsing zone position: invalid integer: strconv.Atoi: parsing "": invalid syntax`, err.Error())
	})

	t.Run("ErrorGettingGarden", func(t *testing.T) {
		err = handler.handle("garden/data/water", []byte("water,zone=0 millis=6000"))
		require.Error(t, err)
		require.Equal(t, "error getting garden with topic-prefix \"garden\": no garden found", err.Error())
	})

	garden := &pkg.Garden{
		ID:          babyapi.NewID(),
		TopicPrefix: "garden",
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	require.NoError(t, err)

	t.Run("ErrorGettingZone", func(t *testing.T) {
		err = handler.handle("garden/data/water", []byte("water,zone=0 millis=6000"))
		require.Error(t, err)
		require.Equal(t, "error getting zone with position 0: no zone found", err.Error())
	})

	zero := uint(0)
	zone := &pkg.Zone{
		ID:       babyapi.NewID(),
		GardenID: garden.ID.ID,
		Position: &zero,
	}
	err = storageClient.Zones.Set(context.Background(), zone)
	require.NoError(t, err)

	t.Run("SuccessfulWithNoNotificationClients", func(t *testing.T) {
		err = handler.handle("garden/data/water", []byte("water,zone=0 millis=6000"))
		require.NoError(t, err)
	})
}
