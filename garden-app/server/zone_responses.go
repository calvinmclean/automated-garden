package server

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
)

// ZoneResponse is used to represent a Zone in the response body with the additional data
// and hypermedia Links fields
type ZoneResponse struct {
	*pkg.Zone
	WeatherData *WeatherData     `json:"weather_data,omitempty"`
	NextWater   NextWaterDetails `json:"next_water,omitempty"`
	Links       []Link           `json:"links,omitempty"`

	// History is only used in HTML responses and is excluded from JSON
	History      ZoneWaterHistoryResponse  `json:"-"`
	HistoryError string                    `json:"-"`
	Progress     *pkg.WaterHistoryProgress `json:"-"`

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

func (zr *ZoneResponse) HTML(_ http.ResponseWriter, r *http.Request) string {
	// ignoring errors here since this can only be reached for a valid request
	timeRange, _ := rangeQueryParam(r)
	limit, _ := limitQueryParam(r)

	return zoneDetailsTemplate.Render(r, map[string]any{
		"TimeRange": timeRange,
		"Limit":     limit,
		"Response":  zr,
	})
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (zr *ZoneResponse) Render(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	excludeWeatherData := excludeWeatherData(r)

	logger := babyapi.GetLoggerFromContext(r.Context())

	ws := []*pkg.WaterSchedule{}
	for _, id := range zr.Zone.WaterScheduleIDs {
		result, err := zr.api.storageClient.WaterSchedules.Get(ctx, id.String())
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

	if render.GetAcceptedContentType(r) == render.ContentTypeHTML {
		history, apiErr := zr.api.getWaterHistoryFromRequest(r, zr.Zone, logger)
		if apiErr != nil {
			logger.Error("error getting water history", "error", apiErr)
			zr.HistoryError = apiErr.ErrorText
		}
		zr.History = NewZoneWaterHistoryResponse(history)

		// Reverse history for better presentation in UI
		slices.Reverse(zr.History.History)

		progress := pkg.CalculateWaterProgress(zr.History.History)
		if progress != (pkg.WaterHistoryProgress{}) {
			zr.Progress = &progress
		}

		if r.Method == http.MethodPut {
			w.Header().Add("HX-Trigger", "newZone")
		}
	}

	nextWaterSchedule := zr.api.worker.GetNextActiveWaterSchedule(ws)

	if nextWaterSchedule == nil {
		zr.NextWater = NextWaterDetails{
			Message: "no active WaterSchedules",
		}
		return nil
	}

	zr.NextWater = GetNextWaterDetails(r, nextWaterSchedule, zr.api.worker, excludeWeatherData)
	zr.NextWater.WaterScheduleID = &nextWaterSchedule.ID.ID

	if zr.Zone.SkipCount != nil && *zr.Zone.SkipCount > 0 {
		zr.NextWater.Message = fmt.Sprintf("skip_count %d affected the time", *zr.Zone.SkipCount)
		//nolint:gosec
		newNextTime := zr.NextWater.Time.Add(time.Duration(*zr.Zone.SkipCount) * nextWaterSchedule.Interval.Duration)
		zr.NextWater.Time = &newNextTime
	}

	if nextWaterSchedule.HasWeatherControl() && !excludeWeatherData {
		zr.WeatherData = getWeatherData(ctx, nextWaterSchedule, zr.api.storageClient)
	}

	return nil
}

type AllZonesResponse struct {
	babyapi.ResourceList[*ZoneResponse]

	api *babyapi.API[*pkg.Zone]
}

func (azr AllZonesResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return azr.ResourceList.Render(w, r)
}

func (azr AllZonesResponse) HTML(_ http.ResponseWriter, r *http.Request) string {
	slices.SortFunc(azr.Items, func(z *ZoneResponse, zz *ZoneResponse) int {
		return strings.Compare(z.Name, zz.Name)
	})

	garden, err := babyapi.GetResourceFromContext[*pkg.Garden](r.Context(), azr.api.ParentContextKey())
	if err != nil {
		panic(err)
	}

	data := map[string]any{
		"Items":  azr.Items,
		"Garden": garden,
	}

	if r.URL.Query().Get("refresh") == "true" {
		return zonesTemplate.Render(r, data)
	}

	return zonesPageTemplate.Render(r, data)
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
	count := 0
	for _, h := range history {
		if h.Status == pkg.WaterStatusCompleted {
			total += h.Duration.Duration
			count++
		}
	}

	average := time.Duration(0)
	if count != 0 {
		average = time.Duration(int(total) / count)
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
