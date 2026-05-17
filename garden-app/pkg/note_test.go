package pkg

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/assert"
)

func TestNoteGetID(t *testing.T) {
	id := babyapi.NewID()
	note := &Note{ID: id}
	assert.Equal(t, id.String(), note.GetID())
}

func TestNoteParentID(t *testing.T) {
	note := &Note{}
	assert.Equal(t, "", note.ParentID())
}

func TestNotePatch(t *testing.T) {
	now := clock.Now()
	gardenID := "garden123"
	zoneID := "zone456"

	tests := []struct {
		name         string
		initial      *Note
		newNote      *Note
		expected     *Note
		expectedResp *babyapi.ErrResponse
	}{
		{
			name: "Patch Title",
			initial: &Note{
				Title:   "Old Title",
				Content: "Content",
			},
			newNote: &Note{
				Title: "New Title",
			},
			expected: &Note{
				Title:   "New Title",
				Content: "Content",
			},
		},
		{
			name: "Patch Content",
			initial: &Note{
				Title:   "Title",
				Content: "Old Content",
			},
			newNote: &Note{
				Content: "New Content",
			},
			expected: &Note{
				Title:   "Title",
				Content: "New Content",
			},
		},
		{
			name: "Patch GardenID",
			initial: &Note{
				Title:    "Title",
				GardenID: nil,
			},
			newNote: &Note{
				GardenID: &gardenID,
			},
			expected: &Note{
				Title:    "Title",
				GardenID: &gardenID,
			},
		},
		{
			name: "Patch ZoneID",
			initial: &Note{
				Title:  "Title",
				ZoneID: nil,
			},
			newNote: &Note{
				ZoneID: &zoneID,
			},
			expected: &Note{
				Title:  "Title",
				ZoneID: &zoneID,
			},
		},
		{
			name: "Patch CreatedAt",
			initial: &Note{
				Title:     "Title",
				CreatedAt: &now,
			},
			newNote: &Note{
				CreatedAt: func() *time.Time { t := now.Add(time.Hour); return &t }(),
			},
			expected: &Note{
				Title:     "Title",
				CreatedAt: func() *time.Time { t := now.Add(time.Hour); return &t }(),
			},
		},
		{
			name: "Empty newNote doesn't change anything",
			initial: &Note{
				Title:   "Title",
				Content: "Content",
			},
			newNote: &Note{},
			expected: &Note{
				Title:   "Title",
				Content: "Content",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.initial.Patch(tt.newNote)
			assert.Equal(t, tt.expectedResp, resp)
			assert.Equal(t, tt.expected.Title, tt.initial.Title)
			assert.Equal(t, tt.expected.Content, tt.initial.Content)
			assert.Equal(t, tt.expected.GardenID, tt.initial.GardenID)
			assert.Equal(t, tt.expected.ZoneID, tt.initial.ZoneID)
			assert.Equal(t, tt.expected.CreatedAt, tt.initial.CreatedAt)
		})
	}
}

func TestNoteBind(t *testing.T) {
	now := clock.Now()
	clock.MockTime()
	defer clock.Reset()

	tests := []struct {
		name        string
		method      string
		note        *Note
		expectedErr error
		validate    func(t *testing.T, note *Note)
	}{
		{
			name:   "POST with valid note",
			method: http.MethodPost,
			note: &Note{
				Title: "Test Note",
			},
			validate: func(t *testing.T, note *Note) {
				assert.NotNil(t, note.CreatedAt)
				assert.Equal(t, clock.Now(), *note.CreatedAt)
			},
		},
		{
			name:        "POST missing title",
			method:      http.MethodPost,
			note:        &Note{},
			expectedErr: errors.New("missing required title field"),
		},
		{
			name:   "PUT with valid note",
			method: http.MethodPut,
			note: &Note{
				ID:        babyapi.NewID(),
				Title:     "Test Note",
				CreatedAt: &now,
			},
			validate: func(t *testing.T, note *Note) {
				assert.Equal(t, now, *note.CreatedAt)
			},
		},
		{
			name:   "PUT missing title",
			method: http.MethodPut,
			note: &Note{
				ID:        babyapi.NewID(),
				CreatedAt: &now,
			},
			expectedErr: errors.New("missing required title field"),
		},
		{
			name:   "PUT creates created_at if missing",
			method: http.MethodPut,
			note: &Note{
				ID:    babyapi.NewID(),
				Title: "Test Note",
			},
			validate: func(t *testing.T, note *Note) {
				assert.NotNil(t, note.CreatedAt)
				assert.Equal(t, clock.Now(), *note.CreatedAt)
			},
		},
		{
			name:   "PATCH valid",
			method: http.MethodPatch,
			note: &Note{
				Title: "Test Note",
			},
		},
		{
			name:   "POST clears empty GardenID",
			method: http.MethodPost,
			note: &Note{
				Title:    "Test Note",
				GardenID: func() *string { s := ""; return &s }(),
			},
			validate: func(t *testing.T, note *Note) {
				assert.Nil(t, note.GardenID)
			},
		},
		{
			name:   "POST clears empty ZoneID",
			method: http.MethodPost,
			note: &Note{
				Title:  "Test Note",
				ZoneID: func() *string { s := ""; return &s }(),
			},
			validate: func(t *testing.T, note *Note) {
				assert.Nil(t, note.ZoneID)
			},
		},
		{
			name:        "nil note",
			method:      http.MethodPost,
			note:        nil,
			expectedErr: errors.New("missing required Note fields"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.note.Bind(&http.Request{Method: tt.method})
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
				return
			}
			assert.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, tt.note)
			}
		})
	}
}

func TestNoteRender(t *testing.T) {
	note := &Note{}
	err := note.Render(nil, nil)
	assert.NoError(t, err)
}
