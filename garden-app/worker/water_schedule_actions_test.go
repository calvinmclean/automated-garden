package worker

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestExecuteScheduledWaterAction(t *testing.T) {
	CreateNewID = func() xid.ID { return xid.NilID() }
	defer func() { CreateNewID = xid.New }()

	garden := &pkg.Garden{
		ID:          babyapi.ID{ID: id},
		Name:        "garden",
		TopicPrefix: "garden",
	}
	weatherClientID, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	temperatureControl := &weather.ScaleControl{
		BaselineValue: float32Pointer(70),
		Factor:        float32Pointer(0.5),
		Range:         float32Pointer(30),
		ClientID:      weatherClientID,
	}
	rainControl := &weather.ScaleControl{
		BaselineValue: float32Pointer(0),
		Factor:        float32Pointer(0),
		Range:         float32Pointer(50),
		ClientID:      weatherClientID,
	}

	tests := []struct {
		name          string
		waterSchedule *pkg.WaterSchedule
		zone          *pkg.Zone
		setupMock     func(*mqtt.MockClient, *influxdb.MockClient, *storage.Client)
		expectedError string
	}{
		{
			"Successful",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				mqttClient.On("Publish", "garden/command/water", mock.Anything).Return(nil)
			},
			"",
		},
		{
			"SuccessfulRainScaleToZero",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Rain: rainControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_mm":       50,
						"rain_interval": "24h",
					},
				})
				assert.NoError(t, err)
				// No MQTT calls made
			},
			"",
		},
		{
			"SuccessfulNoRainScaling",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Rain: rainControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_mm":       0,
						"rain_interval": "24h",
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":1000,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			"RainDelayErrorStillWaters",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Rain: rainControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_interval": "24h",
						"error":         "weather client error",
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":1000,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulRainPartialScaling",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Rain: rainControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_mm":       25,
						"rain_interval": "24h",
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":500,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulNoTemperatureScaling",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_interval":        "24h",
						"avg_high_temperature": 70,
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":1000,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulTemperaturePartialScaleUp",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_interval":        "24h",
						"avg_high_temperature": 85,
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":1250,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulTemperatureMaxScaleUp",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_interval":        "24h",
						"avg_high_temperature": 100,
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":1500,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulTemperatureMaxScaleUpPastMax",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_interval":        "24h",
						"avg_high_temperature": 120,
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":1500,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulTemperaturePartialScaleDown",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_interval":        "24h",
						"avg_high_temperature": 55,
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":750,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulTemperatureMaxScaleDown",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_interval":        "24h",
						"avg_high_temperature": 40,
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":500,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulTemperatureBeyondMaxScaleDown",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_interval":        "24h",
						"avg_high_temperature": 0,
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":500,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			"GetAverageTemperatureErrorStillWatersDefault",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_interval": "24h",
						"error":         "weather client error",
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":1000,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			// Scenario emulating summer where temperature causes increased watering, but
			// recent rain scales it down again
			"CompoundScalingSummerRain",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
					Rain:        rainControl,
				},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_interval":        "24h",
						"rain_mm":              25,
						"avg_high_temperature": 85,
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":625,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			// Scenario emulating winter where temperature causes decreased watering and
			// recent rain scales it down even more
			"CompoundScalingWinterRain",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
					Rain:        rainControl,
				},
			},
			&pkg.Zone{
				Position:  uintPointer(0),
				SkipCount: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Type: "fake",
					Options: map[string]interface{}{
						"rain_interval":        "24h",
						"rain_mm":              25,
						"avg_high_temperature": 55,
					},
				})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":375,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			"SkipCount>1WillSkip",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Rain: rainControl,
				},
			},
			&pkg.Zone{
				ID:        babyapi.ID{ID: id},
				Position:  uintPointer(0),
				SkipCount: uintPointer(1),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.Gardens.Set(context.Background(), &pkg.Garden{ID: babyapi.ID{ID: id}})
				assert.NoError(t, err)
				err = sc.Zones.Set(context.Background(), &pkg.Zone{ID: babyapi.ID{ID: id}, GardenID: id})
				assert.NoError(t, err)
				// no other mock calls are made because watering is skipped
			},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)
			defer weather.ResetCache()

			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient, sc)

			err = NewWorker(sc, influxdbClient, mqttClient, slog.Default()).ExecuteScheduledWaterAction(garden, tt.zone, tt.waterSchedule)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			} else {
				assert.NoError(t, err)
			}
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
		})
	}
}
