package action

import (
	"errors"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/stretchr/testify/mock"
)

func TestGardenAction(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
	}

	tests := []struct {
		name      string
		action    *GardenAction
		setupMock func(*mqtt.MockClient, *influxdb.MockClient)
		assert    func(error, *testing.T)
	}{
		{
			"SuccessfulGardenActionWithLightAction",
			&GardenAction{
				Light: &LightAction{},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("LightTopic", "garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				if err != nil {
					t.Errorf("Unexpected error occurred when executing GardenAction: %v", err)
				}
			},
		},
		{
			"FailedGardenActionWithLightAction",
			&GardenAction{
				Light: &LightAction{},
			},
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
		{
			"SuccessfulGardenActionWithStopAction",
			&GardenAction{
				Stop: &StopAction{},
			},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("StopTopic", "garden").Return("garden/action/stop", nil)
				mqttClient.On("Publish", "garden/action/stop", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				if err != nil {
					t.Errorf("Unexpected error occurred when executing GardenAction: %v", err)
				}
			},
		},
		{
			"FailedGardenActionWithStopAction",
			&GardenAction{
				Stop: &StopAction{},
			},
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

			err := tt.action.Execute(garden, nil, NewScheduler(nil, influxdbClient, mqttClient))
			tt.assert(err, t)
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
		})
	}
}

func TestLightActionExecute(t *testing.T) {
	garden := &pkg.Garden{
		Name:        "garden",
		TopicPrefix: "garden",
		LightSchedule: &pkg.LightSchedule{
			Duration:  "15h",
			StartTime: "23:00:00-07:00",
		},
	}

	tests := []struct {
		name      string
		action    *LightAction
		setupMock func(*mqtt.MockClient, *influxdb.MockClient, *MockScheduler)
		assert    func(error, *testing.T)
	}{
		{
			"Successful",
			&LightAction{},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, scheduler *MockScheduler) {
				mqttClient.On("LightTopic", "garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				if err != nil {
					t.Errorf("Unexpected error occurred when executing LightAction: %v", err)
				}
			},
		},
		{
			"SuccessfulWithDelay",
			&LightAction{State: pkg.LightStateOff, ForDuration: "30s"},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, scheduler *MockScheduler) {
				mqttClient.On("LightTopic", "garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", mock.Anything).Return(nil)
				scheduler.On("ScheduleLightDelay", mock.AnythingOfType("*logrus.Entry"), mock.Anything, mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				if err != nil {
					t.Errorf("Unexpected error occurred when executing LightAction: %v", err)
				}
			},
		},
		{
			"LightDelayError",
			&LightAction{State: pkg.LightStateOff, ForDuration: "30s"},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, scheduler *MockScheduler) {
				mqttClient.On("LightTopic", "garden").Return("garden/action/light", nil)
				mqttClient.On("Publish", "garden/action/light", mock.Anything).Return(nil)
				scheduler.On("ScheduleLightDelay", mock.AnythingOfType("*logrus.Entry"), mock.Anything, mock.Anything).Return(errors.New("delay error"))
			},
			func(err error, t *testing.T) {
				if err == nil {
					t.Error("Expected error, but nil was returned")
				}
				if err.Error() != "unable to handle light delay: delay error" {
					t.Errorf("Unexpected error string: %v", err)
				}
			},
		},
		{
			"PublishError",
			&LightAction{State: pkg.LightStateOff, ForDuration: "30s"},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, scheduler *MockScheduler) {
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
			&LightAction{},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient, scheduler *MockScheduler) {
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
			mqttClient := new(mqtt.MockClient)
			influxdbClient := new(influxdb.MockClient)
			scheduler := new(MockScheduler)
			scheduler.On("MQTTClient").Return(mqttClient)
			tt.setupMock(mqttClient, influxdbClient, scheduler)

			err := tt.action.Execute(garden, nil, scheduler)
			tt.assert(err, t)
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
			scheduler.AssertExpectations(t)
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
		action    *StopAction
		setupMock func(*mqtt.MockClient, *influxdb.MockClient)
		assert    func(error, *testing.T)
	}{
		{
			"Successful",
			&StopAction{},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("StopTopic", "garden").Return("garden/action/stop", nil)
				mqttClient.On("Publish", "garden/action/stop", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				if err != nil {
					t.Errorf("Unexpected error occurred when executing StopAction: %v", err)
				}
			},
		},
		{
			"SuccessfulStopAll",
			&StopAction{true},
			func(mqttClient *mqtt.MockClient, influxdbClient *influxdb.MockClient) {
				mqttClient.On("StopAllTopic", "garden").Return("garden/action/stop_all", nil)
				mqttClient.On("Publish", "garden/action/stop_all", mock.Anything).Return(nil)
			},
			func(err error, t *testing.T) {
				if err != nil {
					t.Errorf("Unexpected error occurred when executing StopAction: %v", err)
				}
			},
		},
		{
			"TopicTemplateError",
			&StopAction{},
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

			err := tt.action.Execute(garden, nil, NewScheduler(nil, influxdbClient, mqttClient))
			tt.assert(err, t)
			mqttClient.AssertExpectations(t)
			influxdbClient.AssertExpectations(t)
		})
	}
}
