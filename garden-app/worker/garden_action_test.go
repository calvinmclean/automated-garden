package worker

import (
	"errors"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGardenAction(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
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
				mqttClient.On("LightTopic", "garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"FailedGardenActionWithLightAction",
			&action.GardenAction{
				Light: &action.LightAction{},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("LightTopic", "garden").Return("", errors.New("template error"))
			},
			func(err error, t *testing.T) {
				if err == nil {
					t.Error("Expected error, but nil was returned")
				}
				if err.Error() != "unable to execute LightAction: unable to fill MQTT topic template: template error" {
					t.Errorf("Unexpected error string: %v", err)
				}
			},
		},
		{
			"SuccessfulGardenActionWithStopAction",
			&action.GardenAction{
				Stop: &action.StopAction{},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("StopTopic", "garden").Return("garden/action/stop", nil)
				mqttClient.On("Publish", "garden/action/stop", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"FailedGardenActionWithStopAction",
			&action.GardenAction{
				Stop: &action.StopAction{},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("StopTopic", "garden").Return("", errors.New("template error"))
			},
			func(err error, t *testing.T) {
				if err == nil {
					t.Error("Expected error, but nil was returned")
				}
				if err.Error() != "unable to execute StopAction: unable to fill MQTT topic template: template error" {
					t.Errorf("Unexpected error string: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient)

			err := NewWorker(nil, influxdbClient, mqttClient, logrus.New()).ExecuteGardenAction(garden, tt.action)
			tt.assert(err, t)
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestLightActionExecute(t *testing.T) {
	now := time.Now()
	garden := &pkg.Garden{
		ID:          xid.New(),
		Name:        "garden",
		TopicPrefix: "garden",
		LightSchedule: &pkg.LightSchedule{
			Duration:  &pkg.Duration{Duration: 15 * time.Hour},
			StartTime: "23:00:00-07:00",
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
				mqttClient.On("LightTopic", "garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"SuccessfulWithDelay",
			&action.LightAction{State: pkg.LightStateOff, ForDuration: &pkg.Duration{Duration: 30 * time.Second}},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("LightTopic", "garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"PublishError",
			&action.LightAction{State: pkg.LightStateOff, ForDuration: &pkg.Duration{Duration: 30 * time.Second}},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("LightTopic", "garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", mock.Anything).Return(errors.New("publish error"))
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
		{
			"TopicTemplateError",
			&action.LightAction{},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("LightTopic", "garden").Return("", errors.New("template error"))
			},
			func(err error, t *testing.T) {
				if err == nil {
					t.Error("Expected error, but nil was returned")
				}
				if err.Error() != "unable to fill MQTT topic template: template error" {
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

			worker := NewWorker(storageClient, influxdbClient, mqttClient, logrus.New())
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
				mqttClient.On("StopTopic", "garden").Return("garden/action/stop", nil)
				mqttClient.On("Publish", "garden/action/stop", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"SuccessfulStopAll",
			&action.StopAction{All: true},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("StopAllTopic", "garden").Return("garden/action/stop_all", nil)
				mqttClient.On("Publish", "garden/action/stop_all", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				assert.NoError(t, err)
			},
		},
		{
			"TopicTemplateError",
			&action.StopAction{},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("StopTopic", "garden").Return("", errors.New("template error"))
			},
			func(err error, t *testing.T) {
				if err == nil {
					t.Error("Expected error, but nil was returned")
				}
				if err.Error() != "unable to fill MQTT topic template: template error" {
					t.Errorf("Unexpected error string: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			tt.setupMock(mqttClient, influxdbClient)

			err := NewWorker(nil, influxdbClient, mqttClient, logrus.New()).ExecuteStopAction(garden, tt.action)
			tt.assert(err, t)
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
		})
	}
}
