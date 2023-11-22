package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/worker"
	"github.com/rs/xid"
)

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
