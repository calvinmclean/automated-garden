package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
)

// ZoneResponse is used to represent a Zone in the response body with the additional Moisture data
// and hypermedia Links fields
type ZoneResponse struct {
	*pkg.Zone
	WeatherData *WeatherData     `json:"weather_data,omitempty"`
	NextWater   NextWaterDetails `json:"next_water,omitempty"`
	Links       []Link           `json:"links,omitempty"`

	api *ZonesAPI
}

// NewZoneResponse creates a self-referencing ZoneResponse
func (api *ZonesAPI) NewZoneResponse(zone *pkg.Zone, links ...Link) *ZoneResponse {
	return &ZoneResponse{
		Zone:  zone,
		Links: links,

		api: api,
	}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (zr *ZoneResponse) Render(_ http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	excludeWeatherData := excludeWeatherData(r)

	logger := babyapi.GetLoggerFromContext(r.Context())

	ws := []*pkg.WaterSchedule{}
	for _, id := range zr.Zone.WaterScheduleIDs {
		result, err := zr.api.storageClient.WaterSchedules.Get(id.String())
		if err != nil {
			return fmt.Errorf("unable to get WaterSchedule for ZoneResponse: %w", err)
		}

		ws = append(ws, result)
	}

	garden, httpErr := zr.api.getGardenFromRequest(r)
	if httpErr != nil {
		logger.Error("unable to get garden for zone", "error", httpErr)
		return httpErr
	}

	gardenPath := fmt.Sprintf("%s/%s", gardenBasePath, garden.ID)
	zr.Links = append(zr.Links,
		Link{
			"self",
			fmt.Sprintf("%s%s/%s", gardenPath, zoneBasePath, zr.Zone.ID),
		},
		Link{
			"garden",
			gardenPath,
		},
	)

	if zr.Zone.EndDated() {
		return nil
	}

	zr.Links = append(zr.Links,
		Link{
			"action",
			fmt.Sprintf("%s%s/%s/action", gardenPath, zoneBasePath, zr.Zone.ID),
		},
		Link{
			"history",
			fmt.Sprintf("%s%s/%s/history", gardenPath, zoneBasePath, zr.Zone.ID),
		},
	)

	nextWaterSchedule := zr.api.worker.GetNextActiveWaterSchedule(ws)

	if nextWaterSchedule == nil {
		zr.NextWater = NextWaterDetails{
			Message: "no active WaterSchedules",
		}
		return nil
	}

	zr.NextWater = GetNextWaterDetails(nextWaterSchedule, zr.api.worker, excludeWeatherData)
	zr.NextWater.WaterScheduleID = &nextWaterSchedule.ID.ID

	if zr.Zone.SkipCount != nil && *zr.Zone.SkipCount > 0 {
		zr.NextWater.Message = fmt.Sprintf("skip_count %d affected the time", *zr.Zone.SkipCount)
		newNextTime := zr.NextWater.Time.Add(time.Duration(*zr.Zone.SkipCount) * nextWaterSchedule.Interval.Duration)
		zr.NextWater.Time = &newNextTime
	}

	if nextWaterSchedule.HasWeatherControl() && !excludeWeatherData {
		zr.WeatherData = getWeatherData(ctx, nextWaterSchedule, zr.api.storageClient)

		if nextWaterSchedule.HasSoilMoistureControl() && garden != nil {
			logger.Debug("getting moisture data for Zone")
			soilMoisture, err := zr.api.getMoisture(ctx, garden, zr.Zone)
			if err != nil {
				logger.Warn("unable to get moisture data for Zone", "error", err)
			} else {
				logger.Debug("successfully got moisture data for Zone", "moisture", soilMoisture)
				zr.WeatherData.SoilMoisturePercent = &soilMoisture
			}
		}
	}

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

func filterZoneByGardenID(gardenID string) babyapi.FilterFunc[*pkg.Zone] {
	return func(z *pkg.Zone) bool {
		return z.GardenID.String() == gardenID
	}
}

type ZoneActionResponse struct{}

func (*ZoneActionResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
