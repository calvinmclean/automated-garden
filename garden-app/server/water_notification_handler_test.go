package server

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
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

func TestHandleMessage(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	require.NoError(t, err)

	handler := NewWaterNotificationHandler(storageClient, slog.Default())

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
		ID:                   babyapi.NewID(),
		TopicPrefix:          "garden",
		NotificationClientID: nil,
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	require.NoError(t, err)

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

	t.Run("CreateNotificationClient", func(t *testing.T) {
		nc := &notifications.Client{
			ID:   babyapi.NewID(),
			Type: "pushover",
			Options: map[string]any{
				"app_token":       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"recipient_token": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		}

		err := storageClient.NotificationClientConfigs.Set(context.Background(), nc)
		require.NoError(t, err)

		ncID := nc.GetID()
		apiErr := garden.Patch(&pkg.Garden{NotificationClientID: &ncID})
		require.Nil(t, apiErr)
		err = storageClient.Gardens.Set(context.Background(), garden)
		require.NoError(t, err)
	})

	t.Run("ErrorGettingZone", func(t *testing.T) {
		err = handler.handle("garden/data/water", []byte("water,zone=1 millis=6000"))
		require.Error(t, err)
		require.Equal(t, "error getting zone with position 1: no zone found", err.Error())
	})

	t.Run("ErrorUsingPushover", func(t *testing.T) {
		r, err := recorder.New("testdata/fixtures/pushover_fail")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			require.NoError(t, r.Stop())
		}()

		if r.Mode() != recorder.ModeRecordOnce {
			t.Fatal("Recorder should be in ModeRecordOnce")
		}

		// github.com/gregdel/pushover uses http.DefaultClient
		http.DefaultClient = r.GetDefaultClient()

		err = handler.handle("garden/data/water", []byte("water,zone=0 millis=6000"))
		require.Error(t, err)
		require.Equal(t, "Errors:\napplication token is invalid, see https://pushover.net/api", err.Error())
	})

	t.Run("Success", func(t *testing.T) {
		numMessages := 0

		r, err := recorder.New(
			"testdata/fixtures/pushover_success",
			recorder.WithHook(func(i *cassette.Interaction) error {
				// Use hook to count number of message requests
				if i.Request.URL == "https://api.pushover.net/1/messages.json" {
					numMessages++
				}
				return nil
			}, recorder.BeforeResponseReplayHook),
		)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			require.NoError(t, r.Stop())
		}()

		if r.Mode() != recorder.ModeRecordOnce {
			t.Fatal("Recorder should be in ModeRecordOnce")
		}

		// github.com/gregdel/pushover uses http.DefaultClient
		http.DefaultClient = r.GetDefaultClient()

		err = handler.handle("garden/data/water", []byte("water,zone=0 millis=6000"))
		require.NoError(t, err)

		// ensure a message is sent by API
		require.Equal(t, 1, numMessages)
	})
}
