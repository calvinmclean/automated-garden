package controller

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-co-op/gocron"
	"github.com/rivo/tview"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

var pubLogger *logrus.Logger
var subLogger *logrus.Logger

// Config holds all the options and sub-configs for the mock controller
type Config struct {
	MQTTConfig    mqtt.Config `mapstructure:"mqtt"`
	ControllerCfg `mapstructure:"controller"`
	LogLevel      logrus.Level
}

// ControllerCfg is an unfortunate struct that I had to create to have this nested under the 'controller' key
// in the YAML config
type ControllerCfg struct {
	TopicPrefix       string        `mapstructure:"topic_prefix"`
	NumZones          int           `mapstructure:"num_zones"`
	MoistureStrategy  string        `mapstructure:"moisture_strategy"`
	MoistureValue     int           `mapstructure:"moisture_value"`
	MoistureInterval  time.Duration `mapstructure:"moisture_interval"`
	PublishWaterEvent bool          `mapstructure:"publish_watering_event"`
	PublishHealth     bool          `mapstructure:"publish_health"`
	HealthInterval    time.Duration `mapstructure:"health_interval"`
	EnableUI          bool          `mapstructure:"enable_ui"`

	// Configs only used for generate-config
	WifiConfig           `mapstructure:"wifi"`
	Zones                []ZoneConfig  `mapstructure:"zones"`
	DefaultWaterTime     time.Duration `mapstructure:"default_water_time"`
	EnableButtons        bool          `mapstructure:"enable_buttons"`
	EnableMoistureSensor bool          `mapstructure:"enable_moisture_sensor"`
	LightPin             string        `mapstructure:"light_pin"`
	StopButtonPin        string        `mapstructure:"stop_water_button"`
	DisableWatering      bool          `mapstructure:"disable_watering"`
}

// Controller struct holds the necessary data for running the mock garden-controller
type Controller struct {
	Config
	mqttClient mqtt.Client
}

// Start runs the main code of the mock garden-controller by creating MQTT clients and
// subscribing to each topic
func Start(config Config) {
	controller := Controller{Config: config}

	logger = controller.setupLogger()
	subLogger = controller.setupLogger()
	pubLogger = controller.setupLogger()

	var app *tview.Application
	if controller.EnableUI {
		app = controller.setupUI()
	}

	logger.Infof("starting controller '%s'\n", controller.TopicPrefix)

	if config.NumZones > 0 {
		pubLogger.Infof("publishing moisture data for %d Zones", config.NumZones)
	}

	topics, err := controller.topics()
	if err != nil {
		logger.Errorf("unable to determine topics: %v", err)
		return
	}

	// Build TopicHandlers to handle subscription to each topic
	var handlers []mqtt.TopicHandler
	for _, topic := range topics {
		subLogger.Infof("subscribing on topic: %s", topic)
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
	// Override configured ClientID with the TopicPrefix from command flags
	controller.MQTTConfig.ClientID = fmt.Sprintf(controller.TopicPrefix)
	controller.mqttClient, err = mqtt.NewClient(controller.MQTTConfig, defaultHandler, handlers...)
	if err != nil {
		logger.Errorf("unable to initialize MQTT client: %v", err)
		return
	}
	if err := controller.mqttClient.Connect(); err != nil {
		logger.Errorf("unable to connect to MQTT broker: %v", err.Error())
	}

	// Initialize scheduler and schedule publishing Jobs
	scheduler := gocron.NewScheduler(time.Local)
	for p := 0; p < controller.NumZones; p++ {
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

	if controller.EnableUI {
		if err := app.Run(); err != nil {
			panic(err)
		}
	} else {
		wg.Wait()
	}
	logger.Infof("controller shutdown gracefully in %v", time.Since(shutdownStart))
}

// setupLogger creates and configures a logger with colors and specified log level
func (c *Controller) setupLogger() *logrus.Logger {
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		ForceColors:   true,
		FullTimestamp: true,
	})
	l.SetLevel(c.LogLevel)
	return l
}

// setupUI configures the two-column view for publish and subscribe logs
func (c *Controller) setupUI() *tview.Application {
	app := tview.NewApplication()

	left := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetText("Subscribe Logs").
		SetDynamicColors(true).
		SetChangedFunc(func() { app.Draw() })
	right := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetText("Publish Logs").
		SetDynamicColors(true).
		SetChangedFunc(func() { app.Draw() })

	subLogger.SetOutput(tview.ANSIWriter(left))
	pubLogger.SetOutput(tview.ANSIWriter(right))

	header := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(c.TopicPrefix)
	tview.ANSIWriter(header).Write([]byte(fmt.Sprintf(
		"\n%d Zones\nPublishWaterEvent: %t, PublishHealth: %t, MoistureStrategy: %s",
		c.NumZones, c.PublishWaterEvent, c.PublishHealth, c.MoistureStrategy),
	))

	grid := tview.NewGrid().
		SetRows(3, 0).
		SetBorders(true)

	grid.
		AddItem(header, 0, 0, 1, 2, 0, 0, false).
		AddItem(left, 1, 0, 1, 1, 0, 100, false).
		AddItem(right, 1, 1, 1, 1, 0, 100, false)

	tview.ANSIWriter(left).Write([]byte("\n"))
	tview.ANSIWriter(right).Write([]byte("\n"))

	return app.SetRoot(grid, true)
}

// publishMoistureData publishes an InfluxDB line containing moisture data for a Zone
func (c *Controller) publishMoistureData(zone int) {
	moisture := c.createMoistureData()
	topic := fmt.Sprintf("%s/data/moisture", c.TopicPrefix)
	pubLogger.Infof("publishing moisture data for Zone %d on topic %s: %d", zone, topic, moisture)
	err := c.mqttClient.Publish(
		topic,
		[]byte(fmt.Sprintf("moisture,zone=%d value=%d", zone, moisture)),
	)
	if err != nil {
		pubLogger.Errorf("encountered error publishing: %v", err)
	}
}

// publishHealthData publishes an InfluxDB line to record that the controller is alive and active
func (c *Controller) publishHealthData() {
	topic := fmt.Sprintf("%s/data/health", c.TopicPrefix)
	pubLogger.Infof("publishing health data on topic %s", topic)
	err := c.mqttClient.Publish(topic, []byte(fmt.Sprintf("health garden=\"%s\"", c.TopicPrefix)))
	if err != nil {
		pubLogger.Errorf("encountered error publishing: %v", err)
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

// publishWaterEvent logs moisture data to InfluxDB via Telegraf and MQTT
func (c *Controller) publishWaterEvent(waterMsg action.WaterMessage, cmdTopic string) {
	if !c.PublishWaterEvent {
		return
	}
	// Incoming topic is "{{.TopicPrefix}}/command/water" but we need to publish on "{{.TopicPrefix}}/data/water"
	dataTopic := strings.ReplaceAll(cmdTopic, "command", "data")
	pubLogger.Infof("publishing watering event for Zone on topic %s: %v", dataTopic, waterMsg)
	err := c.mqttClient.Publish(
		dataTopic,
		[]byte(fmt.Sprintf("water,zone=%d millis=%d", waterMsg.Position, waterMsg.Duration)),
	)
	if err != nil {
		pubLogger.Errorf("encountered error publishing: %v", err)
	}
}

// getHandlerForTopic provides a different MessageHandler function for each of the expected
// topics to be able to handle them in different ways
func (c *Controller) getHandlerForTopic(topic string) paho.MessageHandler {
	switch t := strings.Split(topic, "/")[2]; t {
	case "water":
		return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
			var waterMsg action.WaterMessage
			err := json.Unmarshal(msg.Payload(), &waterMsg)
			if err != nil {
				subLogger.Errorf("unable to unmarshal WaterMessage JSON: %s", err.Error())
			}
			subLogger.WithFields(logrus.Fields{
				"topic":    msg.Topic(),
				"zone_id":  waterMsg.ZoneID,
				"position": waterMsg.Position,
				"duration": waterMsg.Duration,
			}).Info("received WaterAction")
			c.publishWaterEvent(waterMsg, topic)
		})
	case "stop":
		return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
			subLogger.WithFields(logrus.Fields{
				"topic": msg.Topic(),
			}).Info("received StopAction")
		})
	case "stop_all":
		return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
			subLogger.WithFields(logrus.Fields{
				"topic": msg.Topic(),
			}).Info("received StopAllAction")
		})
	case "light":
		return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
			var action action.LightAction
			err := json.Unmarshal(msg.Payload(), &action)
			if err != nil {
				subLogger.Errorf("unable to unmarshal LightAction JSON: %s", err.Error())
			}
			subLogger.WithFields(logrus.Fields{
				"topic": msg.Topic(),
				"state": action.State,
			}).Info("received LightAction")
		})
	default:
		return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
			subLogger.WithFields(logrus.Fields{
				"topic": msg.Topic(),
			}).Infof("received message on unexpected topic: %s", string(msg.Payload()))
		})
	}
}

// topics returns a list of topics based on the Config values and provided TopicPrefix
func (c *Controller) topics() ([]string, error) {
	topics := []string{}
	templateFuncs := []func(string) (string, error){
		c.MQTTConfig.WaterTopic,
		c.MQTTConfig.StopTopic,
		c.MQTTConfig.StopAllTopic,
		c.MQTTConfig.LightTopic,
	}
	for _, templateFunc := range templateFuncs {
		topic, err := templateFunc(c.TopicPrefix)
		if err != nil {
			return topics, err
		}
		topics = append(topics, topic)
	}
	return topics, nil
}
