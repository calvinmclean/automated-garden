package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/rs/xid"
)

// AllWaterSchedulesResponse is a simple struct being used to render and return a list of all WaterSchedules
type AllWaterSchedulesResponse struct {
	WaterSchedules []*WaterScheduleResponse `json:"water_schedules"`
}

// NewAllWaterSchedulesResponse will create an AllWaterSchedulesResponse from a list of WaterSchedules
func (wsr *WaterSchedulesResource) NewAllWaterSchedulesResponse(waterschedules []*pkg.WaterSchedule) *AllWaterSchedulesResponse {
	waterscheduleResponses := []*WaterScheduleResponse{}
	for _, ws := range waterschedules {
		waterscheduleResponses = append(waterscheduleResponses, wsr.NewWaterScheduleResponse(ws))
	}
	return &AllWaterSchedulesResponse{waterscheduleResponses}
}

// Render ...
func (asr *AllWaterSchedulesResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	for _, ws := range asr.WaterSchedules {
		err := ws.Render(nil, r)
		if err != nil {
			return fmt.Errorf("error rendering water schedule: %w", err)
		}
	}
	return nil
}

// NextWaterDetails has information about the next time this WaterSchedule will be used
type NextWaterDetails struct {
	Time            *time.Time `json:"time,omitempty"`
	Duration        string     `json:"duration,omitempty"`
	WaterScheduleID *xid.ID    `json:"water_schedule_id,omitempty"`
	Message         string     `json:"message,omitempty"`
}

// GetNextWaterDetails returns the NextWaterDetails for the WaterSchedule
func GetNextWaterDetails(ws *pkg.WaterSchedule, worker *worker.Worker, excludeWeatherData bool) NextWaterDetails {
	result := NextWaterDetails{
		Time:     worker.GetNextWaterTime(ws),
		Duration: ws.Duration.Duration.String(),
	}

	if ws.HasWeatherControl() && !excludeWeatherData {
		wd, hadErr := worker.ScaleWateringDuration(ws)
		if hadErr {
			result.Message = "error impacted duration scaling"
		}

		result.Duration = time.Duration(wd).String()
	}

	return result
}

// WaterScheduleResponse is used to represent a WaterSchedule in the response body with the additional Moisture data
// and hypermedia Links fields
type WaterScheduleResponse struct {
	*pkg.WaterSchedule
	WeatherData *WeatherData     `json:"weather_data,omitempty"`
	NextWater   NextWaterDetails `json:"next_water,omitempty"`
	Links       []Link           `json:"links,omitempty"`

	wsr *WaterSchedulesResource
}

// NewWaterScheduleResponse creates a self-referencing WaterScheduleResponse
func (wsr *WaterSchedulesResource) NewWaterScheduleResponse(ws *pkg.WaterSchedule, links ...Link) *WaterScheduleResponse {
	response := &WaterScheduleResponse{
		WaterSchedule: ws,
		Links:         links,
		wsr:           wsr,
	}
	return response
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (ws *WaterScheduleResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	ws.Links = append(ws.Links,
		Link{
			"self",
			fmt.Sprintf("%s/%s", waterScheduleBasePath, ws.ID),
		},
	)

	if ws.HasWeatherControl() && !ws.EndDated() && !excludeWeatherData(r) {
		ws.WeatherData = getWeatherData(r.Context(), ws.WaterSchedule, ws.wsr.storageClient)
	}

	if !ws.EndDated() {
		ws.NextWater = GetNextWaterDetails(ws.WaterSchedule, ws.wsr.worker, excludeWeatherData(r))
	}

	return nil
}
