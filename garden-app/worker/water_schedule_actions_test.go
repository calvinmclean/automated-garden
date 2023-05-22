package worker

import (
	"errors"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestExecuteScheduledWaterAction(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
	}
	zone := &pkg.Zone{
		Position: uintPointer(0),
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

	fifty := 50

	tests := []struct {
		name          string
		waterSchedule *pkg.WaterSchedule
		setupMock     func(*mqtt.MockClient, *influxdb.MockClient, *weather.MockClient, *storage.MockClient)
		expectedError string
	}{
		{
			"Successful",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
			},
			"",
		},
		{
			"TopicTemplateError",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				mqttClient.On("WaterTopic", "garden").Return("", errors.New("template error"))
			},
			"unable to fill MQTT topic template: template error",
		},
		{
			"SuccessWhenMoistureLessThanMinimum",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					SoilMoisture: &weather.SoilMoistureControl{
						MinimumMoisture: &fifty,
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
				influxdbClient.On("GetMoisture", mock.Anything, uint(0), garden.Name).Return(float64(0), nil)
				influxdbClient.On("Close")
			},
			"",
		},
		{
			"SuccessWhenMoistureGreaterThanMinimum",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					SoilMoisture: &weather.SoilMoistureControl{
						MinimumMoisture: &fifty,
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				influxdbClient.On("GetMoisture", mock.Anything, uint(0), garden.Name).Return(float64(51), nil)
				influxdbClient.On("Close")
				// No MQTT calls made
			},
			"",
		},
		{
			"InfluxDBClientErrorStillWaters",
			&pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: time.Second},
				Interval: &pkg.Duration{Duration: time.Hour * 24},
				WeatherControl: &weather.Control{
					SoilMoisture: &weather.SoilMoistureControl{
						MinimumMoisture: &fifty,
					},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
				influxdbClient.On("GetMoisture", mock.Anything, uint(0), garden.Name).Return(float64(0), errors.New("influxdb error"))
				influxdbClient.On("Close")
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetTotalRain", mock.Anything).Return(float32(50), nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetTotalRain", mock.Anything).Return(float32(0), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1000,"id":null,"position":0}`)).Return(nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetTotalRain", mock.Anything).Return(float32(0), errors.New("weather client error"))
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1000,"id":null,"position":0}`)).Return(nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetTotalRain", mock.Anything).Return(float32(25), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":500,"id":null,"position":0}`)).Return(nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(70), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1000,"id":null,"position":0}`)).Return(nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(85), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1250,"id":null,"position":0}`)).Return(nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(100), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1500,"id":null,"position":0}`)).Return(nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(120), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1500,"id":null,"position":0}`)).Return(nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(55), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":750,"id":null,"position":0}`)).Return(nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(40), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":500,"id":null,"position":0}`)).Return(nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(0), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":500,"id":null,"position":0}`)).Return(nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(0), errors.New("weather client error"))
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":1000,"id":null,"position":0}`)).Return(nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetTotalRain", mock.Anything).Return(float32(25), nil)
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(85), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":625,"id":null,"position":0}`)).Return(nil)
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
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient, sc *storage.MockClient) {
				sc.On("GetWeatherClient", weatherClientID).Return(wc, nil)
				wc.On("GetTotalRain", mock.Anything).Return(float32(25), nil)
				wc.On("GetAverageHighTemperature", mock.Anything).Return(float32(55), nil)
				mqttClient.On("WaterTopic", "garden").Return("garden/action/water", nil)
				mqttClient.On("Publish", "garden/action/water", []byte(`{"duration":375,"id":null,"position":0}`)).Return(nil)
			},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			wc := new(weather.MockClient)
			sc := new(storage.MockClient)
			tt.setupMock(mqttClient, influxdbClient, wc, sc)

			err := NewWorker(sc, influxdbClient, mqttClient, logrus.New()).ExecuteScheduledWaterAction(garden, zone, tt.waterSchedule)
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
			} else {
				assert.NoError(t, err)
			}
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
			wc.AssertExpectations(t)
			sc.AssertExpectations(t)
		})
	}
}
