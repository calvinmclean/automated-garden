package pkg

import (
	"fmt"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/rs/xid"
	"github.com/stretchr/testify/mock"
)

type MockMQTTClient struct {
	mock.Mock
}

func (m *MockMQTTClient) Publish(topic string, message []byte) error {
	args := m.Called(topic, message)
	return args.Error(0)
}
func (m *MockMQTTClient) Subscribe(topic string, blockingHandler func()) error {
	args := m.Called(topic, blockingHandler)
	return args.Error(0)
}

func (m *MockMQTTClient) WateringTopic(gardenName string) (string, error) {
	args := m.Called(gardenName)
	return args.String(0), args.Error(1)
}

func (m *MockMQTTClient) StopTopic(gardenName string) (string, error) {
	args := m.Called(gardenName)
	return args.String(0), args.Error(1)
}

func (m *MockMQTTClient) StopAllTopic(gardenName string) (string, error) {
	args := m.Called(gardenName)
	return args.String(0), args.Error(1)
}

func TestWaterActionExecute(t *testing.T) {
	garden := &Garden{
		Name: "garden",
	}
	action := &WaterAction{
		Duration: 1000,
	}
	influxdbConfig := influxdb.Config{}
	t.Run("Successful", func(t *testing.T) {
		mqttClient := new(MockMQTTClient)
		mqttClient.On("WateringTopic", "garden").Return("garden/action/water", nil)
		mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)

		err := action.Execute(garden, &Plant{}, mqttClient, influxdbConfig)
		if err != nil {
			t.Errorf("Error occurred when executing WaterAction: %v", err)
		}
	})
	t.Run("TopicTemplateError", func(t *testing.T) {
		mqttClient := new(MockMQTTClient)
		mqttClient.On("WateringTopic", "garden").Return("", fmt.Errorf("template error"))
		mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)

		err := action.Execute(garden, &Plant{}, mqttClient, influxdbConfig)
		if err == nil {
			t.Error("Expected error, but nil was returned")
		}
		if err.Error() != "unable to fill MQTT topic template: template error" {
			t.Errorf("Unexpected error string: %v", err)
		}
	})
	t.Run("ErrorWhenSkipGreaterThanZero", func(t *testing.T) {
		plant := &Plant{
			SkipCount: 1,
		}
		mqttClient := new(MockMQTTClient)
		mqttClient.On("WateringTopic", "garden").Return("garden/action/water", nil)
		mqttClient.On("Publish", "garden/action/water", mock.Anything).Return(nil)

		err := action.Execute(garden, plant, mqttClient, influxdbConfig)
		if err == nil {
			t.Error("Expected error, but nil was returned")
		}
		if err.Error() != fmt.Sprintf("plant %s is configured to skip watering", xid.NilID()) {
			t.Errorf("Unexpected error string: %v", err)
		}
		if plant.SkipCount != 0 {
			t.Errorf("Plant.SkipCount expected to be 0 after watering, but was %d", plant.SkipCount)
		}
	})
	// t.Run("IgnoreMoistureIsFalse", func(t *&testing.T) {

	// })
}
