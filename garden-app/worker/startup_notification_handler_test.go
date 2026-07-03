package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetGardenAndSendStartupMessage(t *testing.T) {
	// When a GardenController reboots, the light probably turned off. If the LightSchedule shows it should be on,
	// turn it on
	c := clock.MockTime()
	defer clock.Reset()
	now := c.Now()

	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	require.NoError(t, err)

	garden := &pkg.Garden{
		ID:          babyapi.NewID(),
		TopicPrefix: "garden",
		Name:        "garden",
		// This light scheduled turned on 3 hours ago and should still be on due to the 12 hour duration
		LightSchedule: &pkg.LightSchedule{
			Duration: &pkg.Duration{Duration: 12 * time.Hour},
			StartTime: &pkg.StartTime{
				Time: now.Add(-3 * time.Hour),
			},
		},
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	require.NoError(t, err)

	mqttClient := new(mqtt.MockClient)
	w := NewWorker(storageClient, nil, mqttClient, slog.Default())

	t.Run("LightTurnsOn", func(t *testing.T) {
		mqttClient.On("Publish", mock.Anything, "garden/command/light", []byte(`{"state":"ON"}`)).Return(nil)
		err = w.getGardenAndSendStartupMessage("garden/data/logs", "logs message=\"garden-controller setup complete\"")
		require.NoError(t, err)
		mqttClient.AssertExpectations(t)
	})

	t.Run("LightTurnsOff", func(t *testing.T) {
		c.Add(12 * time.Hour)
		fmt.Println("LightTime", garden.LightSchedule.StartTime.Time)
		fmt.Println("Now", clock.Now())
		fmt.Println(garden.LightSchedule.NextChange(c.Now()))
		mqttClient.On("Publish", mock.Anything, "garden/command/light", []byte(`{"state":"OFF"}`)).Return(nil)
		err = w.getGardenAndSendStartupMessage("garden/data/logs", "logs message=\"garden-controller setup complete\"")
		require.NoError(t, err)
		mqttClient.AssertExpectations(t)
	})

	t.Run("Shutdown", func(t *testing.T) {
		mqttClient.On("Disconnect", uint(100)).Return()
		w.Stop()
		mqttClient.AssertExpectations(t)
	})
}

func TestSetExpectedFanState(t *testing.T) {
	c := clock.MockTime()
	defer clock.Reset()

	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	require.NoError(t, err)

	power := uint(50)
	garden := &pkg.Garden{
		ID:          babyapi.NewID(),
		TopicPrefix: "garden",
		Name:        "garden",
		FanSchedule: &pkg.FanSchedule{
			Duration: &pkg.Duration{Duration: 30 * time.Minute},
			Interval: &pkg.Duration{Duration: 2 * time.Hour},
			Power:    &power,
		},
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	require.NoError(t, err)

	mqttClient := new(mqtt.MockClient)
	w := NewWorker(storageClient, nil, mqttClient, slog.Default())

	t.Run("FanTurnsOn", func(t *testing.T) {
		// At mock start time (10:00), the fan cycle just started, so 30 minutes remain
		expectedMsg, _ := json.Marshal(&action.FanAction{
			Duration: (30 * time.Minute).Milliseconds(),
			Power:    garden.FanSchedule.PowerToPWM(),
		})
		mqttClient.On("Publish", mock.Anything, "garden/command/fan", expectedMsg).Return(nil)
		err = w.setExpectedFanState(context.Background(), garden)
		require.NoError(t, err)
		mqttClient.AssertExpectations(t)
	})

	t.Run("FanTurnsOff", func(t *testing.T) {
		// 1 hour later the fan is in the OFF portion of the cycle
		c.Add(time.Hour)
		err = w.setExpectedFanState(context.Background(), garden)
		require.NoError(t, err)
		mqttClient.AssertExpectations(t)
	})

	t.Run("Shutdown", func(t *testing.T) {
		mqttClient.On("Disconnect", uint(100)).Return()
		w.Stop()
		mqttClient.AssertExpectations(t)
	})
}

func TestParseStartupMessage(t *testing.T) {
	input := "logs message=\"garden-controller setup complete\""
	msg := parseStartupMessage(input)
	require.Equal(t, "garden-controller setup complete", msg)
}

func TestSendGardenStartupMessage_WarnLogs(t *testing.T) {
	tests := []struct {
		name         string
		garden       *pkg.Garden
		topic        string
		payload      string
		expectedLogs string
	}{
		{
			"NotificationsDisabled",
			&pkg.Garden{},
			"", "",
			`level=WARN msg="garden does not have controller_startup notification enabled" garden_id=00000000000000000000 topic=""
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			w := &Worker{
				logger: slog.New(slog.NewTextHandler(&logBuffer, nil)),
			}
			err := w.sendGardenStartupMessage(context.Background(), tt.garden, tt.topic, tt.payload)
			require.NoError(t, err)

			// Remove the time attribute before asserting
			logs := strings.SplitN(logBuffer.String(), " ", 2)[1]
			require.Equal(t, tt.expectedLogs, logs)
		})
	}
}

func TestGetGardenAndSendMessage_WarnLogs(t *testing.T) {
	tests := []struct {
		name         string
		garden       *pkg.Garden
		topic        string
		payload      string
		expectedLogs string
	}{
		{
			"UnexpectedMessage",
			&pkg.Garden{},
			"topic", "NOT THE MESSAGE",
			`level=WARN msg="unexpected message from controller" topic=topic message="NOT THE MESSAGE"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			w := &Worker{
				logger: slog.New(slog.NewTextHandler(&logBuffer, nil)),
			}
			err := w.getGardenAndSendStartupMessage(tt.topic, tt.payload)
			require.NoError(t, err)

			// Remove the time attribute before asserting
			logs := strings.SplitN(logBuffer.String(), " ", 2)[1]
			require.Equal(t, tt.expectedLogs, logs)
		})
	}
}
