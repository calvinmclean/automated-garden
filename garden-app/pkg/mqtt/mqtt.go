package mqtt

//go:generate mockery --all --inpackage

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
)

const QOS = byte(1)

var mqttClientSummary = prometheus.NewSummaryVec(prometheus.SummaryOpts{
	Namespace: "garden_app",
	Name:      "mqtt_client_duration_seconds",
	Help:      "summary of MQTT client calls",
}, []string{"function", "topic"})

// Config is used to read the necessary configuration values from a YAML file
type Config struct {
	ClientID string `mapstructure:"client_id" yaml:"client_id"`
	Broker   string `mapstructure:"broker" yaml:"broker"`
	Port     int    `mapstructure:"port" yaml:"port"`
}

// Client is an interface that allows access to MQTT functionality within the garden-app
type Client interface {
	Publish(string, []byte) error
	Connect() error
	Disconnect(uint)
	AddHandler(TopicHandler)
}

// client is a wrapper struct for connecting our config and MQTT Client. It implements the Client interface
type client struct {
	mu sync.Mutex
	mqtt.Client

	handlers []TopicHandler

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
	client := &client{
		Config:   config,
		handlers: handlers,
	}

	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:%d", config.Broker, config.Port))
	opts.ClientID = config.ClientID
	opts.AutoReconnect = true
	opts.CleanSession = false
	opts.OnConnect = func(c mqtt.Client) {
		for _, handler := range client.handlers {
			token := c.Subscribe(handler.Topic, QOS, handler.Handler)
			if token.Wait() && token.Error() != nil {
				// TODO: can I return an error instead of panicking (recover maybe?)
				panic(token.Error())
			}
		}
	}
	opts.DefaultPublishHandler = defaultHandler

	err := prometheus.Register(mqttClientSummary)
	if err != nil && errors.Is(err, prometheus.AlreadyRegisteredError{}) {
		return nil, err
	}

	client.Client = mqtt.NewClient(opts)

	return client, nil
}

func (c *client) AddHandler(handler TopicHandler) {
	c.handlers = append(c.handlers, handler)
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

func DefaultHandler(logger *slog.Logger) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		logger.With(
			"topic", msg.Topic(),
			"message", string(msg.Payload()),
		).Info("default handler called with message")
	}
}
