package server

import (
	"context"
	"net/http"
	"slices"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
)

// NoteResponse is used to represent a Note in the response body
type NoteResponse struct {
	*pkg.Note
	GardenName string `json:"garden_name,omitempty"` // HTML only
	ZoneName   string `json:"zone_name,omitempty"`   // HTML only
	Links      []Link `json:"links,omitempty"`

	api *NotesAPI
}

// NewNoteResponse creates a self-referencing NoteResponse
func (api *NotesAPI) NewNoteResponse(note *pkg.Note, links ...Link) *NoteResponse {
	return &NoteResponse{
		Note: note,
		Links: append(links, Link{
			"self",
			notesBasePath + "/" + note.ID.String(),
		}),
		api: api,
	}
}

// populateNames fills in GardenName and ZoneName by looking up the referenced resources
func (nr *NoteResponse) populateNames(ctx context.Context) {
	if nr.GardenID != nil {
		garden, err := nr.api.storageClient.Gardens.Get(ctx, *nr.GardenID)
		if err == nil {
			nr.GardenName = garden.Name
		}
	}
	if nr.ZoneID != nil {
		zone, err := nr.api.storageClient.Zones.Get(ctx, *nr.ZoneID)
		if err == nil {
			nr.ZoneName = zone.Name
		}
	}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (nr *NoteResponse) Render(w http.ResponseWriter, r *http.Request) error {
	if render.GetAcceptedContentType(r) == render.ContentTypeHTML && r.Method == http.MethodPut {
		w.Header().Add("HX-Trigger", "newNote")
	}
	return nil
}

func (nr *NoteResponse) HTML(_ http.ResponseWriter, r *http.Request) string {
	nr.populateNames(r.Context())
	return noteCardTemplate.Render(r, nr)
}

// AllNotesResponse is a simple struct being used to render and return a list of all Notes
type AllNotesResponse struct {
	babyapi.ResourceList[*NoteResponse]
	api *NotesAPI
}

func (anr AllNotesResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return anr.ResourceList.Render(w, r)
}

func (anr AllNotesResponse) HTML(_ http.ResponseWriter, r *http.Request) string {
	// Populate garden/zone names for each note before rendering
	for _, item := range anr.Items {
		item.populateNames(r.Context())
	}

	slices.SortFunc(anr.Items, func(n1, n2 *NoteResponse) int {
		// Sort by created_at descending (newest first)
		return n2.CreatedAt.Compare(*n1.CreatedAt)
	})

	if r.URL.Query().Get("refresh") == "true" {
		return notesTemplate.Render(r, anr)
	}

	return notesPageTemplate.Render(r, anr)
}
