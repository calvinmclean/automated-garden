package worker

import (
	"log/slog"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
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
					Duration: &pkg.Duration{Duration: 1000},
				},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", "garden/command/water", mock.Anything).Return(nil)
			},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone := &pkg.Zone{
				Position: uintPointer(0),
			}
			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient)

			err := NewWorker(nil, influxdbClient, mqttClient, slog.Default()).ExecuteZoneAction(garden, zone, tt.action)
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
		Duration: &pkg.Duration{Duration: time.Second},
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
				Position: uintPointer(0),
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, wc *weather.MockClient) {
				mqttClient.On("Publish", "garden/command/water", mock.Anything).Return(nil)
			},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)

			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			wc := new(weather.MockClient)
			tt.setupMock(mqttClient, influxdbClient, wc)

			err = NewWorker(storageClient, influxdbClient, mqttClient, slog.Default()).ExecuteWaterAction(garden, tt.zone, action)
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

func uintPointer(n uint) *uint {
	uintn := uint(n)
	return &uintn
}

func float32Pointer(n float64) *float32 {
	f := float32(n)
	return &f
}
