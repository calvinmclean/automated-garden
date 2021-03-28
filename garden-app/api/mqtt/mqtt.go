package mqtt

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
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
	mqtt.Client
	Config
}

// NewMQTTClient is used to create and return a MQTTClient
func NewMQTTClient() (Client, error) {
	// Read MQTT info from config
	var c Config
	if err := viper.UnmarshalKey("mqtt", &c); err != nil {
		return Client{}, err
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", c.Broker, c.Port))
	opts.SetClientID(c.ClientID)
	return Client{mqtt.NewClient(opts), c}, nil
}
