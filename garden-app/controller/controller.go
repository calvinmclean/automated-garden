package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"sync"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
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
	GardenName     string          `mapstructure:"garden_name"`
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

	logger.Infof("starting controller '%s'\n", controller.GardenName)

	topics, err := topics(controller.MQTTConfig, controller.GardenName)
	if err != nil {
		logger.Errorf("unable to determine topics: %v", err)
		return
	}

	wg := sync.WaitGroup{}
	for i, topic := range topics {
		wg.Add(1)
		logger.Infof("subscribing on topic: %s", topic)

		// Override configured ClientID with the GardenName from command flags
		controller.MQTTConfig.ClientID = fmt.Sprintf("%s-%d", controller.GardenName, i)
		mqttClient, err := mqtt.NewMQTTClient(controller.MQTTConfig, getHandlerForTopic(topic))
		if err != nil {
			logger.Errorf("unable to initialize MQTT client: %v", err)
			return
		}
		go mqttClient.Subscribe(topic, func() {
			for true {
			}
			wg.Done()
		})
	}
	wg.Wait()
}

// getHandlerForTopic provides a different MessageHandler function for each of the expected
// topics to be able to handle them in different ways
func getHandlerForTopic(topic string) paho.MessageHandler {
	switch t := strings.Split(topic, "/")[2]; t {
	case "water":
		return paho.MessageHandler(func(c paho.Client, msg paho.Message) {
			var waterMsg pkg.WaterMessage
			err := json.Unmarshal(msg.Payload(), &waterMsg)
			if err != nil {
				logger.Errorf("unable to unmarshal WaterMessage JSON: %s", err.Error())
			}
			logger.WithFields(logrus.Fields{
				"topic":          msg.Topic(),
				"plant_id":       waterMsg.PlantID,
				"plant_position": waterMsg.PlantPosition,
				"duration":       waterMsg.Duration,
			}).Info("received WaterAction")
		})
	case "stop":
		return paho.MessageHandler(func(c paho.Client, msg paho.Message) {
			logger.WithFields(logrus.Fields{
				"topic": msg.Topic(),
			}).Info("received StopAction")
		})
	case "stop_all":
		return paho.MessageHandler(func(c paho.Client, msg paho.Message) {
			logger.WithFields(logrus.Fields{
				"topic": msg.Topic(),
			}).Info("received StopAllAction")
		})
	default:
		return paho.MessageHandler(func(c paho.Client, msg paho.Message) {
			logger.WithFields(logrus.Fields{
				"topic": msg.Topic(),
			}).Infof("received message on unexpected topic: %s", string(msg.Payload()))
		})
	}
}

// topics returns a list of topics based on the Config values and provided GardenName
func topics(config mqtt.Config, gardenName string) ([]string, error) {
	templateData := map[string]string{"Garden": gardenName}
	topics := []string{}
	templates := []string{config.WateringTopicTemplate, config.StopTopicTemplate, config.StopAllTopicTemplate}
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
