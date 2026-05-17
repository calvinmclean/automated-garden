package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
	babytest "github.com/calvinmclean/babyapi/test"
	"github.com/stretchr/testify/assert"
)

func TestNotesAPI(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	assert.NoError(t, err)

	garden := createExampleGarden()
	err = storageClient.Gardens.Set(context.Background(), garden)
	assert.NoError(t, err)

	zone := &pkg.Zone{
		ID:       babyapi.NewID(),
		GardenID: garden.ID.ID,
		Name:     "Test Zone",
		Position: pointer(uint(0)),
	}
	err = storageClient.Zones.Set(context.Background(), zone)
	assert.NoError(t, err)

	api := NewNotesAPI()
	api.setup(storageClient)

	now := time.Now()
	gardenID := garden.GetID()
	zoneID := zone.GetID()

	t.Run("CreateNote", func(t *testing.T) {
		note := pkg.Note{
			ID:        babyapi.NewID(),
			Title:     "Test Note",
			Content:   "This is a test note",
			CreatedAt: &now,
		}

		body, err := json.Marshal(note)
		assert.NoError(t, err)

		r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", notesBasePath, note.GetID()), bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := babytest.TestRequest(t, api.API, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("CreateNoteWithGardenAndZone", func(t *testing.T) {
		note := pkg.Note{
			ID:        babyapi.NewID(),
			Title:     "Garden Note",
			Content:   "Note about a garden",
			CreatedAt: &now,
			GardenID:  &gardenID,
			ZoneID:    &zoneID,
		}

		body, err := json.Marshal(note)
		assert.NoError(t, err)

		r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", notesBasePath, note.GetID()), bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := babytest.TestRequest(t, api.API, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("CreateNote_ErrorGardenNotExist", func(t *testing.T) {
		body := `{"title": "Bad Note", "garden_id": "c5cvhpcbcv45e8bp16d1"}`

		r := httptest.NewRequest(http.MethodPost, notesBasePath, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := babytest.TestRequest(t, api.API, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "garden not found")
	})

	t.Run("CreateNote_ErrorZoneNotExist", func(t *testing.T) {
		body := `{"title": "Bad Note", "zone_id": "c5cvhpcbcv45e8bp16d1"}`

		r := httptest.NewRequest(http.MethodPost, notesBasePath, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := babytest.TestRequest(t, api.API, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "zone not found")
	})

	t.Run("CreateNote_MissingTitle", func(t *testing.T) {
		body := `{"content": "No title here"}`

		r := httptest.NewRequest(http.MethodPost, notesBasePath, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := babytest.TestRequest(t, api.API, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing required title field")
	})

	t.Run("GetNote", func(t *testing.T) {
		note := pkg.Note{
			ID:        babyapi.NewID(),
			Title:     "Get Test",
			Content:   "Content for get test",
			CreatedAt: &now,
		}

		body, err := json.Marshal(note)
		assert.NoError(t, err)

		// Create first
		r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", notesBasePath, note.GetID()), bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		babytest.TestRequest(t, api.API, r)

		// Get
		r = httptest.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", notesBasePath, note.GetID()), http.NoBody)
		w := babytest.TestRequest(t, api.API, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"title":"Get Test"`)
	})

	t.Run("DeleteNote", func(t *testing.T) {
		note := pkg.Note{
			ID:        babyapi.NewID(),
			Title:     "Delete Test",
			Content:   "Content for delete test",
			CreatedAt: &now,
		}

		body, err := json.Marshal(note)
		assert.NoError(t, err)

		// Create first
		r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", notesBasePath, note.GetID()), bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		babytest.TestRequest(t, api.API, r)

		// Delete
		r = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("%s/%s", notesBasePath, note.GetID()), http.NoBody)
		w := babytest.TestRequest(t, api.API, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("ListNotes", func(t *testing.T) {
		// List should work and return notes
		r := httptest.NewRequest(http.MethodGet, notesBasePath, http.NoBody)
		w := babytest.TestRequest(t, api.API, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("PatchNote", func(t *testing.T) {
		note := pkg.Note{
			ID:        babyapi.NewID(),
			Title:     "Patch Test",
			Content:   "Original content",
			CreatedAt: &now,
		}

		body, err := json.Marshal(note)
		assert.NoError(t, err)

		// Create first
		r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("%s/%s", notesBasePath, note.GetID()), bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		babytest.TestRequest(t, api.API, r)

		// Patch
		patchBody := `{"title": "Patched Title"}`
		r = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("%s/%s", notesBasePath, note.GetID()), strings.NewReader(patchBody))
		r.Header.Set("Content-Type", "application/json")
		w := babytest.TestRequest(t, api.API, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
