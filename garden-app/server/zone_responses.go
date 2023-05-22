package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

// AllZonesResponse is a simple struct being used to render and return a list of all Zones
type AllZonesResponse struct {
	Zones []*ZoneResponse `json:"zones"`
}

// NewAllZonesResponse will create an AllZonesResponse from a list of Zones
func (zr ZonesResource) NewAllZonesResponse(ctx context.Context, zones []*pkg.Zone, garden *pkg.Garden) *AllZonesResponse {
	zoneResponses := []*ZoneResponse{}
	for _, z := range zones {
		zoneResponses = append(zoneResponses, zr.NewZoneResponse(ctx, garden, z))
	}
	return &AllZonesResponse{zoneResponses}
}

// Render ...
func (zr *AllZonesResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// ZoneResponse is used to represent a Zone in the response body with the additional Moisture data
// and hypermedia Links fields
type ZoneResponse struct {
	*pkg.Zone
	WeatherData       *WeatherData `json:"weather_data,omitempty"`
	NextWaterTime     *time.Time   `json:"next_water_time,omitempty"`
	NextWaterDuration string       `json:"next_water_duration,omitempty"`
	Links             []Link       `json:"links,omitempty"`
}

// NewZoneResponse creates a self-referencing ZoneResponse
func (zr ZonesResource) NewZoneResponse(ctx context.Context, garden *pkg.Garden, zone *pkg.Zone, links ...Link) *ZoneResponse {
	logger := getLoggerFromContext(ctx).WithField(zoneIDLogField, zone.ID.String())

	ws, err := zr.storageClient.GetMultipleWaterSchedules(zone.WaterScheduleIDs)
	if err != nil {
		logger.Errorf("unable to get WaterSchedule for ZoneResponse: %v", err)
	}

	nextWaterSchedule := zr.worker.GetNextWaterSchedule(ws)
	response := &ZoneResponse{
		Zone:          zone,
		Links:         links,
		NextWaterTime: zr.worker.GetNextWaterTime(nextWaterSchedule),
	}

	gardenPath := fmt.Sprintf("%s/%s", gardenBasePath, garden.ID)
	response.Links = append(response.Links,
		Link{
			"self",
			fmt.Sprintf("%s%s/%s", gardenPath, zoneBasePath, zone.ID),
		},
		Link{
			"garden",
			gardenPath,
		},
	)
	if !zone.EndDated() {
		response.Links = append(response.Links,
			Link{
				"action",
				fmt.Sprintf("%s%s/%s/action", gardenPath, zoneBasePath, zone.ID),
			},
			Link{
				"history",
				fmt.Sprintf("%s%s/%s/history", gardenPath, zoneBasePath, zone.ID),
			},
		)
	}

	// TODO: In order to do this, I need to return the "nextWaterSchedule" instead of just the next time
	//       I wil basically reset the refactored GetNextWaterTime and take the code from there to create a function to get the next schedule
	if nextWaterSchedule.HasWeatherControl() {
		response.WeatherData = getWeatherData(ctx, nextWaterSchedule, zr.storageClient)

		if nextWaterSchedule.HasSoilMoistureControl() && garden != nil {
			logger.Debug("getting moisture data for Zone")
			soilMoisture, err := zr.getMoisture(ctx, garden, zone)
			if err != nil {
				logger.WithError(err).Warn("unable to get moisture data for Zone")
			} else {
				logger.Debugf("successfully got moisture data for Zone: %f", soilMoisture)
				response.WeatherData.SoilMoisturePercent = &soilMoisture
			}
		}
	}

	response.NextWaterDuration = nextWaterSchedule.Duration.Duration.String()
	if nextWaterSchedule.HasWeatherControl() && !zone.EndDated() {
		wd, err := zr.worker.ScaleWateringDuration(nextWaterSchedule)
		if err != nil {
			logger.WithError(err).Warn("unable to determine water duration scale")
		} else {
			response.NextWaterDuration = time.Duration(wd).String()
		}
	}

	return response
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (z *ZoneResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// ZoneWaterHistoryResponse wraps a slice of WaterHistory structs plus some aggregate stats for an HTTP response
type ZoneWaterHistoryResponse struct {
	History []pkg.WaterHistory `json:"history"`
	Count   int                `json:"count"`
	Average string             `json:"average"`
	Total   string             `json:"total"`
}

// NewZoneWaterHistoryResponse creates a response by creating some basic statistics about a list of history events
func NewZoneWaterHistoryResponse(history []pkg.WaterHistory) ZoneWaterHistoryResponse {
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
	return ZoneWaterHistoryResponse{
		History: history,
		Count:   count,
		Average: average.String(),
		Total:   time.Duration(total).String(),
	}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (resp ZoneWaterHistoryResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
