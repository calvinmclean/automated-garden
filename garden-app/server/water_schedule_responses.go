package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// AllWaterSchedulesResponse is a simple struct being used to render and return a list of all WaterSchedules
type AllWaterSchedulesResponse struct {
	WaterSchedules []*WaterScheduleResponse `json:"water_schedules"`
}

// NewAllWaterSchedulesResponse will create an AllWaterSchedulesResponse from a list of WaterSchedules
func (wsr WaterSchedulesResource) NewAllWaterSchedulesResponse(ctx context.Context, waterschedules []*pkg.WaterSchedule) *AllWaterSchedulesResponse {
	waterscheduleResponses := []*WaterScheduleResponse{}
	for _, ws := range waterschedules {
		waterscheduleResponses = append(waterscheduleResponses, wsr.NewWaterScheduleResponse(ctx, ws))
	}
	return &AllWaterSchedulesResponse{waterscheduleResponses}
}

// Render ...
func (zr *AllWaterSchedulesResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// WaterScheduleResponse is used to represent a WaterSchedule in the response body with the additional Moisture data
// and hypermedia Links fields
type WaterScheduleResponse struct {
	*pkg.WaterSchedule
	WeatherData       *WeatherData `json:"weather_data,omitempty"`
	NextWaterTime     *time.Time   `json:"next_water_time,omitempty"`
	NextWaterDuration string       `json:"next_water_duration,omitempty"`
	Links             []Link       `json:"links,omitempty"`
}

// NewWaterScheduleResponse creates a self-referencing WaterScheduleResponse
func (wsr WaterSchedulesResource) NewWaterScheduleResponse(ctx context.Context, ws *pkg.WaterSchedule, links ...Link) *WaterScheduleResponse {
	logger := getLoggerFromContext(ctx).WithField(waterScheduleIDLogField, ws.ID.String())

	response := &WaterScheduleResponse{
		WaterSchedule: ws,
		Links:         links,
	}
	if !ws.EndDated() {
		response.NextWaterTime = wsr.worker.GetNextWaterTime(ws)
	}

	response.Links = append(response.Links,
		Link{
			"self",
			fmt.Sprintf("%s/%s", waterScheduleBasePath, ws.ID),
		},
	)

	if ws.HasWeatherControl() && !ws.EndDated() {
		response.WeatherData = getWeatherData(ctx, ws, wsr.storageClient)
	}

	if !ws.EndDated() {
		response.NextWaterDuration = ws.Duration.Duration.String()

		if ws.HasWeatherControl() {
			wd, err := wsr.worker.ScaleWateringDuration(ws)
			if err != nil {
				logger.WithError(err).Warn("unable to determine water duration scale")
			} else {
				response.NextWaterDuration = time.Duration(wd).String()
			}
		}
	}

	return response
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (z *WaterScheduleResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
