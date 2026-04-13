package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/concurrent"
	"github.com/calvinmclean/babyapi"
	"github.com/go-chi/render"
)

// ZoneResponse is used to represent a Zone in the response body with the additional data
// and hypermedia Links fields
type ZoneResponse struct {
	*pkg.Zone
	WeatherData *WeatherData     `json:"weather_data,omitempty"`
	NextWater   NextWaterDetails `json:"next_water,omitzero"`
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
	if r.Header.Get("HX-Request") == "true" && !strings.Contains(r.URL.Path, "/action") {
		referer := r.Header.Get("Referer")
		if strings.Contains(referer, "/zones/") && !strings.HasSuffix(referer, "/zones") {
			// We're on the zone details page, return just the WaterScheduleCard
			return waterScheduleCardTemplate.Render(r, zr)
		}
		// We're on the zones list page, return the ZoneCard
		return zoneCardTemplate.Render(r, zr)
	}

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

	// Fetch water schedules concurrently
	ws := zr.fetchWaterSchedules(ctx, logger)

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

	// Prepare tasks for concurrent execution
	var tasks []concurrent.TaskFunc

	// Add water history task for HTML responses
	if render.GetAcceptedContentType(r) == render.ContentTypeHTML {
		tasks = append(tasks, concurrent.TaskFunc{
			Name: "water-history",
			Fn: func(_ context.Context) error {
				history, apiErr := zr.api.getWaterHistoryFromRequest(r, zr.Zone, logger)
				if apiErr != nil {
					logger.Error("error getting water history", "error", apiErr)
					zr.HistoryError = apiErr.ErrorText
					return nil // Don't fail the whole request for history errors
				}
				zr.History = NewZoneWaterHistoryResponse(history, getLocationFromRequest(r))

				// Reverse history for better presentation in UI
				slices.Reverse(zr.History.History)

				progress := pkg.CalculateWaterProgress(zr.History.History)
				if progress != (pkg.WaterHistoryProgress{}) {
					zr.Progress = &progress
				}
				return nil
			},
		})
	}

	// Execute all tasks concurrently with timeout
	if len(tasks) > 0 {
		errors := concurrent.RunFuncs(ctx, influxDBTimeout, tasks)
		for taskName, err := range errors {
			logger.Warn("failed to execute task", "task", taskName, "error", err)
		}
	}

	if render.GetAcceptedContentType(r) == render.ContentTypeHTML && r.Method == http.MethodPut {
		w.Header().Add("HX-Trigger", "newZone")
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
		zr.WeatherData = zr.api.getCachedWeatherData(ctx, nextWaterSchedule, logger)
	}

	return nil
}

// fetchWaterSchedules fetches water schedules concurrently and returns them.
// It uses a timeout to prevent slow storage operations from blocking the response.
func (zr *ZoneResponse) fetchWaterSchedules(ctx context.Context, logger *slog.Logger) []*pkg.WaterSchedule {
	if len(zr.Zone.WaterScheduleIDs) == 0 {
		return nil
	}

	tasks := make([]concurrent.Task[*pkg.WaterSchedule], 0, len(zr.Zone.WaterScheduleIDs))
	for _, id := range zr.Zone.WaterScheduleIDs {
		scheduleID := id // capture loop variable
		tasks = append(tasks, concurrent.Task[*pkg.WaterSchedule]{
			Name: "water-schedule-" + scheduleID.String(),
			Fn: func(taskCtx context.Context) (*pkg.WaterSchedule, error) {
				return zr.api.storageClient.WaterSchedules.Get(taskCtx, scheduleID.String())
			},
		})
	}

	// Use a longer timeout for storage operations (they're typically fast but can vary)
	results := concurrent.Run(ctx, 5*time.Second, tasks)

	var schedules []*pkg.WaterSchedule
	for _, result := range results {
		if result.Error != nil {
			logger.Error("failed to get water schedule", "schedule", result.Name, "error", result.Error)
			continue
		}
		schedules = append(schedules, result.Value)
	}

	return schedules
}

type AllZonesResponse struct {
	babyapi.ResourceList[*ZoneResponse]

	api *babyapi.API[*pkg.Zone]
}

func (azr AllZonesResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return azr.ResourceList.Render(w, r)
}

// getLocationFromRequest reads the X-TZ-Offset header and returns the corresponding time.Location
func getLocationFromRequest(r *http.Request) *time.Location {
	tzHeader := r.Header.Get("X-TZ-Offset")
	if tzHeader != "" {
		loc, err := pkg.TimeLocationFromOffset(tzHeader)
		if err == nil {
			return loc
		}
	}
	return nil
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
// and converting times to the specified location
func NewZoneWaterHistoryResponse(history []pkg.WaterHistory, loc *time.Location) ZoneWaterHistoryResponse {
	total := time.Duration(0)
	count := 0
	for i, h := range history {
		if h.Status == pkg.WaterStatusCompleted {
			total += h.Duration.Duration
			count++
		}
		// Convert times to the target timezone
		if loc != nil {
			history[i].SentAt = h.SentAt.In(loc)
			history[i].StartedAt = h.StartedAt.In(loc)
			history[i].CompletedAt = h.CompletedAt.In(loc)
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

type ZoneActionResponse struct{}

func (*ZoneActionResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
