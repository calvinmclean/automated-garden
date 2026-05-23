package worker

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseControllerInfoMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *pkg.ControllerInfo
		wantErr  bool
	}{
		{
			"ValidFullMessage",
			`info mac="aa:bb:cc:dd:ee:ff",ip="192.168.1.42",version="abc123d"`,
			&pkg.ControllerInfo{
				MACAddress:      "aa:bb:cc:dd:ee:ff",
				IPAddress:       "192.168.1.42",
				FirmwareVersion: "abc123d",
			},
			false,
		},
		{
			"ValidPartialMessage",
			`info mac="aa:bb:cc:dd:ee:ff"`,
			&pkg.ControllerInfo{
				MACAddress: "aa:bb:cc:dd:ee:ff",
			},
			false,
		},
		{
			"MissingInfoPrefix",
			`mac="aa:bb:cc:dd:ee:ff"`,
			nil,
			true,
		},
		{
			"EmptyMessage",
			``,
			nil,
			true,
		},
		{
			"UnknownFieldIgnored",
			`info mac="aa:bb:cc:dd:ee:ff",ip="192.168.1.42",version="abc123d",unknown="xyz"`,
			&pkg.ControllerInfo{
				MACAddress:      "aa:bb:cc:dd:ee:ff",
				IPAddress:       "192.168.1.42",
				FirmwareVersion: "abc123d",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseControllerInfoMessage(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected.MACAddress, info.MACAddress)
			assert.Equal(t, tt.expected.IPAddress, info.IPAddress)
			assert.Equal(t, tt.expected.FirmwareVersion, info.FirmwareVersion)
		})
	}
}

func TestGetGardenAndSaveControllerInfo(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	require.NoError(t, err)

	garden := &pkg.Garden{
		ID:          babyapi.NewID(),
		TopicPrefix: "garden",
		Name:        "garden",
	}
	err = storageClient.Gardens.Set(context.Background(), garden)
	require.NoError(t, err)

	w := NewWorker(storageClient, nil, nil, slog.Default())

	t.Run("SaveControllerInfo", func(t *testing.T) {
		err := w.getGardenAndSaveControllerInfo("garden/data/info", `info mac="aa:bb:cc:dd:ee:ff",ip="192.168.1.42",version="abc123d"`)
		require.NoError(t, err)

		info, err := storageClient.ControllerInfo.Get(context.Background(), garden.GetID())
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.Equal(t, "aa:bb:cc:dd:ee:ff", info.MACAddress)
		assert.Equal(t, "192.168.1.42", info.IPAddress)
		assert.Equal(t, "abc123d", info.FirmwareVersion)
		assert.NotNil(t, info.UpdatedAt)
		assert.WithinDuration(t, time.Now(), *info.UpdatedAt, 5*time.Second)
	})

	t.Run("UnknownTopicPrefix", func(t *testing.T) {
		err := w.getGardenAndSaveControllerInfo("unknown/data/info", `info mac="aa:bb:cc:dd:ee:ff"`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "topic-prefix")
	})

	t.Run("UnexpectedMessage", func(t *testing.T) {
		var logBuffer bytes.Buffer
		w.logger = slog.New(slog.NewTextHandler(&logBuffer, nil))
		err := w.getGardenAndSaveControllerInfo("garden/data/info", `not an info message`)
		require.NoError(t, err)
		assert.Contains(t, logBuffer.String(), "unexpected controller info message")
	})
}
