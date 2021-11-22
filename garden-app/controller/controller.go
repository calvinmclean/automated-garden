package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-co-op/gocron"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// Config holds all the options and sub-configs for the mock controller
type Config struct {
	MQTTConfig           mqtt.Config   `mapstructure:"mqtt"`
	Garden               string        `mapstructure:"garden_name"`
	NumPlants            int           `mapstructure:"num_plants"`
	MoistureStrategy     string        `mapstructure:"moisture_strategy"`
	MoistureValue        int           `mapstructure:"moisture_value"`
	MoistureInterval     time.Duration `mapstructure:"moisture_interval"`
	PublishWateringEvent bool          `mapstructure:"publish_watering_event"`
	PublishHealth        bool          `mapstructure:"publish_health"`
	HealthInterval       time.Duration `mapstructure:"health_interval"`
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

	if config.NumPlants > 0 {
		logger.Infof("publishing moisture data for %d Plants", config.NumPlants)
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

	// Initialize scheduler and schedule publishing Jobs
	scheduler := gocron.NewScheduler(time.Local)
	for p := 0; p < controller.NumPlants; p++ {
		scheduler.Every(controller.MoistureInterval).Do(controller.publishMoistureData, p)
	}
	if controller.PublishHealth {
		scheduler.Every(controller.HealthInterval).Do(controller.publishHealthData)
	}
	scheduler.StartAsync()

	// Shutdown gracefully on Ctrl+C
	wg := &sync.WaitGroup{}
	wg.Add(1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	var shutdownStart time.Time
	go func() {
		<-quit
		shutdownStart = time.Now()
		logger.Info("gracefully shutting down controller")

		scheduler.Stop()

		// Disconnect mqttClient
		logger.Info("disconnecting MQTT Client")
		controller.mqttClient.Disconnect(1000)
		wg.Done()
	}()
	wg.Wait()
	logger.Infof("controller shutdown gracefully in %v", time.Since(shutdownStart))
}

// publishMoistureData publishes an InfluxDB line containing moisture data for a Plant
func (c *Controller) publishMoistureData(plant int) {
	moisture := c.createMoistureData()
	topic := fmt.Sprintf("%s/data/moisture", c.Garden)
	logger.Infof("publishing moisture data for Plant %d on topic %s: %d", plant, topic, moisture)
	err := c.mqttClient.Publish(
		topic,
		[]byte(fmt.Sprintf("moisture,plant=%d value=%d", plant, moisture)),
	)
	if err != nil {
		logger.Errorf("encountered error publishing: %v", err)
	}
}

// publishHealthData publishes an InfluxDB line to record that the controller is alive and active
func (c *Controller) publishHealthData() {
	topic := fmt.Sprintf("%s/data/health", c.Garden)
	logger.Infof("publishing health data on topic %s", topic)
	err := c.mqttClient.Publish(topic, []byte(fmt.Sprintf("health garden=\"%s\"", c.Garden)))
	if err != nil {
		logger.Errorf("encountered error publishing: %v", err)
	}
}

// createMoistureData uses the MoistureStrategy config to create a moisture data point
func (c *Controller) createMoistureData() int {
	switch c.MoistureStrategy {
	case "random":
		source := rand.New(rand.NewSource(time.Now().UnixNano()))
		return source.Intn(c.MoistureValue)
	case "constant":
		return c.MoistureValue
	case "increasing":
		c.MoistureValue++
		if c.MoistureValue > 100 {
			c.MoistureValue = 0
		}
		return c.MoistureValue
	case "decreasing":
		c.MoistureValue--
		if c.MoistureValue < 0 {
			c.MoistureValue = 100
		}
		return c.MoistureValue
	default:
		return 0
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
	case "light":
		return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
			var action pkg.LightAction
			err := json.Unmarshal(msg.Payload(), &action)
			if err != nil {
				logger.Errorf("unable to unmarshal LightAction JSON: %s", err.Error())
			}
			logger.WithFields(logrus.Fields{
				"topic": msg.Topic(),
				"state": action.State,
			}).Info("received LightAction")
		})
	default:
		return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
			logger.WithFields(logrus.Fields{
				"topic": msg.Topic(),
			}).Infof("received message on unexpected topic: %s", string(msg.Payload()))
		})
	}
}

// topics returns a list of topics based on the Config values and provided GardenName
func (c *Controller) topics() ([]string, error) {
	topics := []string{}
	templates := []string{
		c.Config.MQTTConfig.WateringTopicTemplate,
		c.Config.MQTTConfig.StopTopicTemplate,
		c.Config.MQTTConfig.StopAllTopicTemplate,
		c.Config.MQTTConfig.LightTopicTemplate,
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
