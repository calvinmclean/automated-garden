package mqtt

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
)

var mqttClientSummary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Namespace: "garden_app",
	Name:      "mqtt_client_duration_seconds",
	Help:      "summary of MQTT client calls",
}, []string{"function", "topic"})

// Config is used to read the necessary configuration values from a YAML file
type Config struct {
	ClientID string `mapstructure:"client_id"`
	Broker   string `mapstructure:"broker"`
	Port     int    `mapstructure:"port"`

	WaterTopicTemplate   string `mapstructure:"water_topic"`
	StopTopicTemplate    string `mapstructure:"stop_topic"`
	StopAllTopicTemplate string `mapstructure:"stop_all_topic"`
	LightTopicTemplate   string `mapstructure:"light_topic"`
}

// Client is an interface that allows access to MQTT functionality within the garden-app
type Client interface {
	Publish(string, []byte) error
	WaterTopic(string) (string, error)
	StopTopic(string) (string, error)
	StopAllTopic(string) (string, error)
	LightTopic(string) (string, error)
	Connect() error
	Disconnect(uint)
}

// client is a wrapper struct for connecting our config and MQTT Client. It implements the Client interface
type client struct {
	mu sync.Mutex
	mqtt.Client
	Config
}

// TopicHandler is a struct that contains a topic string and MessageHandler for instructing the client how to handle topics
type TopicHandler struct {
	Topic   string
	Handler mqtt.MessageHandler
}

// NewClient is used to create and return a MQTTClient. The handlers argument enables the subscriber
// using the supplied functions to handle incoming messages. It really should be used with only one function,
// but I wanted to make it an optional argument, which required using the variadic function argument
func NewClient(config Config, defaultHandler mqtt.MessageHandler, handlers ...TopicHandler) (Client, error) {
	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", config.Broker, config.Port))
	opts.ClientID = config.ClientID
	opts.AutoReconnect = true
	opts.CleanSession = false
	if len(handlers) > 0 {
		opts.OnConnect = func(c mqtt.Client) {
			for _, handler := range handlers {
				if token := c.Subscribe(handler.Topic, byte(1), handler.Handler); token.Wait() && token.Error() != nil {
					// TODO: can I return an error instead of panicking (recover maybe?)
					panic(token.Error())
				}
			}
		}
	}
	opts.DefaultPublishHandler = defaultHandler

	err := prometheus.Register(mqttClientSummary)
	if err != nil && errors.Is(err, prometheus.AlreadyRegisteredError{}) {
		return nil, err
	}

	return &client{Client: mqtt.NewClient(opts), Config: config}, nil
}

// Connect uses the MQTT Client's Connect function but returns the error instead of Token
func (c *client) Connect() error {
	timer := prometheus.NewTimer(mqttClientSummary.WithLabelValues("Connect", ""))
	defer timer.ObserveDuration()

	if c.Client.IsConnected() {
		return nil
	}
	token := c.Client.Connect()
	token.Wait()
	return token.Error()
}

// Publish will send the message to the specified MQTT topic
func (c *client) Publish(topic string, message []byte) error {
	timer := prometheus.NewTimer(mqttClientSummary.WithLabelValues("Publish", topic))
	defer timer.ObserveDuration()

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(topic) == 0 {
		return fmt.Errorf("unable to publish with an empty topic")
	}
	if err := c.Connect(); err != nil {
		return fmt.Errorf("unable to connect to MQTT broker: %v", err)
	}
	if token := c.Client.Publish(topic, byte(1), false, message); token.Wait() && token.Error() != nil {
		return fmt.Errorf("unable to publish MQTT message: %v", token.Error())
	}
	return nil
}

// WaterTopic returns the topic string for watering a zone
func (c *Config) WaterTopic(topicPrefix string) (string, error) {
	return c.executeTopicTemplate(c.WaterTopicTemplate, topicPrefix)
}

// StopTopic returns the topic string for stopping watering a single zone
func (c *Config) StopTopic(topicPrefix string) (string, error) {
	return c.executeTopicTemplate(c.StopTopicTemplate, topicPrefix)
}

// StopAllTopic returns the topic string for stopping watering all zones in a garden
func (c *Config) StopAllTopic(topicPrefix string) (string, error) {
	return c.executeTopicTemplate(c.StopAllTopicTemplate, topicPrefix)
}

// LightTopic returns the topic string for changing the light state in a Garden
func (c *Config) LightTopic(topicPrefix string) (string, error) {
	return c.executeTopicTemplate(c.LightTopicTemplate, topicPrefix)
}

// executeTopicTemplate is a helper function used by all the exported topic evaluation functions
func (c *Config) executeTopicTemplate(templateString string, topicPrefix string) (string, error) {
	t := template.Must(template.New("topic").Parse(templateString))
	var result bytes.Buffer
	data := map[string]string{"Garden": topicPrefix}
	err := t.Execute(&result, data)
	return result.String(), err
}
