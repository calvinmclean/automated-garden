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
	Moisture         float64    `json:"moisture,omitempty"`
	NextWateringTime *time.Time `json:"next_watering_time,omitempty"`
	Links            []Link     `json:"links,omitempty"`
}

// NewZoneResponse creates a self-referencing ZoneResponse
func (zr ZonesResource) NewZoneResponse(ctx context.Context, garden *pkg.Garden, zone *pkg.Zone, links ...Link) *ZoneResponse {
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
	moisture := 0.0
	var err error
	if zone.WaterSchedule.MinimumMoisture > 0 && garden != nil {
		moisture, err = zr.getMoisture(ctx, garden, zone)
		if err != nil {
			// Log moisture error but do not return an error since this isn't critical information
			logger.Errorf("unable to get moisture of Zone %v: %v", zone.ID, err)
		}
	}
	return &ZoneResponse{
		zone,
		moisture,
		zr.scheduler.GetNextWateringTime(zone),
		links,
	}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (z *ZoneResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// ZoneWateringHistoryResponse wraps a slice of WateringHistory structs plus some aggregate stats for an HTTP response
type ZoneWateringHistoryResponse struct {
	History []pkg.WateringHistory `json:"history"`
	Count   int                   `json:"count"`
	Average string                `json:"average"`
	Total   string                `json:"total"`
}

// NewZoneWateringHistoryResponse creates a response by creating some basic statistics about a list of history events
func NewZoneWateringHistoryResponse(history []pkg.WateringHistory) ZoneWateringHistoryResponse {
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
	return ZoneWateringHistoryResponse{
		History: history,
		Count:   count,
		Average: average.String(),
		Total:   time.Duration(total).String(),
	}
}

// Render is used to make this struct compatible with the go-chi webserver for writing
// the JSON response
func (resp ZoneWateringHistoryResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}