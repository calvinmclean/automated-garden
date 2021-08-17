package controller

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"sync"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// Config holds all the options and sub-configs for the mock controller
type Config struct {
	InfluxDBConfig influxdb.Config `mapstructure:"influxdb"`
	MQTTConfig     mqtt.Config     `mapstructure:"mqtt"`
	GardenName     string
}

// Controller struct holds the necessary data for running the mock garden-controller
type Controller struct {
	Config
}

// Start runs the main code of the mock garden-controller by creating MQTT clients and
// subscribing to each topic
func Start(config Config) {
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	controller := Controller{Config: config}

	logger.Infof("Starting controller '%s'\n", controller.GardenName)

	topics, err := topics(controller.MQTTConfig, controller.GardenName)
	if err != nil {
		logger.Errorf("unable to determine topics: %v", err)
		return
	}

	wg := sync.WaitGroup{}
	for i, topic := range topics {
		wg.Add(1)
		logger.Infof("Subscribing on topic: %s", topic)

		// Override configured ClientID with the GardenName from command flags
		controller.MQTTConfig.ClientID = fmt.Sprintf("%s-%d", controller.GardenName, i)
		mqttClient, err := mqtt.NewMQTTClient(controller.MQTTConfig, getHandlerForTopic(topic))
		if err != nil {
			logger.Errorf("unable to initialize MQTT client: %v", err)
			return
		}
		go mqttClient.Subscribe(topic)
	}
	wg.Wait()
}

// getHandlerForTopic provides a different MessageHandler function for each of the expected
// topics to be able to handle them in different ways
func getHandlerForTopic(topic string) paho.MessageHandler {
	switch t := strings.Split(topic, "/")[2]; t {
	case "water":
		return paho.MessageHandler(func(c paho.Client, msg paho.Message) {
			fmt.Printf("WATER: %s MESSAGE: %s\n", msg.Topic(), string(msg.Payload()))
		})
	case "stop":
		return paho.MessageHandler(func(c paho.Client, msg paho.Message) {
			fmt.Printf("STOP: %s MESSAGE: %s\n", msg.Topic(), string(msg.Payload()))
		})
	case "stop_all":
		return paho.MessageHandler(func(c paho.Client, msg paho.Message) {
			fmt.Printf("STOP ALL: %s MESSAGE: %s\n", msg.Topic(), string(msg.Payload()))
		})
	default:
		return paho.MessageHandler(func(c paho.Client, msg paho.Message) {
			fmt.Printf("DEFAULT: %s MESSAGE: %s\n", msg.Topic(), string(msg.Payload()))
		})
	}
}

// topics returns a list of topics based on the Config values and provided GardenName
func topics(config mqtt.Config, gardenName string) ([]string, error) {
	templateData := map[string]string{"Garden": gardenName}
	topics := []string{}
	templates := []string{config.WateringTopic, config.StopTopic, config.StopAllTopic}
	for _, topicTemplate := range templates {
		t := template.Must(template.New("topic").Parse(topicTemplate))
		var topic bytes.Buffer
		err := t.Execute(&topic, templateData)
		if err != nil {
			return topics, err
		}
		topics = append(topics, topic.String())
	}
	return topics, nil
}
