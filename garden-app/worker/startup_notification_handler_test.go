package worker

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/stretchr/testify/require"
)

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
			`level=WARN msg="garden does not have controller_startup notification enabled" topic="" garden_id=00000000000000000000
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			w := &Worker{
				logger: slog.New(slog.NewTextHandler(&logBuffer, nil)),
			}
			err := w.sendGardenStartupMessage(tt.garden, tt.topic, tt.payload)
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
