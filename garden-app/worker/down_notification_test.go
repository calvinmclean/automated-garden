package worker

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/fake"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDowntimeThreshold = 5 * time.Minute

func TestCheckHealthMessage(t *testing.T) {
	tests := []struct {
		topic    string
		message  string
		expected bool
	}{
		{
			"/",
			`health garden=""`,
			false,
		},
		{
			"topic/data/health",
			`health garden="topic"`,
			true,
		},
		{
			"topic/suffix",
			`health garden="topic"`,
			false,
		},
		{
			"topic/data/health",
			`bad message`,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			assert.Equal(t, tt.expected, checkHealthMessage(tt.topic, tt.message))
		})
	}
}

func TestHandleHealthMessage(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	require.NoError(t, err)

	garden := &pkg.Garden{
		Name:                 "MyGarden",
		ID:                   babyapi.NewID(),
		TopicPrefix:          "garden",
		NotificationClientID: nil,
		NotificationSettings: &pkg.NotificationSettings{
			Downtime: &pkg.Duration{Duration: testDowntimeThreshold},
		},
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	require.NoError(t, err)

	clock.Reset()
	defer clock.Reset()

	t.Run("CreateNotificationClient", func(t *testing.T) {
		nc := &notifications.Client{
			ID:      babyapi.NewID(),
			Type:    "fake",
			Options: map[string]any{},
		}

		err := storageClient.NotificationClientConfigs.Set(context.Background(), nc)
		require.NoError(t, err)

		ncID := nc.GetID()
		apiErr := garden.Patch(&pkg.Garden{NotificationClientID: &ncID})
		require.Nil(t, apiErr)
		err = storageClient.Gardens.Set(context.Background(), garden)
		require.NoError(t, err)
	})

	t.Run("NotifyDueToDowntime", func(t *testing.T) {
		defer fake.Reset()
		mockClock := clock.MockTime()

		var logBuffer bytes.Buffer
		w := NewWorker(storageClient, nil, nil, slog.New(slog.NewTextHandler(&logBuffer, nil)))

		topic := "garden/data/health"
		w.handleHealthMessage(topic, `health garden="garden"`)

		mockClock.Add(testDowntimeThreshold + 1*time.Second)

		w.Stop()

		lastMessage := fake.LastMessage()
		assert.Equal(t, "MyGarden is down", lastMessage.Title)
		assert.Equal(t, "Garden has been down for > 5m0s", lastMessage.Message)

		logs := logBuffer.String()
		assert.Contains(t, logs, `msg="successfully sent down notification" source=worker topic=garden/data/health`)
		assert.Contains(t, logs, `msg="created new timer" source=worker topic=garden/data/health`)
	})

	t.Run("TimerIsReset", func(t *testing.T) {
		defer fake.Reset()
		mockClock := clock.MockTime()

		var logBuffer bytes.Buffer
		w := NewWorker(storageClient, nil, nil, slog.New(slog.NewTextHandler(&logBuffer, nil)))

		topic := "garden/data/health"

		// send initial message
		w.handleHealthMessage(topic, `health garden="garden"`)

		// send another message before threshold
		mockClock.Add(testDowntimeThreshold - 1*time.Second)
		w.handleHealthMessage(topic, `health garden="garden"`)

		// now jump forward a bit to after the initial threshold
		mockClock.Add(2 * time.Second)

		w.Stop()

		assert.Empty(t, fake.Messages())

		logs := logBuffer.String()
		assert.Contains(t, logs, `msg="reset timer" source=worker topic=garden/data/health`)
	})

	t.Run("GardenWithoutDowntime_NoNotification", func(t *testing.T) {
		defer fake.Reset()
		mockClock := clock.MockTime()

		garden := &pkg.Garden{
			Name:        "MyNewGarden",
			ID:          babyapi.NewID(),
			TopicPrefix: "new-garden",
		}
		err = storageClient.Gardens.Set(context.Background(), garden)
		require.NoError(t, err)

		var logBuffer bytes.Buffer
		w := NewWorker(storageClient, nil, nil, slog.New(slog.NewTextHandler(&logBuffer, nil)))

		topic := "new-garden/data/health"
		w.handleHealthMessage(topic, `health garden="new-garden"`)

		mockClock.Add(testDowntimeThreshold + 1*time.Second)

		w.Stop()

		assert.Empty(t, fake.Messages())

		logs := logBuffer.String()
		assert.Contains(t, logs, `msg="received message" source=worker topic=new-garden/data/health`)
	})
}
