package server

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
	"github.com/rs/xid"
)

// NextWaterDetails has information about the next time this WaterSchedule will be used
type NextWaterDetails struct {
	Time            *time.Time    `json:"time,omitempty"`
	Duration        *pkg.Duration `json:"duration,omitempty"`
	WaterScheduleID *xid.ID       `json:"water_schedule_id,omitempty"`
	Message         string        `json:"message,omitempty"`
}

// GetNextWaterDetails returns the NextWaterDetails for the WaterSchedule
func GetNextWaterDetails(r *http.Request, ws *pkg.WaterSchedule, worker *worker.Worker, excludeWeatherData bool) NextWaterDetails {
	result := NextWaterDetails{
		Time:     worker.GetNextWaterTime(ws),
		Duration: ws.Duration,
	}

	if ws.HasWeatherControl() && !excludeWeatherData {
		wd, hadErr := worker.ScaleWateringDuration(ws)
		if hadErr {
			result.Message = "error impacted duration scaling"
		}

		result.Duration = &pkg.Duration{Duration: time.Duration(wd)}
	}

	var loc *time.Location
	tzHeader := r.Header.Get("X-TZ-Offset")
	if tzHeader != "" {
		var err error
		loc, err = pkg.TimeLocationFromOffset(tzHeader)
		if err != nil {
			result.Message = fmt.Sprintf("error parsing timezone from header: %v", err)
		}
	}
	if loc == nil {
		loc = ws.StartTime.Time.Location()
	}

	offsetTime := result.Time.In(loc)
	result.Time = &offsetTime

	return result
}

// WaterScheduleResponse is used to represent a WaterSchedule in the response body with the additional Moisture data
// and hypermedia Links fields
type WaterScheduleResponse struct {
	*pkg.WaterSchedule
	WeatherData *WeatherData     `json:"weather_data,omitempty"`
	NextWater   NextWaterDetails `json:"next_water,omitempty"`
	Links       []Link           `json:"links,omitempty"`

	api *WaterSchedulesAPI
}

// NewWaterScheduleResponse creates a self-referencing WaterScheduleResponse
func (api *WaterSchedulesAPI) NewWaterScheduleResponse(ws *pkg.WaterSchedule, links ...Link) *WaterScheduleResponse {
	response := &WaterScheduleResponse{
		WaterSchedule: ws,
		Links:         links,
		api:           api,
	}
	return response
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (ws *WaterScheduleResponse) Render(w http.ResponseWriter, r *http.Request) error {
	ws.Links = append(ws.Links,
		Link{
			"self",
			fmt.Sprintf("%s/%s", waterScheduleBasePath, ws.ID),
		},
	)

	if ws.HasWeatherControl() && !ws.EndDated() && !excludeWeatherData(r) {
		ws.WeatherData = getWeatherData(r.Context(), ws.WaterSchedule, ws.api.storageClient)
	}

	if !ws.EndDated() {
		ws.NextWater = GetNextWaterDetails(r, ws.WaterSchedule, ws.api.worker, excludeWeatherData(r))
	}

	if render.GetAcceptedContentType(r) == render.ContentTypeHTML && r.Method == http.MethodPut {
		w.Header().Add("HX-Trigger", "newWaterSchedule")
	}

	return nil
}

// AllWaterSchedulesResponse is a simple struct being used to render and return a list of all WaterSchedules
type AllWaterSchedulesResponse struct {
	babyapi.ResourceList[*WaterScheduleResponse]
}

func (aws AllWaterSchedulesResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return aws.ResourceList.Render(w, r)
}

func (aws AllWaterSchedulesResponse) HTML(r *http.Request) string {
	slices.SortFunc(aws.Items, func(w *WaterScheduleResponse, x *WaterScheduleResponse) int {
		return strings.Compare(w.Name, x.Name)
	})

	if r.URL.Query().Get("refresh") == "true" {
		return waterSchedulesTemplate.Render(r, aws)
	}

	return waterSchedulesPageTemplate.Render(r, aws)
}
