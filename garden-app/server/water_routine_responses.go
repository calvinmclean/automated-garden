package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
)

// WaterRoutineResponse is used to represent a WaterRoutine in the response body
type WaterRoutineResponse struct {
	*pkg.WaterRoutine
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (wr *WaterRoutineResponse) Render(w http.ResponseWriter, r *http.Request) error {
	if render.GetAcceptedContentType(r) == render.ContentTypeHTML && r.Method == http.MethodPut {
		w.Header().Add("HX-Trigger", "newWaterRoutine")
	}
	return nil
}

// AllWaterRoutinesResponse is a simple struct being used to render and return a list of all WaterRoutines
type AllWaterRoutinesResponse struct {
	babyapi.ResourceList[*WaterRoutineResponse]
}

func (awrr AllWaterRoutinesResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return awrr.ResourceList.Render(w, r)
}

func (awrr AllWaterRoutinesResponse) HTML(_ http.ResponseWriter, r *http.Request) string {
	slices.SortFunc(awrr.Items, func(wr1, wr2 *WaterRoutineResponse) int {
		return strings.Compare(wr1.Name, wr2.Name)
	})

	if r.URL.Query().Get("refresh") == "true" {
		return waterRoutinesTemplate.Render(r, awrr)
	}

	return waterRoutinesPageTemplate.Render(r, awrr)
}
