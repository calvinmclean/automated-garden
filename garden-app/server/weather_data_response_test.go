package server

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
)

func TestWeatherDataCanceledEvapotranspiration(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{ConnectionString: ":memory:"})
	require.NoError(t, err)
	clientID := xid.New()
	require.NoError(t, storageClient.WeatherClientConfigs.Set(context.Background(), &weather.Config{
		ID:      babyapi.ID{ID: clientID},
		Type:    "openmeteo",
		Options: map[string]any{"latitude": 33.4, "longitude": -112.1},
	}))
	ws := createExampleWaterSchedule()
	ws.WeatherControl = &weather.Control{
		Evapotranspiration: &weather.EvapotranspirationScaler{ClientID: clientID},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	data, err := getEvapotranspirationData(ctx, ws, storageClient)
	require.Nil(t, data)
	require.ErrorIs(t, err, context.Canceled)
	require.Contains(t, err.Error(), clientID.String())

	// Exercise the concurrent refresh path with ET configured independently of temperature.
	response := getWeatherData(ctx, ws, storageClient, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.Equal(t, &WeatherData{}, response)
}
