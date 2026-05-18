package pkg

import (
	"errors"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/babyapi"
)

// Note represents a user-created note that can optionally be associated with a Garden and/or Zone
type Note struct {
	ID        babyapi.ID `json:"id" yaml:"id,omitempty"`
	Title     string     `json:"title" yaml:"title"`
	Content   string     `json:"content" yaml:"content"`
	CreatedAt *time.Time `json:"created_at" yaml:"created_at,omitempty"`
	GardenID  *string    `json:"garden_id,omitempty" yaml:"garden_id,omitempty"`
	ZoneID    *string    `json:"zone_id,omitempty" yaml:"zone_id,omitempty"`
}

func (n *Note) GetID() string {
	return n.ID.String()
}

func (n *Note) ParentID() string {
	return ""
}

// Patch allows for easily updating individual fields of a Note by passing in a new Note containing
// the desired values
func (n *Note) Patch(newNote *Note) *babyapi.ErrResponse {
	if newNote.Title != "" {
		n.Title = newNote.Title
	}
	if newNote.Content != "" {
		n.Content = newNote.Content
	}
	if newNote.CreatedAt != nil {
		n.CreatedAt = newNote.CreatedAt
	}
	if newNote.GardenID != nil {
		n.GardenID = newNote.GardenID
	}
	if newNote.ZoneID != nil {
		n.ZoneID = newNote.ZoneID
	}

	return nil
}

func (n *Note) Bind(r *http.Request) error {
	if n == nil {
		return errors.New("missing required Note fields")
	}

	err := n.ID.Bind(r)
	if err != nil {
		return err
	}

	// Ignore empty string provided for optional IDs
	if n.GardenID != nil && *n.GardenID == "" {
		n.GardenID = nil
	}
	if n.ZoneID != nil && *n.ZoneID == "" {
		n.ZoneID = nil
	}

	now := clock.Now()
	switch r.Method {
	case http.MethodPost:
		n.CreatedAt = &now
		fallthrough
	case http.MethodPut:
		if n.CreatedAt == nil || n.CreatedAt.IsZero() {
			n.CreatedAt = &now
		}
		if n.Title == "" {
			return errors.New("missing required title field")
		}
	}

	return nil
}

func (n *Note) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
