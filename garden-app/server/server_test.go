package server

import (
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/assert"
)

func TestValidateAllStoredResources(t *testing.T) {
	tests := []struct {
		name          string
		initStorage   func(*storage.Client) error
		expectedError string
	}{
		{
			"EmptySuccess",
			func(_ *storage.Client) error { return nil },
			"",
		},
		{
			"InvalidGardenMissingID",
			func(s *storage.Client) error {
				return s.Gardens.Set(&pkg.Garden{})
			},
			"invalid Garden: missing required field 'id'",
		},
		{
			"InvalidGarden",
			func(s *storage.Client) error {
				return s.Gardens.Set(&pkg.Garden{
					ID: babyapi.ID{ID: id},
				})
			},
			"invalid Garden \"c5cvhpcbcv45e8bp16dg\": missing required name field",
		},
		{
			"InvalidZone",
			func(s *storage.Client) error {
				g := createExampleGarden()
				err := s.Gardens.Set(g)
				if err != nil {
					return err
				}

				return s.Zones.Set(&pkg.Zone{ID: babyapi.ID{ID: id}, GardenID: g.ID.ID})
			},
			"invalid Zone \"c5cvhpcbcv45e8bp16dg\": missing required position field",
		},
		{
			"InvalidWaterScheduleMissingID",
			func(s *storage.Client) error {
				return s.WaterSchedules.Set(&pkg.WaterSchedule{})
			},
			"invalid WaterSchedule: missing required field 'id'",
		},
		{
			"InvalidWaterSchedule",
			func(s *storage.Client) error {
				return s.WaterSchedules.Set(&pkg.WaterSchedule{
					ID: babyapi.ID{ID: id},
				})
			},
			"invalid WaterSchedule \"c5cvhpcbcv45e8bp16dg\": missing required interval field",
		},
		{
			"InvalidWeatherClientMissingID",
			func(s *storage.Client) error {
				return s.WeatherClientConfigs.Set(&weather.Config{})
			},
			"invalid WeatherClient: missing required field 'id'",
		},
		{
			"InvalidWeatherClient",
			func(s *storage.Client) error {
				return s.WeatherClientConfigs.Set(&weather.Config{
					ID: babyapi.ID{ID: id},
				})
			},
			"invalid WeatherClient \"c5cvhpcbcv45e8bp16dg\": missing required type field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			err = tt.initStorage(storageClient)
			assert.NoError(t, err)

			err = validateAllStoredResources(storageClient)
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			}
		})
	}
}
