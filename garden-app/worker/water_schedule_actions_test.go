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

func float64Ptr(f float64) *float64              { return &f }
func durationPtr(d time.Duration) *time.Duration { return &d }

func TestExecuteScheduledWaterAction(t *testing.T) {
	CreateNewID = func() xid.ID { return xid.NilID() }
	defer func() { CreateNewID = xid.New }()

	garden := &pkg.Garden{
		ID:          babyapi.ID{ID: id},
		Name:        "garden",
		TopicPrefix: "garden",
	}

	tests := []struct {
		name          string
		waterSchedule *pkg.WaterSchedule
		zone          *pkg.Zone
		duration      *time.Duration // nil means use waterSchedule.Duration
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
			nil,
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				mqttClient.On("Publish", "garden/command/water", mock.Anything).Return(nil)
			},
			"",
		},
		{
			"ZeroDurationSkipsWatering",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			durationPtr(0),
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				// No MQTT calls made when duration is 0
			},
			"",
		},
		{
			"WithScaledDuration",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
			},
			&pkg.Zone{
				Position: uintPointer(0),
			},
			durationPtr(500 * time.Millisecond),
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				mqttClient.On("Publish", "garden/command/water", []byte(`{"duration":500,"zone_id":"00000000000000000000","position":0,"id":"00000000000000000000","source":"schedule"}`)).Return(nil)
			},
			"",
		},
		{
			"SkipCountGreaterThanZero",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
			},
			&pkg.Zone{
				ID:        babyapi.ID{ID: id},
				Position:  uintPointer(0),
				SkipCount: uintPointer(1),
			},
			nil,
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.Gardens.Set(context.Background(), &pkg.Garden{ID: babyapi.ID{ID: id}})
				assert.NoError(t, err)
				err = sc.Zones.Set(context.Background(), &pkg.Zone{ID: babyapi.ID{ID: id}, GardenID: id})
				assert.NoError(t, err)
				// no MQTT calls because watering is skipped
			},
			"",
		},
		{
			"SkipCountZeroDoesNotSkip",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
			},
			&pkg.Zone{
				ID:        babyapi.ID{ID: id},
				Position:  uintPointer(0),
				SkipCount: uintPointer(0),
			},
			nil,
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, sc *storage.Client) {
				err := sc.Gardens.Set(context.Background(), &pkg.Garden{ID: babyapi.ID{ID: id}})
				assert.NoError(t, err)
				err = sc.Zones.Set(context.Background(), &pkg.Zone{ID: babyapi.ID{ID: id}, GardenID: id})
				assert.NoError(t, err)
				mqttClient.On("Publish", "garden/command/water", mock.Anything).Return(nil)
			},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)
			defer weather.ResetCache()

			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient, sc)

			duration := tt.waterSchedule.Duration.Duration
			if tt.duration != nil {
				duration = *tt.duration
			}

			err = NewWorker(sc, influxdbClient, mqttClient, slog.Default()).ExecuteScheduledWaterAction(garden, tt.zone, tt.waterSchedule, duration)
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

func TestScaleWateringDuration(t *testing.T) {
	weatherClientID, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	temperatureControl := &weather.WeatherScaler{
		ClientID:      weatherClientID,
		Interpolation: weather.Linear,
		InputMin:      float64Ptr(40),
		InputMax:      float64Ptr(100),
		FactorMin:     float64Ptr(0.5),
		FactorMax:     float64Ptr(1.5),
	}
	rainControl := &weather.WeatherScaler{
		ClientID:      weatherClientID,
		Interpolation: weather.Linear,
		InputMin:      float64Ptr(0),
		InputMax:      float64Ptr(50),
		FactorMin:     float64Ptr(1.0),
		FactorMax:     float64Ptr(0.0),
	}

	tests := []struct {
		name             string
		waterSchedule    *pkg.WaterSchedule
		setupWeather     func(*storage.Client)
		expectedDuration time.Duration
	}{
		{
			name: "NoWeatherControl",
			waterSchedule: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
			},
			setupWeather:     func(sc *storage.Client) {},
			expectedDuration: time.Second,
		},
		{
			name: "RainScaleToZero",
			waterSchedule: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Rain: rainControl,
				},
			},
			setupWeather: func(sc *storage.Client) {
				_ = sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Name: "test",
					Type: "fake",
					Options: map[string]any{
						"rain_mm":       50,
						"rain_interval": "24h",
					},
				})
			},
			expectedDuration: 0,
		},
		{
			name: "NoRainScaling",
			waterSchedule: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Rain: rainControl,
				},
			},
			setupWeather: func(sc *storage.Client) {
				_ = sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Name: "test",
					Type: "fake",
					Options: map[string]any{
						"rain_mm":       0,
						"rain_interval": "24h",
					},
				})
			},
			expectedDuration: time.Second,
		},
		{
			name: "RainPartialScaling",
			waterSchedule: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Rain: rainControl,
				},
			},
			setupWeather: func(sc *storage.Client) {
				_ = sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Name: "test",
					Type: "fake",
					Options: map[string]any{
						"rain_mm":       25,
						"rain_interval": "24h",
					},
				})
			},
			expectedDuration: 500 * time.Millisecond,
		},
		{
			name: "TemperaturePartialScaleUp",
			waterSchedule: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
				},
			},
			setupWeather: func(sc *storage.Client) {
				_ = sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Name: "test",
					Type: "fake",
					Options: map[string]any{
						"rain_interval":        "24h",
						"avg_high_temperature": 85,
					},
				})
			},
			expectedDuration: 1250 * time.Millisecond,
		},
		{
			name: "TemperatureMaxScaleUp",
			waterSchedule: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
				},
			},
			setupWeather: func(sc *storage.Client) {
				_ = sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Name: "test",
					Type: "fake",
					Options: map[string]any{
						"rain_interval":        "24h",
						"avg_high_temperature": 100,
					},
				})
			},
			expectedDuration: 1500 * time.Millisecond,
		},
		{
			name: "TemperaturePartialScaleDown",
			waterSchedule: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
				},
			},
			setupWeather: func(sc *storage.Client) {
				_ = sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Name: "test",
					Type: "fake",
					Options: map[string]any{
						"rain_interval":        "24h",
						"avg_high_temperature": 55,
					},
				})
			},
			expectedDuration: 750 * time.Millisecond,
		},
		{
			name: "TemperatureMaxScaleDown",
			waterSchedule: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
				},
			},
			setupWeather: func(sc *storage.Client) {
				_ = sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Name: "test",
					Type: "fake",
					Options: map[string]any{
						"rain_interval":        "24h",
						"avg_high_temperature": 40,
					},
				})
			},
			expectedDuration: 500 * time.Millisecond,
		},
		{
			name: "CompoundScalingSummerRain",
			waterSchedule: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
					Rain:        rainControl,
				},
			},
			setupWeather: func(sc *storage.Client) {
				_ = sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Name: "test",
					Type: "fake",
					Options: map[string]any{
						"rain_interval":        "24h",
						"rain_mm":              25,
						"avg_high_temperature": 85,
					},
				})
			},
			expectedDuration: 625 * time.Millisecond,
		},
		{
			name: "CompoundScalingWinterRain",
			waterSchedule: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Temperature: temperatureControl,
					Rain:        rainControl,
				},
			},
			setupWeather: func(sc *storage.Client) {
				_ = sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Name: "test",
					Type: "fake",
					Options: map[string]any{
						"rain_interval":        "24h",
						"rain_mm":              25,
						"avg_high_temperature": 55,
					},
				})
			},
			expectedDuration: 375 * time.Millisecond,
		},
		{
			name: "WeatherClientErrorReturnsBaseDuration",
			waterSchedule: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					Rain: rainControl,
				},
			},
			setupWeather: func(sc *storage.Client) {
				_ = sc.WeatherClientConfigs.Set(context.Background(), &weather.Config{
					ID:   babyapi.ID{ID: weatherClientID},
					Name: "test",
					Type: "fake",
					Options: map[string]any{
						"rain_interval": "24h",
						"error":         "weather client error",
					},
				})
			},
			expectedDuration: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)
			defer weather.ResetCache()

			tt.setupWeather(sc)

			worker := NewWorker(sc, new(influxdb.MockClient), new(mqtt.MockClient), slog.Default())
			duration, _ := worker.ScaleWateringDuration(tt.waterSchedule)

			assert.Equal(t, tt.expectedDuration, duration)
		})
	}
}
