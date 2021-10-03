package pkg

import (
	"errors"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/stretchr/testify/mock"
)

func TestAggregateAction(t *testing.T) {
	garden := &Garden{
		Name: "garden",
	}
	t.Run("SuccessfulEmptyAggregateAction", func(t *testing.T) {
		plant := &Plant{}
		action := &AggregateAction{}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err != nil {
			t.Errorf("Unexpected error occurred when executing AggregateAction: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
	t.Run("SuccessfulAggregateActionWithStopAction", func(t *testing.T) {
		plant := &Plant{}
		action := &AggregateAction{
			Stop: &StopAction{},
		}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)
		mqttClient.On("StopTopic", "garden").Return("garden/action/stop", nil)
		mqttClient.On("Publish", "garden/action/stop", mock.Anything).Return(nil)

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err != nil {
			t.Errorf("Unexpected error occurred when executing AggregateAction: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
	t.Run("FailedAggregateActionWithStopAction", func(t *testing.T) {
		plant := &Plant{}
		action := &AggregateAction{
			Stop: &StopAction{},
		}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)
		mqttClient.On("StopTopic", "garden").Return("", errors.New("template error"))

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err == nil {
			t.Error("Expected error, but nil was returned")
		}
		if err.Error() != "unable to fill MQTT topic template: template error" {
			t.Errorf("Unexpected error string: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
	t.Run("SuccessfulAggregateActionWithWaterAction", func(t *testing.T) {
		plant := &Plant{}
		action := &AggregateAction{
			Water: &WaterAction{
				Duration: 1000,
			},
		}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)
		mqttClient.On("WateringTopic", "garden").Return("garden/action/water", nil)
		mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err != nil {
			t.Errorf("Unexpected error occurred when executing AggregateAction: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
	t.Run("FailedAggregateActionWithWaterAction", func(t *testing.T) {
		plant := &Plant{}
		action := &AggregateAction{
			Water: &WaterAction{
				Duration: 1000,
			},
		}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)
		mqttClient.On("WateringTopic", "garden").Return("", errors.New("template error"))

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err == nil {
			t.Error("Expected error, but nil was returned")
		}
		if err.Error() != "unable to fill MQTT topic template: template error" {
			t.Errorf("Unexpected error string: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
}

func TestStopActionExecute(t *testing.T) {
	garden := &Garden{
		Name: "garden",
	}
	t.Run("Successful", func(t *testing.T) {
		plant := &Plant{}
		action := &StopAction{}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)
		mqttClient.On("StopTopic", "garden").Return("garden/action/stop", nil)
		mqttClient.On("Publish", "garden/action/stop", mock.Anything).Return(nil)

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err != nil {
			t.Errorf("Unexpected error occurred when executing StopAction: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
	t.Run("SuccessfulStopAll", func(t *testing.T) {
		action := &StopAction{true}
		plant := &Plant{}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)
		mqttClient.On("StopAllTopic", "garden").Return("garden/action/stop_all", nil)
		mqttClient.On("Publish", "garden/action/stop_all", mock.Anything).Return(nil)

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err != nil {
			t.Errorf("Unexpected error occurred when executing StopAction: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
	t.Run("TopicTemplateError", func(t *testing.T) {
		plant := &Plant{}
		action := &StopAction{}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)
		mqttClient.On("StopTopic", "garden").Return("", errors.New("template error"))

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err == nil {
			t.Error("Expected error, but nil was returned")
		}
		if err.Error() != "unable to fill MQTT topic template: template error" {
			t.Errorf("Unexpected error string: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
}

func TestWaterActionExecute(t *testing.T) {
	garden := &Garden{
		Name: "garden",
	}
	action := &WaterAction{
		Duration: 1000,
	}
	t.Run("Successful", func(t *testing.T) {
		plant := &Plant{}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)
		mqttClient.On("WateringTopic", "garden").Return("garden/action/water", nil)
		mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err != nil {
			t.Errorf("Unexpected error occurred when executing WaterAction: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
	t.Run("TopicTemplateError", func(t *testing.T) {
		plant := &Plant{}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)
		mqttClient.On("WateringTopic", "garden").Return("", errors.New("template error"))

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err == nil {
			t.Error("Expected error, but nil was returned")
		}
		if err.Error() != "unable to fill MQTT topic template: template error" {
			t.Errorf("Unexpected error string: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
	t.Run("ErrorWhenSkipGreaterThanZero", func(t *testing.T) {
		plant := &Plant{
			SkipCount: 1,
		}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err == nil {
			t.Error("Expected error, but nil was returned")
		}
		if err.Error() != "plant 00000000000000000000 is configured to skip watering" {
			t.Errorf("Unexpected error string: %v", err)
		}
		if plant.SkipCount != 0 {
			t.Errorf("Plant.SkipCount expected to be 0 after watering, but was %d", plant.SkipCount)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
	t.Run("SuccessWhenMoistureLessThanMinimum", func(t *testing.T) {
		plant := &Plant{
			WateringStrategy: WateringStrategy{
				MinimumMoisture: 50,
			},
		}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)
		mqttClient.On("WateringTopic", "garden").Return("garden/action/water", nil)
		mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)
		influxdbClient.On("GetMoisture", mock.Anything, 0, garden.Name).Return(float64(0), nil)

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err != nil {
			t.Errorf("Unexpected error occurred when executing WaterAction: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
	t.Run("ErrorWhenMoistureGreaterThanMinimum", func(t *testing.T) {
		plant := &Plant{
			WateringStrategy: WateringStrategy{
				MinimumMoisture: 50,
			},
		}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)
		influxdbClient.On("GetMoisture", mock.Anything, 0, garden.Name).Return(float64(51), nil)

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err == nil {
			t.Error("Expected error, but nil was returned")
		}
		if err.Error() != "moisture value 51.00% is above threshold 50%" {
			t.Errorf("Unexpected error string: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
	t.Run("ErrorInfluxDBClient", func(t *testing.T) {
		plant := &Plant{
			WateringStrategy: WateringStrategy{
				MinimumMoisture: 50,
			},
		}
		mqttClient := new(mqtt.MockClient)
		influxdbClient := new(influxdb.MockClient)
		influxdbClient.On("GetMoisture", mock.Anything, 0, garden.Name).Return(float64(0), errors.New("influxdb error"))

		err := action.Execute(garden, plant, mqttClient, influxdbClient)
		if err == nil {
			t.Error("Expected error, but nil was returned")
		}
		if err.Error() != "error getting Plant's moisture data: influxdb error" {
			t.Errorf("Unexpected error string: %v", err)
		}
		mqttClient.AssertExpectations(t)
		influxdbClient.AssertExpectations(t)
	})
}
