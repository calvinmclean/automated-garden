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
func (zr ZonesResource) NewAllZonesResponse(ctx context.Context, zones []*pkg.Zone, garden *pkg.Garden, excludeWeatherData bool) *AllZonesResponse {
	zoneResponses := []*ZoneResponse{}
	for _, z := range zones {
		zoneResponses = append(zoneResponses, zr.NewZoneResponse(ctx, garden, z, excludeWeatherData))
	}
	return &AllZonesResponse{zoneResponses}
}

// Render ...
func (zr *AllZonesResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

// ZoneResponse is used to represent a Zone in the response body with the additional Moisture data
// and hypermedia Links fields
type ZoneResponse struct {
	*pkg.Zone
	WeatherData *WeatherData     `json:"weather_data,omitempty"`
	NextWater   NextWaterDetails `json:"next_water,omitempty"`
	Links       []Link           `json:"links,omitempty"`
}

// NewZoneResponse creates a self-referencing ZoneResponse
func (zr ZonesResource) NewZoneResponse(ctx context.Context, garden *pkg.Garden, zone *pkg.Zone, excludeWeatherData bool, links ...Link) *ZoneResponse {
	logger := getLoggerFromContext(ctx).WithField(zoneIDLogField, zone.ID.String())

	ws, err := zr.storageClient.GetMultipleWaterSchedules(zone.WaterScheduleIDs)
	if err != nil {
		logger.Errorf("unable to get WaterSchedule for ZoneResponse: %v", err)
	}

	response := &ZoneResponse{
		Zone:  zone,
		Links: links,
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

	if zone.EndDated() {
		return response
	}

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

	nextWaterSchedule := zr.worker.GetNextActiveWaterSchedule(ws)

	if nextWaterSchedule == nil {
		response.NextWater = NextWaterDetails{
			Message: "no active WaterSchedules",
		}
		return response
	}

	response.NextWater = GetNextWaterDetails(nextWaterSchedule, zr.worker, excludeWeatherData)
	response.NextWater.WaterScheduleID = &nextWaterSchedule.ID

	if zone.SkipCount != nil && *zone.SkipCount > 0 {
		response.NextWater.Message = fmt.Sprintf("skip_count %d affected the time", *zone.SkipCount)
		newNextTime := response.NextWater.Time.Add(time.Duration(*zone.SkipCount) * nextWaterSchedule.Interval.Duration)
		response.NextWater.Time = &newNextTime
	}

	if nextWaterSchedule.HasWeatherControl() && !excludeWeatherData {
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

	return response
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (z *ZoneResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
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
func (resp ZoneWaterHistoryResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
