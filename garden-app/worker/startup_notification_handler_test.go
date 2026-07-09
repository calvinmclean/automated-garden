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
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/fake"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetGardenAndHandleLogMessage(t *testing.T) {
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
		err = w.getGardenAndHandleLogMessage("garden/data/logs", "logs message=\"garden-controller setup complete\"")
		require.NoError(t, err)
		mqttClient.AssertExpectations(t)
	})

	t.Run("LightTurnsOff", func(t *testing.T) {
		c.Add(12 * time.Hour)
		fmt.Println("LightTime", garden.LightSchedule.StartTime.Time)
		fmt.Println("Now", clock.Now())
		fmt.Println(garden.LightSchedule.NextChange(c.Now()))
		mqttClient.On("Publish", mock.Anything, "garden/command/light", []byte(`{"state":"OFF"}`)).Return(nil)
		err = w.getGardenAndHandleLogMessage("garden/data/logs", "logs message=\"garden-controller setup complete\"")
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

func TestStartupLogIncludesResetReason(t *testing.T) {
	fake.Reset()
	defer fake.Reset()

	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	require.NoError(t, err)

	nc := &notifications.Client{
		ID:   babyapi.NewID(),
		Name: "test",
		URL:  "fake://success",
	}
	err = storageClient.NotificationClientConfigs.Set(context.Background(), nc)
	require.NoError(t, err)

	notificationClientID := nc.GetID()
	garden := &pkg.Garden{
		ID:                   babyapi.NewID(),
		TopicPrefix:          "garden",
		Name:                 "garden",
		NotificationClientID: &notificationClientID,
		NotificationSettings: &pkg.NotificationSettings{
			ControllerStartup: true,
		},
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	require.NoError(t, err)

	w := NewWorker(storageClient, nil, nil, slog.Default())

	err = w.getGardenAndHandleLogMessage("garden/data/logs", `logs,level=info,source=startup message="garden-controller setup complete",reset_reason="Reset due to power-on event."`)
	require.NoError(t, err)

	last := fake.LastMessage()
	require.Equal(t, "garden connected", last.Title)
	require.Contains(t, last.Message, "garden-controller setup complete")
	require.Contains(t, last.Message, "reset_reason=Reset due to power-on event.")
}

func TestParseControllerLogMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *controllerLog
		wantErr  bool
	}{
		{
			"LegacyStartupMessage",
			`logs message="garden-controller setup complete"`,
			&controllerLog{
				Level:   "info",
				Message: "garden-controller setup complete",
			},
			false,
		},
		{
			"StartupMessageWithResetReason",
			`logs,level=info,source=startup message="garden-controller setup complete",reset_reason="Reset due to power-on event."`,
			&controllerLog{
				Level:       "info",
				Source:      "startup",
				Message:     "garden-controller setup complete",
				ResetReason: "Reset due to power-on event.",
			},
			false,
		},
		{
			"FullMessage",
			`logs,level=error,source=wifi_manager message="error restarting mDNS after reconnect"`,
			&controllerLog{
				Level:   "error",
				Source:  "wifi_manager",
				Message: "error restarting mDNS after reconnect",
			},
			false,
		},
		{
			"LevelDefaultsToInfo",
			`logs,source=main message="hello"`,
			&controllerLog{
				Level:   "info",
				Source:  "main",
				Message: "hello",
			},
			false,
		},
		{
			"MissingMessage",
			`logs,level="error" state=1`,
			nil,
			true,
		},
		{
			"WrongMeasurement",
			`water,status=complete zone=1`,
			nil,
			true,
		},
		{
			"EmptyMessage",
			``,
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, err := parseControllerLogMessage(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, log)
		})
	}
}

func TestHandleGenericLog(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	require.NoError(t, err)

	nc := &notifications.Client{
		ID:   babyapi.NewID(),
		Name: "test",
		URL:  "fake://success",
	}
	err = storageClient.NotificationClientConfigs.Set(context.Background(), nc)
	require.NoError(t, err)

	notificationClientID := nc.GetID()
	garden := &pkg.Garden{
		ID:                   babyapi.NewID(),
		TopicPrefix:          "garden",
		Name:                 "garden",
		NotificationClientID: &notificationClientID,
		NotificationSettings: &pkg.NotificationSettings{
			ControllerErrors: true,
		},
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	require.NoError(t, err)

	w := NewWorker(storageClient, nil, nil, slog.Default())

	t.Run("ErrorNotifies", func(t *testing.T) {
		var logBuffer bytes.Buffer
		w.logger = slog.New(slog.NewTextHandler(&logBuffer, nil))
		err := w.getGardenAndHandleLogMessage("garden/data/logs", `logs,level=error,source=wifi_manager message="error restarting mDNS after reconnect"`)
		require.NoError(t, err)

		logs := logBuffer.String()
		require.Contains(t, logs, "controller error")
		require.Contains(t, logs, "level=error")
		require.Contains(t, logs, "source=wifi_manager")
		require.Contains(t, logs, "message=\"error restarting mDNS after reconnect\"")
	})

	t.Run("ErrorDoesNotNotifyWhenDisabled", func(t *testing.T) {
		garden.NotificationSettings.ControllerErrors = false
		err := storageClient.Gardens.Set(context.Background(), garden)
		require.NoError(t, err)

		var logBuffer bytes.Buffer
		w.logger = slog.New(slog.NewTextHandler(&logBuffer, nil))
		err = w.getGardenAndHandleLogMessage("garden/data/logs", `logs,level=error message="error"`)
		require.NoError(t, err)
		require.Contains(t, logBuffer.String(), "controller error")
	})

	t.Run("InfoDoesNotNotify", func(t *testing.T) {
		garden.NotificationSettings.ControllerErrors = true
		err := storageClient.Gardens.Set(context.Background(), garden)
		require.NoError(t, err)

		var logBuffer bytes.Buffer
		w.logger = slog.New(slog.NewTextHandler(&logBuffer, nil))
		err = w.getGardenAndHandleLogMessage("garden/data/logs", `logs,level=info,source=main message="hello"`)
		require.NoError(t, err)
		require.Contains(t, logBuffer.String(), "controller log")
	})
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

func TestGetGardenAndHandleLogMessage_WarnLogs(t *testing.T) {
	tests := []struct {
		name            string
		garden          *pkg.Garden
		topic           string
		payload         string
		expectedContain []string
	}{
		{
			"UnexpectedMessage",
			&pkg.Garden{},
			"topic", "NOT THE MESSAGE",
			[]string{
				`level=WARN msg="unexpected controller log message"`,
				`topic=topic`,
				`message="NOT THE MESSAGE"`,
				`error="error parsing line protocol`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			w := &Worker{
				logger: slog.New(slog.NewTextHandler(&logBuffer, nil)),
			}
			err := w.getGardenAndHandleLogMessage(tt.topic, tt.payload)
			require.NoError(t, err)

			// Remove the time attribute before asserting
			logs := strings.SplitN(logBuffer.String(), " ", 2)[1]
			for _, expected := range tt.expectedContain {
				require.Contains(t, logs, expected)
			}
		})
	}
}
