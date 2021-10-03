package mqtt

import (
	"bytes"
	"fmt"
	"html/template"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Config is used to read the necessary configuration values from a YAML file
type Config struct {
	ClientID string `mapstructure:"client_id"`
	Broker   string `mapstructure:"broker"`
	Port     int    `mapstructure:"port"`

	WateringTopicTemplate string `mapstructure:"watering_topic"`
	StopTopicTemplate     string `mapstructure:"stop_topic"`
	StopAllTopicTemplate  string `mapstructure:"stop_all_topic"`
}

// Client is an interface that allows access to MQTT functionality within the garden-app
type Client interface {
	Publish(string, []byte) error
	Subscribe(string, func()) error
	WateringTopic(string) (string, error)
	StopTopic(string) (string, error)
	StopAllTopic(string) (string, error)
}

// client is a wrapper struct for connecting our config and MQTT Client. It implements the Client interface
type client struct {
	mu sync.Mutex
	mqtt.Client
	Config
}

// NewMQTTClient is used to create and return a MQTTClient. The handlers argument enables the subscriber
// using the supplied functions to handle incoming messages. It really should be used with only one function,
// but I wanted to make it an optional argument, which required using the variadic function argument
func NewMQTTClient(config Config, handlers ...mqtt.MessageHandler) (Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", config.Broker, config.Port))
	opts.SetClientID(config.ClientID)
	if len(handlers) > 0 {
		opts.SetDefaultPublishHandler(mqtt.MessageHandler(func(c mqtt.Client, msg mqtt.Message) {
			for _, handler := range handlers {
				handler(c, msg)
			}
		}))
	}
	return &client{Client: mqtt.NewClient(opts), Config: config}, nil
}

// Publish will send the message to the specified MQTT topic
func (c *client) Publish(topic string, message []byte) error {
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

// Subscribe is a blocking method that listens on a topic. This should be used in a Go routine with a blocking handler supplied
func (c *client) Subscribe(topic string, blockingHandler func()) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer c.Client.Disconnect(250)
	if token := c.Client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("unable to connect to MQTT broker: %v", token.Error())
	}
	if token := c.Client.Subscribe(topic, byte(0), nil); token.Wait() && token.Error() != nil {
		return fmt.Errorf("unable to subscribe to MQTT broker on topic '%s': %v", topic, token.Error())
	}

	blockingHandler()

	return nil
}

// WateringTopic returns the topic string for watering a plant
func (c *client) WateringTopic(gardenName string) (string, error) {
	return c.executeTopicTemplate(c.WateringTopicTemplate, gardenName)
}

// StopTopic returns the topic string for stopping watering a single plant
func (c *client) StopTopic(gardenName string) (string, error) {
	return c.executeTopicTemplate(c.StopTopicTemplate, gardenName)
}

// StopAllTopic returns the topic string for stopping watering all plants in a garden
func (c *client) StopAllTopic(gardenName string) (string, error) {
	return c.executeTopicTemplate(c.StopAllTopicTemplate, gardenName)
}

// executeTopicTemplate is a helper function used by all the exported topic evaluation functions
func (c *client) executeTopicTemplate(templateString string, gardenName string) (string, error) {
	t := template.Must(template.New("topic").Parse(templateString))
	var result bytes.Buffer
	data := map[string]string{"Garden": gardenName}
	err := t.Execute(&result, data)
	return result.String(), err
}
