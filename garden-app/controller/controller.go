package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"sync"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// Config holds all the options and sub-configs for the mock controller
type Config struct {
	InfluxDBConfig       influxdb.Config `mapstructure:"influxdb"`
	MQTTConfig           mqtt.Config     `mapstructure:"mqtt"`
	Garden               string          `mapstructure:"garden_name"`
	Plants               []int           `mapstructure:"plants"`
	PublishWateringEvent bool            `mapstructure:"publish_watering_event"`
	PublishHealth        bool            `mapstructure:"publish_health"`
	LogLevel             logrus.Level
}

// Controller struct holds the necessary data for running the mock garden-controller
type Controller struct {
	Config
	mqttClient mqtt.Client
}

// Start runs the main code of the mock garden-controller by creating MQTT clients and
// subscribing to each topic
func Start(config Config) {
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	logger.SetLevel(config.LogLevel)

	controller := Controller{Config: config}

	logger.Infof("starting controller '%s'\n", controller.Garden)

	if len(config.Plants) > 0 {
		logger.Infof("publishing moisture data for Plants: %v", config.Plants)
	}

	topics, err := controller.topics()
	if err != nil {
		logger.Errorf("unable to determine topics: %v", err)
		return
	}

	// Build TopicHandlers to handle subscription to each topic
	var handlers []mqtt.TopicHandler
	for _, topic := range topics {
		logger.Infof("subscribing on topic: %s", topic)
		handlers = append(handlers, mqtt.TopicHandler{
			Topic:   topic,
			Handler: controller.getHandlerForTopic(topic),
		})
	}

	// Create default handler and mqttClient, then connect
	defaultHandler := paho.MessageHandler(func(c paho.Client, msg paho.Message) {
		logger.WithFields(logrus.Fields{
			"topic": msg.Topic(),
		}).Infof("default handler called with message: %s", string(msg.Payload()))
	})
	// Override configured ClientID with the GardenName from command flags
	controller.MQTTConfig.ClientID = fmt.Sprintf(controller.Garden)
	controller.mqttClient, err = mqtt.NewMQTTClient(controller.MQTTConfig, defaultHandler, handlers...)
	if err != nil {
		logger.Errorf("unable to initialize MQTT client: %v", err)
		return
	}
	if err := controller.mqttClient.Connect(); err != nil {
		logger.Errorf("unable to connect to MQTT broker: %v", err.Error())
	}

	// Initiate publishing goroutines and wait
	wg := sync.WaitGroup{}
	wg.Add(1)
	for _, plant := range config.Plants {
		wg.Add(1)
		go controller.publishMoistureData(plant)
	}
	if controller.PublishHealth {
		go controller.publishHealthInfo()
	}
	wg.Wait()
}

func (c *Controller) publishMoistureData(plant int) {
	for {
		moisture := 50
		topic := fmt.Sprintf("%s/data/moisture", c.Garden)
		logger.Infof("publishing moisture data for Plant %d on topic %s: %.2f", plant, topic, moisture)
		err := c.mqttClient.Publish(
			topic,
			[]byte(fmt.Sprintf("moisture,plant=%d value=%d", plant, moisture)),
		)
		if err != nil {
			logger.Errorf("encountered error publishing: %v", err)
		}

		time.Sleep(5 * time.Second)
	}
}

// publishWateringEvent logs moisture data to InfluxDB via Telegraf and MQTT
func (c *Controller) publishWateringEvent(waterMsg pkg.WaterMessage, cmdTopic string) {
	if !c.PublishWateringEvent {
		return
	}
	// Incoming topic is "{{.GardenName}}/command/water" but we need to publish on "{{.GardenName}}/data/water"
	dataTopic := strings.ReplaceAll(cmdTopic, "command", "data")
	logger.Infof("publishing watering event for Plant on topic %s: %v", dataTopic, waterMsg)
	err := c.mqttClient.Publish(
		dataTopic,
		[]byte(fmt.Sprintf("water,plant=%d millis=%d", waterMsg.PlantPosition, waterMsg.Duration)),
	)
	if err != nil {
		logger.Errorf("encountered error publishing: %v", err)
	}
}

// getHandlerForTopic provides a different MessageHandler function for each of the expected
// topics to be able to handle them in different ways
func (c *Controller) getHandlerForTopic(topic string) paho.MessageHandler {
	switch t := strings.Split(topic, "/")[2]; t {
	case "water":
		return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
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
			c.publishWateringEvent(waterMsg, topic)
		})
	case "stop":
		return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
			logger.WithFields(logrus.Fields{
				"topic": msg.Topic(),
			}).Info("received StopAction")
		})
	case "stop_all":
		return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
			logger.WithFields(logrus.Fields{
				"topic": msg.Topic(),
			}).Info("received StopAllAction")
		})
	default:
		return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
			logger.WithFields(logrus.Fields{
				"topic": msg.Topic(),
			}).Infof("received message on unexpected topic: %s", string(msg.Payload()))
		})
	}
}

// publishHealthInfo publishes an InfluxDB line every minute to record that the controller is alive and active
func (c *Controller) publishHealthInfo() {
	topic := fmt.Sprintf("%s/data/health", c.Garden)
	for {
		logger.Infof("publishing health data on topic %s", topic)
		err := c.mqttClient.Publish(topic, []byte(fmt.Sprintf("health garden=\"%s\"", c.Garden)))
		if err != nil {
			logger.Errorf("encountered error publishing: %v", err)
		}
		time.Sleep(1 * time.Minute)
	}
}

// topics returns a list of topics based on the Config values and provided GardenName
func (c *Controller) topics() ([]string, error) {
	topics := []string{}
	templates := []string{
		c.Config.MQTTConfig.WateringTopicTemplate,
		c.Config.MQTTConfig.StopTopicTemplate,
		c.Config.MQTTConfig.StopAllTopicTemplate,
	}
	for _, topicTemplate := range templates {
		t := template.Must(template.New("topic").Parse(topicTemplate))
		var topic bytes.Buffer
		err := t.Execute(&topic, c.Config)
		if err != nil {
			return topics, err
		}
		topics = append(topics, topic.String())
	}
	return topics, nil
}
