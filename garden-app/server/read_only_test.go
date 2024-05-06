package server

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	babytest "github.com/calvinmclean/babyapi/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadOnlyMiddleware(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
	})
	assert.NoError(t, err)

	api := NewAPI()
	err = api.setup(Config{
		WebConfig: WebConfig{
			ReadOnly: true,
		},
	}, storageClient, nil, nil)
	require.NoError(t, err)

	t.Run("CreateGarden", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodPost, "/gardens", bytes.NewBufferString(`{}`))
		require.NoError(t, err)

		w := babytest.TestRequest(t, api.API, r)
		require.Equal(t, http.StatusOK, w.Code)
		require.Empty(t, w.Body.String())
	})

	t.Run("GetAllGardensShowsNone", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "/gardens", bytes.NewBufferString(`{}`))
		require.NoError(t, err)

		w := babytest.TestRequest(t, api.API, r)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, `{"items":null}`, strings.TrimSpace(w.Body.String()))
	})
}
