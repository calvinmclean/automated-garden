package mqtt

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Config is used to read the necessary configuration values from a YAML file
type Config struct {
	ClientID string `yaml:"client_id"`
	Broker   string `yaml:"broker"`
	Port     int    `yaml:"port"`

	WateringTopic string `yaml:"watering_topic"`
	SkipTopic     string `yaml:"skip_topic"`
	StopTopic     string `yaml:"stop_topic"`
	StopAllTopic  string `yaml:"stop_all_topic"`
}

// Client is a wrapper struct for connecting our config and MQTT Client
type Client struct {
	mqtt.Client
	Config
}

// NewMQTTClient is used to create and return a MQTTClient
func NewMQTTClient(config Config) (Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.Broker, config.Port))
	opts.SetClientID(config.ClientID)
	return Client{mqtt.NewClient(opts), config}, nil
}

func (client Client) Publish(topic string, message []byte) error {
	if token := client.Client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("unable to connect to MQTT broker: %v", token.Error())
	}
	if token := client.Client.Publish(topic, 0, false, message); token.Wait() && token.Error() != nil {
		return fmt.Errorf("unable to publish MQTT message: %v", token.Error())
	}
	return nil
}
