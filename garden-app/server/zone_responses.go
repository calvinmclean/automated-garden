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

// Render will take the map of Zones and convert it to a list for a more RESTy response
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

// WeatherData is used to represent the data used for WeatherControl to a user
type WeatherData struct {
	Rain                *RainData        `json:"rain,omitempty"`
	Temperature         *TemperatureData `json:"average_temperature,omitempty"`
	SoilMoisturePercent *float64         `json:"soil_moisture_percent,omitempty"`
}

// RainData shows the total rain in the last watering interval and the scaling factor it would result in
type RainData struct {
	MM          float32 `json:"mm"`
	ScaleFactor float32 `json:"scale_factor"`
}

// TemperatureData shows the average high temperatures in the last watering interval and the scaling factor it would result in
type TemperatureData struct {
	Celcius     float32 `json:"celcius"`
	ScaleFactor float32 `json:"scale_factor"`
}

// NewZoneResponse creates a self-referencing ZoneResponse
func (zr ZonesResource) NewZoneResponse(ctx context.Context, garden *pkg.Garden, zone *pkg.Zone, links ...Link) *ZoneResponse {
	logger := getLoggerFromContext(ctx).WithField(zoneIDLogField, zone.ID.String())

	gardenPath := fmt.Sprintf("%s/%s", gardenBasePath, garden.ID)
	links = append(links,
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
		links = append(links,
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
	var weatherData *WeatherData
	var err error
	if zone.HasWeatherControl() {
		weatherData = &WeatherData{}

		if zone.WaterSchedule.HasSoilMoistureControl() && garden != nil {
			logger.Debug("getting moisture data for Zone")
			soilMoisture, err := zr.getMoisture(ctx, garden, zone)
			if err != nil {
				logger.WithError(err).Warn("unable to get moisture data for Zone")
			} else {
				logger.Debugf("successfully got moisture data for Zone: %f", soilMoisture)
				weatherData.SoilMoisturePercent = &soilMoisture
			}
		}

		interval, err := time.ParseDuration(zone.WaterSchedule.Interval)
		if err != nil {
			logger.WithError(err).Warn("unable to parse WaterSchedule interval duration")
		}
		if zone.WaterSchedule.HasRainControl() && zr.weatherClient != nil {
			logger.Debug("getting rain data for Zone")
			weatherData.Rain = &RainData{}
			weatherData.Rain.MM, err = zr.getRainData(zone)
			if err != nil {
				logger.WithError(err).Warn("unable to get rain data for Zone")
			} else {
				weatherData.Rain.ScaleFactor = zone.WaterSchedule.WeatherControl.Rain.InvertedScaleDownOnly(weatherData.Rain.MM)
			}
		}
		if zone.WaterSchedule.HasTemperatureControl() && zr.weatherClient != nil {
			weatherData.Temperature = &TemperatureData{}
			weatherData.Temperature.Celcius, err = zr.weatherClient.GetAverageHighTemperature(interval)
			if err != nil {
				logger.WithError(err).Warn("unable to get average high temperature from weather client")
			} else {
				weatherData.Temperature.ScaleFactor = zone.WaterSchedule.WeatherControl.Temperature.Scale(weatherData.Temperature.Celcius)
			}
		}
	}

	nextWateringDuration, err := time.ParseDuration(zone.WaterSchedule.Duration)
	if err != nil {
		logger.WithError(err).Warn("unable to determine next water duration")
	} else if zone.HasWeatherControl() && !zone.EndDated() {
		wd, err := zr.worker.ScaleWateringDuration(zone.WaterSchedule, int64(nextWateringDuration))
		if err != nil {
			logger.WithError(err).Warn("unable to determine water duration scale")
		} else {
			nextWateringDuration = time.Duration(wd)
		}
	}

	return &ZoneResponse{
		zone,
		weatherData,
		zr.worker.GetNextWaterTime(zone),
		nextWateringDuration.String(),
		links,
	}
}

func (zr ZonesResource) getRainData(zone *pkg.Zone) (float32, error) {
	intervalDuration, err := time.ParseDuration(zone.WaterSchedule.Interval)
	if err != nil {
		return 0, fmt.Errorf("unable to parse WaterSchedule duration for Zone")
	}

	totalRain, err := zr.weatherClient.GetTotalRain(intervalDuration)
	if err != nil {
		return 0, fmt.Errorf("unable to get rain data from weather client: %w", err)
	}
	return totalRain, nil
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
