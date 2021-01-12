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
}

// Client is a wrapper struct for connecting our config and MQTT Client
type Client struct {
	mqtt.Client
	Config Config
}

// NewMQTTClient is used to create and return a MQTTClient
func NewMQTTClient() Client {
	config := Config{
		ClientID: "garden-app",
		Broker:   "localhost",
		Port:     1883,
	}
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.Broker, config.Port))
	opts.SetClientID(config.ClientID)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	return Client{mqtt.NewClient(opts), config}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("MQTT connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("MQTT connection lost: %v", err)
}
