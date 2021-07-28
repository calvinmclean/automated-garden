package mqtt

import (
	"fmt"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Config is used to read the necessary configuration values from a YAML file
type Config struct {
	ClientID string `mapstructure:"client_id"`
	Broker   string `mapstructure:"broker"`
	Port     int    `mapstructure:"port"`

	WateringTopic string `mapstructure:"watering_topic"`
	SkipTopic     string `mapstructure:"skip_topic"`
	StopTopic     string `mapstructure:"stop_topic"`
	StopAllTopic  string `mapstructure:"stop_all_topic"`
}

// Client is a wrapper struct for connecting our config and MQTT Client
type Client struct {
	mu sync.Mutex
	mqtt.Client
	Config
}

// NewMQTTClient is used to create and return a MQTTClient
func NewMQTTClient(config Config) (*Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.Broker, config.Port))
	opts.SetClientID(config.ClientID)
	return &Client{Client: mqtt.NewClient(opts), Config: config}, nil
}

// Publish will send the message to the specified MQTT topic
func (c *Client) Publish(topic string, message []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer c.Client.Disconnect(250)
	if token := c.Client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("unable to connect to MQTT broker: %v", token.Error())
	}
	if token := c.Client.Publish(topic, byte(0), false, message); token.Wait() && token.Error() != nil {
		return fmt.Errorf("unable to publish MQTT message: %v", token.Error())
	}
	return nil
}
