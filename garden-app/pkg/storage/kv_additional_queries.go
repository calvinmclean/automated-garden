package storage

import (
	"context"
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
)

// KVAdditionalQueries implements AdditionalQueries interface for KV storage
type KVAdditionalQueries struct {
	Gardens        babyapi.Storage[*pkg.Garden]
	Zones          babyapi.Storage[*pkg.Zone]
	WaterSchedules babyapi.Storage[*pkg.WaterSchedule]
}

// GetZonesUsingWaterSchedule will find all Zones that use this WaterSchedule and return the Zones along with the Gardens they belong to
func (k *KVAdditionalQueries) GetZonesUsingWaterSchedule(id string) ([]*pkg.ZoneAndGarden, error) {
	gardens, err := k.Gardens.Search(context.Background(), "", babyapi.EndDatedQueryParam(false))
	if err != nil {
		return nil, fmt.Errorf("unable to get all Gardens: %w", err)
	}

	results := []*pkg.ZoneAndGarden{}
	for _, g := range gardens {
		zones, err := k.Zones.Search(context.Background(), g.GetID(), nil)
		if err != nil {
			return nil, fmt.Errorf("unable to get all Zones for Garden %q: %w", g.ID, err)
		}
		zones = babyapi.FilterFunc[*pkg.Zone](func(z *pkg.Zone) bool {
			if z.EndDated() {
				return false
			}
			for _, wsID := range z.WaterScheduleIDs {
				if wsID.String() == id {
					return true
				}
			}
			return false
		}).Filter(zones)

		for _, z := range zones {
			results = append(results, &pkg.ZoneAndGarden{Zone: z, Garden: g})
		}
	}

	return results, nil
}

// GetWaterSchedulesUsingWeatherClient will return all WaterSchedules that rely on this WeatherClient
func (k *KVAdditionalQueries) GetWaterSchedulesUsingWeatherClient(id string) ([]*pkg.WaterSchedule, error) {
	waterSchedules, err := k.WaterSchedules.Search(context.Background(), "", nil)
	if err != nil {
		return nil, fmt.Errorf("unable to get all WaterSchedules: %w", err)
	}

	waterSchedules = babyapi.FilterFunc[*pkg.WaterSchedule](func(ws *pkg.WaterSchedule) bool {
		if !ws.HasWeatherControl() {
			return false
		}
		if ws.HasRainControl() && ws.WeatherControl.Rain.ClientID.String() == id {
			return true
		}
		if ws.HasTemperatureControl() && ws.WeatherControl.Temperature.ClientID.String() == id {
			return true
		}
		return false
	}).Filter(waterSchedules)

	return waterSchedules, nil
}
