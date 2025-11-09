package worker

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGardenAction(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
		ControllerConfig: &pkg.ControllerConfig{
			ValvePins: []uint{1, 2, 3},
		},
	}

	tests := []struct {
		name      string
		action    *action.GardenAction
		setupMock func(*mqtt.MockClient, *influxdb.MockClient)
		assert    func(error, *testing.T)
	}{
		{
			"SuccessfulGardenActionWithLightAction",
			&action.GardenAction{
				Light: &action.LightAction{},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", "garden/command/light", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"SuccessfulGardenActionWithStopAction",
			&action.GardenAction{
				Stop: &action.StopAction{},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", "garden/command/stop", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"SuccessfulGardenActionWithUpdateAction",
			&action.GardenAction{
				Update: &action.UpdateAction{Config: true},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", "garden/command/update_config", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"UpdateActionErrorFalse",
			&action.GardenAction{
				Update: &action.UpdateAction{Config: false},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {},
			func(err error, t *testing.T) {
				assert.Error(t, err)
				assert.Equal(t, "unable to execute UpdateActin: update action must have config=true", err.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient)

			err := NewWorker(nil, influxdbClient, mqttClient, slog.Default()).ExecuteGardenAction(garden, tt.action)
			tt.assert(err, t)
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestLightActionExecute(t *testing.T) {
	now := clock.Now()
	startTime, _ := pkg.StartTimeFromString("23:00:00-07:00")
	garden := &pkg.Garden{
		ID:          babyapi.NewID(),
		Name:        "garden",
		TopicPrefix: "garden",
		LightSchedule: &pkg.LightSchedule{
			Duration:  &pkg.Duration{Duration: 15 * time.Hour},
			StartTime: startTime,
		},
		CreatedAt: &now,
	}

	tests := []struct {
		name      string
		action    *action.LightAction
		setupMock func(*mqtt.MockClient, *influxdb.MockClient)
		assert    func(error, *testing.T)
	}{
		{
			"Successful",
			&action.LightAction{},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", "garden/command/light", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"PublishError",
			&action.LightAction{State: pkg.LightStateOff},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", "garden/command/light", mock.Anything).Return(errors.New("publish error"))
			},
			func(err error, t *testing.T) {
				if err == nil {
					t.Error("Expected error, but nil was returned")
				}
				if err.Error() != "unable to publish LightAction: publish error" {
					t.Errorf("Unexpected error string: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)
			defer weather.ResetCache()

			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient)
			mqttClient.On("Disconnect", uint(100)).Return()
			influxdbClient.On("Close").Return()

			worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
			err = worker.ScheduleLightActions(garden)
			assert.NoError(t, err)
			worker.StartAsync()

			err = worker.ExecuteLightAction(garden, tt.action)
			tt.assert(err, t)

			worker.Stop()
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestStopActionExecute(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
	}

	tests := []struct {
		name      string
		action    *action.StopAction
		setupMock func(*mqtt.MockClient, *influxdb.MockClient)
		assert    func(error, *testing.T)
	}{
		{
			"Successful",
			&action.StopAction{},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", "garden/command/stop", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"SuccessfulStopAll",
			&action.StopAction{All: true},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("Publish", "garden/command/stop_all", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient)

			err := NewWorker(nil, influxdbClient, mqttClient, slog.Default()).ExecuteStopAction(garden, tt.action)
			tt.assert(err, t)
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
		})
	}
}
