package worker

import (
	"errors"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestZoneAction(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
	}

	tests := []struct {
		name          string
		action        *action.ZoneAction
		setupMock     func(*mqtt.MockClient, *influxdb.MockClient)
		expectedError string
	}{
		{
			"SuccessfulEmptyZoneAction",
			&action.ZoneAction{},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {},
			"",
		},
		{
			"SuccessfulZoneActionWithWaterAction",
			&action.ZoneAction{
				Water: &action.WaterAction{
					Duration: 1000,
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
			},
			"",
		},
		{
			"FailedZoneActionWithWaterAction",
			&action.ZoneAction{
				Water: &action.WaterAction{
					Duration: 1000,
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("WaterTopic", "garden").Return("", errors.New("template error"))
			},
			"unable to execute WaterAction: unable to fill MQTT topic template: template error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone := &pkg.Zone{
				Position:      uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{},
			}
			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient)

			err := NewWorker(nil, influxdbClient, mqttClient, nil, logrus.New()).ExecuteZoneAction(garden, zone, tt.action)
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

func TestWaterActionExecute(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
	}
	action := &action.WaterAction{
		Duration: 1000,
	}
	temperatureControl := &weather.ScaleControl{
		BaselineTemperature: float32Pointer(70),
		Factor:              float32Pointer(0.5),
		Range:               float32Pointer(30),
	}

	tests := []struct {
		name          string
		zone          *pkg.Zone
		setupMock     func(*mqtt.MockClient, *influxdb.MockClient, *weather.MockClient)
		expectedError string
	}{
		{
			"Successful",
			&pkg.Zone{
				Position:      uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
			},
			"",
		},
		{
			"TopicTemplateError",
			&pkg.Zone{
				Position:      uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				mqttClient.On("WaterTopic", "garden").Return("", errors.New("template error"))
			},
			"unable to fill MQTT topic template: template error",
		},
		{
			"SuccessWhenMoistureLessThanMinimum",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						SoilMoisture: &weather.SoilMoistureControl{
							MinimumMoisture: 50,
						},
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
				influxdbClient.On("GetMoisture", mock.Anything, uint(0), garden.Name).Return(float64(0), nil)
				influxdbClient.On("Close")
			},
			"",
		},
		{
			"SuccessWhenMoistureGreaterThanMinimum",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						SoilMoisture: &weather.SoilMoistureControl{
							MinimumMoisture: 50,
						},
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				influxdbClient.On("GetMoisture", mock.Anything, uint(0), garden.Name).Return(float64(51), nil)
				influxdbClient.On("Close")
				// No MQTT calls made
			},
			"",
		},
		{
			"InfluxDBClientErrorStillWaters",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						SoilMoisture: &weather.SoilMoistureControl{
							MinimumMoisture: 50,
						},
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
				influxdbClient.On("GetMoisture", mock.Anything, uint(0), garden.Name).Return(float64(0), errors.New("influxdb error"))
				influxdbClient.On("Close")
			},
			"",
		},
		{
			"SuccessfulRainDelay",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						Rain: &weather.RainControl{
							Threshold: 25.4,
						},
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				wc.On("GetTotalRain", mock.Anything).Return(float32(25.4), nil)
				// No MQTT calls made
			},
			"",
		},
		{
			"SuccessfulNoRainDelay",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						Rain: &weather.RainControl{
							Threshold: 25.4,
						},
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				wc.On("GetTotalRain", mock.Anything).Return(float32(14), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
			},
			"",
		},
		{
			"RainDelayErrorStillWaters",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						Rain: &weather.RainControl{
							Threshold: 25.4,
						},
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				wc.On("GetTotalRain", mock.Anything).Return(float32(0), errors.New("weather client error"))
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
			},
			"",
		},
		{
			"RainDelayErrorParsingIntervalStillWaters",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "wow",
					WeatherControl: &weather.Control{
						Rain: &weather.RainControl{
							Threshold: 25.4,
						},
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
			},
			"",
		},
		{
			"SuccessfulNoTemperatureScaling",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						Temperature: temperatureControl,
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(70), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1000,"id":null,"position":0}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulTemperaturePartialScaleUp",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						Temperature: temperatureControl,
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(85), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1250,"id":null,"position":0}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulTemperatureMaxScaleUp",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						Temperature: temperatureControl,
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(100), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1500,"id":null,"position":0}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulTemperatureMaxScaleUpPastMax",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						Temperature: temperatureControl,
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(120), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1500,"id":null,"position":0}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulTemperaturePartialScaleDown",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						Temperature: temperatureControl,
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(55), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":750,"id":null,"position":0}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulTemperatureMaxScaleDown",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						Temperature: temperatureControl,
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(40), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":500,"id":null,"position":0}`)).Return(nil)
			},
			"",
		},
		{
			"SuccessfulTemperatureBeyondMaxScaleDown",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						Temperature: temperatureControl,
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(0), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":500,"id":null,"position":0}`)).Return(nil)
			},
			"",
		},
		{
			"GetAverageTemperatureErrorStillWatersDefault",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "24h",
					WeatherControl: &weather.Control{
						Temperature: temperatureControl,
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(0), errors.New("weather client error"))
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1000,"id":null,"position":0}`)).Return(nil)
			},
			"",
		},
		{
			"ErrorParsingIntervalForTemperatureControlStillWaters",
			&pkg.Zone{
				Position: uintPointer(0),
				WaterSchedule: &pkg.WaterSchedule{
					Interval: "wow",
					WeatherControl: &weather.Control{
						Temperature: temperatureControl,
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1000,"id":null,"position":0}`)).Return(nil)
			},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			wc := new(weather.MockClient)
			tt.setupMock(mqttClient, influxdbClient, wc)

			err := NewWorker(nil, influxdbClient, mqttClient, wc, logrus.New()).ExecuteWaterAction(garden, tt.zone, action)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			} else {
				assert.NoError(t, err)
			}
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
			wc.AssertExpectations(t)
		})
	}
}

func uintPointer(n int) *uint {
	uintn := uint(n)
	return &uintn
}

func float32Pointer(n float64) *float32 {
	f := float32(n)
	return &f
}
