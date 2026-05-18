package server

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"slices"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/babyapi"
	"github.com/calvinmclean/babyapi/extensions"
	"github.com/go-chi/render"
)

const (
	notesBasePath = "/notes"
)

// NotesAPI encapsulates the structs and dependencies necessary for the "/notes" API
type NotesAPI struct {
	*babyapi.API[*pkg.Note]

	storageClient *storage.Client
}

func NewNotesAPI() *NotesAPI {
	api := &NotesAPI{}

	api.API = babyapi.NewAPI("Notes", notesBasePath, func() *pkg.Note { return &pkg.Note{} })
	api.SetResponseWrapper(func(n *pkg.Note) render.Renderer {
		return api.NewNoteResponse(n)
	})
	api.SetSearchResponseWrapper(func(notes iter.Seq2[*pkg.Note, error]) render.Renderer {
		resp := AllNotesResponse{ResourceList: babyapi.ResourceList[*NoteResponse]{}, api: api}

		for n, err := range notes {
			if err != nil {
				continue
			}
			resp.ResourceList.Items = append(resp.ResourceList.Items, api.NewNoteResponse(n))
		}

		return resp
	})

	api.SetOnCreateOrUpdate(api.onCreateOrUpdate)

	api.AddCustomRoute(http.MethodGet, "/components", babyapi.Handler(func(_ http.ResponseWriter, r *http.Request) render.Renderer {
		switch r.URL.Query().Get("type") {
		case "create_modal":
			return api.noteModalRenderer(r.Context(), &pkg.Note{
				ID: babyapi.NewID(),
			})
		case "zone_select":
			return api.zoneSelectRenderer(r.Context(), r.URL.Query().Get("GardenID"), r.URL.Query().Get("selected_zone_id"))
		default:
			return babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.AddCustomIDRoute(http.MethodGet, "/components", api.GetRequestedResourceAndDo(func(_ http.ResponseWriter, r *http.Request, n *pkg.Note) (render.Renderer, *babyapi.ErrResponse) {
		switch r.URL.Query().Get("type") {
		case "edit_modal":
			return api.noteModalRenderer(r.Context(), n), nil
		default:
			return nil, babyapi.ErrInvalidRequest(fmt.Errorf("invalid component: %s", r.URL.Query().Get("type")))
		}
	}))

	api.ApplyExtension(extensions.HTMX[*pkg.Note]{})

	api.EnableMCP(babyapi.MCPPermRead)

	return api
}

func (api *NotesAPI) setup(storageClient *storage.Client) {
	api.storageClient = storageClient
	api.SetStorage(api.storageClient.Notes)
}

func (api *NotesAPI) noteModalRenderer(ctx context.Context, note *pkg.Note) render.Renderer {
	gardens := make([]*pkg.Garden, 0)
	for g, err := range api.storageClient.Gardens.Search(ctx, "", nil) {
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("error getting all gardens to create note modal: %w", err))
		}
		gardens = append(gardens, g)
	}

	slices.SortFunc(gardens, func(g1, g2 *pkg.Garden) int {
		return strings.Compare(g1.Name, g2.Name)
	})

	// Pre-load zones for the selected garden when editing
	zones := make([]*pkg.Zone, 0)
	if note.GardenID != nil && *note.GardenID != "" {
		for z, err := range api.storageClient.Zones.Search(ctx, *note.GardenID, nil) {
			if err != nil {
				return babyapi.InternalServerError(fmt.Errorf("error getting zones for garden %s: %w", *note.GardenID, err))
			}
			if z.GardenID.String() == *note.GardenID {
				zones = append(zones, z)
			}
		}
	}

	slices.SortFunc(zones, func(z1, z2 *pkg.Zone) int {
		return strings.Compare(z1.Name, z2.Name)
	})

	selectedZoneID := ""
	if note.ZoneID != nil {
		selectedZoneID = *note.ZoneID
	}

	return noteModalTemplate.Renderer(map[string]any{
		"Note":           note,
		"Gardens":        gardens,
		"Zones":          zones,
		"SelectedZoneID": selectedZoneID,
	})
}

func (api *NotesAPI) zoneSelectRenderer(ctx context.Context, gardenID, selectedZoneID string) render.Renderer {
	zones := make([]*pkg.Zone, 0)
	if gardenID == "" {
		return babyapi.ErrInvalidRequest(errors.New("missing GardenID"))
	}

	for z, err := range api.storageClient.Zones.Search(ctx, gardenID, nil) {
		if err != nil {
			return babyapi.InternalServerError(fmt.Errorf("error getting zones for garden %s: %w", gardenID, err))
		}
		if z.GardenID.String() == gardenID {
			zones = append(zones, z)
		}
	}

	slices.SortFunc(zones, func(z1, z2 *pkg.Zone) int {
		return strings.Compare(z1.Name, z2.Name)
	})

	return noteZoneSelectTemplate.Renderer(map[string]any{
		"Zones":          zones,
		"SelectedZoneID": selectedZoneID,
	})
}

func (api *NotesAPI) onCreateOrUpdate(_ http.ResponseWriter, r *http.Request, note *pkg.Note) *babyapi.ErrResponse {
	// Validate Garden exists if provided
	if note.GardenID != nil {
		_, err := api.storageClient.Gardens.Get(r.Context(), *note.GardenID)
		if err != nil {
			if errors.Is(err, babyapi.ErrNotFound) {
				return babyapi.ErrInvalidRequest(fmt.Errorf("garden not found: %w", err))
			}
			return babyapi.InternalServerError(err)
		}
	}

	// Validate Zone exists if provided
	if note.ZoneID != nil {
		_, err := api.storageClient.Zones.Get(r.Context(), *note.ZoneID)
		if err != nil {
			if errors.Is(err, babyapi.ErrNotFound) {
				return babyapi.ErrInvalidRequest(fmt.Errorf("zone not found: %w", err))
			}
			return babyapi.InternalServerError(err)
		}
	}

	return nil
}
