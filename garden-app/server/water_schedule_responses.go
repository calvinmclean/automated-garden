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
func (zr *AllWaterSchedulesResponse) Render(w http.ResponseWriter, r *http.Request) error {
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
		NextWaterTime: wsr.worker.GetNextWaterTime(ws),
	}

	response.Links = append(response.Links,
		Link{
			"self",
			fmt.Sprintf("%s/%s", waterScheduleBasePath, ws.ID),
		},
	)

	if ws.HasWeatherControl() {
		response.WeatherData = getWeatherData(ctx, ws, wsr.storageClient)

		// TODO: Can I re-enable this if moisture comes from WeatherClient instead of garden? Follow up issue #95
		// if ws.HasSoilMoistureControl() {
		// 	logger.Debug("getting moisture data for WaterSchedule")
		// 	soilMoisture, err := wsr.getMoisture(ctx, garden, ws)
		// 	if err != nil {
		// 		logger.WithError(err).Warn("unable to get moisture data for WaterSchedule")
		// 	} else {
		// 		logger.Debugf("successfully got moisture data for WaterSchedule: %f", soilMoisture)
		// 		response.WeatherData.SoilMoisturePercent = &soilMoisture
		// 	}
		// }
	}

	nextWateringDuration := ws.Duration.Duration
	if ws.HasWeatherControl() && !ws.EndDated() {
		wd, err := wsr.worker.ScaleWateringDuration(ws, nextWateringDuration)
		if err != nil {
			logger.WithError(err).Warn("unable to determine water duration scale")
		} else {
			nextWateringDuration = time.Duration(wd)
		}
	}
	response.NextWaterDuration = nextWateringDuration.String()

	return response
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (z *WaterScheduleResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// WaterScheduleWaterHistoryResponse wraps a slice of WaterHistory structs plus some aggregate stats for an HTTP response
type WaterScheduleWaterHistoryResponse struct {
	History []pkg.WaterHistory `json:"history"`
	Count   int                `json:"count"`
	Average string             `json:"average"`
	Total   string             `json:"total"`
}

// NewWaterScheduleWaterHistoryResponse creates a response by creating some basic statistics about a list of history events
func NewWaterScheduleWaterHistoryResponse(history []pkg.WaterHistory) WaterScheduleWaterHistoryResponse {
	total := time.Duration(0)
	for _, h := range history {
		amountDuration, _ := time.ParseDuration(h.Duration)
		total += amountDuration
	}
	count := len(history)
	average := time.Duration(0)
	if count != 0 {
		average = time.Duration(int(total) / len(history))
	}
	return WaterScheduleWaterHistoryResponse{
		History: history,
		Count:   count,
		Average: average.String(),
		Total:   time.Duration(total).String(),
	}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (resp WaterScheduleWaterHistoryResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
