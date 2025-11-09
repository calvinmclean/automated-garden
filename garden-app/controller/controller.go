package controller

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/server"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-co-op/gocron"
	"github.com/rivo/tview"
)

// Config holds all the options and sub-configs for the mock controller
type Config struct {
	MQTTConfig   mqtt.Config `mapstructure:"mqtt"`
	NestedConfig `mapstructure:"controller"`
	LogConfig    server.LogConfig `mapstructure:"log"`
}

// NestedConfig is an unfortunate struct that I had to create to have this nested under the 'controller' key
// in the YAML config
type NestedConfig struct {
	// Configs used only for running mock controller
	EnableUI                        bool    `mapstructure:"enable_ui" survey:"enable_ui"`
	PublishWaterEvent               bool    `mapstructure:"publish_water_event" survey:"publish_water_event"`
	TemperatureValue                float64 `mapstructure:"temperature_value"`
	HumidityValue                   float64 `mapstructure:"humidity_value"`
	TemperatureHumidityDisableNoise bool    `mapstructure:"temperature_humidity_disable_noise"`

	// Configs used for both
	TopicPrefix                 string        `mapstructure:"topic_prefix" survey:"topic_prefix"`
	NumZones                    int           `mapstructure:"num_zones" survey:"num_zones"`
	PublishHealth               bool          `mapstructure:"publish_health" survey:"publish_health"`
	HealthInterval              time.Duration `mapstructure:"health_interval" survey:"health_interval"`
	PublishTemperatureHumidity  bool          `mapstructure:"publish_temperature_humidity" survey:"publish_temperature_humidity"`
	TemperatureHumidityInterval time.Duration `mapstructure:"temperature_humidity_interval" survey:"temperature_humidity_interval"`

	// Configs only used for generate-config
	WifiConfig             `mapstructure:"wifi" survey:"wifi"`
	Zones                  []ZoneConfig `mapstructure:"zones" survey:"zones"`
	LightPin               string       `mapstructure:"light_pin" survey:"light_pin"`
	TemperatureHumidityPin string       `mapstructure:"temperature_humidity_pin" survey:"temperature_humidity_pin"`

	MQTTAddress string `survey:"mqtt_address"`
	MQTTPort    int    `survey:"mqtt_port"`
}

// Controller struct holds the necessary data for running the mock garden-controller
type Controller struct {
	Config
	mqttClient mqtt.Client
	app        *tview.Application

	logger    *slog.Logger
	pubLogger *slog.Logger
	subLogger *slog.Logger

	quit chan os.Signal

	assertionData
}

// NewController creates and initializes everything needed to run a Controller based on config
func NewController(cfg Config) (*Controller, error) {
	controller := &Controller{
		Config: cfg,
		quit:   make(chan os.Signal, 1),
	}

	controller.logger = cfg.LogConfig.NewLogger()
	controller.subLogger = cfg.LogConfig.NewLogger()
	controller.pubLogger = cfg.LogConfig.NewLogger()

	if controller.EnableUI {
		controller.app = controller.setupUI()
	}

	controller.logger.Info("starting controller", "topic_prefix", controller.TopicPrefix)

	topics, err := controller.topics()
	if err != nil {
		return nil, fmt.Errorf("unable to determine topics: %w", err)
	}
	controller.logger.Debug("subscribing to topics", "topics", topics)

	// Build TopicHandlers to handle subscription to each topic
	var handlers []mqtt.TopicHandler
	for _, topic := range topics {
		controller.subLogger.Info("initializing handler for MQTT messages", "topic", topic)
		handlers = append(handlers, mqtt.TopicHandler{
			Topic:   topic,
			Handler: controller.getHandlerForTopic(topic),
		})
	}

	// Override configured ClientID with the TopicPrefix from command flags
	controller.MQTTConfig.ClientID = fmt.Sprint(controller.TopicPrefix)
	controller.mqttClient, err = mqtt.NewClient(controller.MQTTConfig, mqtt.DefaultHandler(controller.logger), handlers...)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize MQTT client: %w", err)
	}
	if err := controller.mqttClient.Connect(); err != nil {
		return nil, fmt.Errorf("unable to connect to MQTT broker: %w", err)
	}

	return controller, nil
}

// Start will run the Controller until it is stopped (blocking)
func (c *Controller) Start() {
	// Initialize scheduler and schedule publishing Jobs
	c.logger.Debug("initializing scheduler")
	scheduler := gocron.NewScheduler(time.Local)
	scheduler.CustomTime(clock.DefaultClock)
	if c.PublishHealth {
		c.logger.Debug("create scheduled job to publish health data", "interval", c.HealthInterval.String())
		_, err := scheduler.Every(c.HealthInterval).Do(c.publishHealthData)
		if err != nil {
			c.logger.Error("error scheduling health publishing", "error", err)
			return
		}
	}
	if c.PublishTemperatureHumidity {
		c.logger.Debug("create scheduled job to publish temperature and humidity data", "interval", c.TemperatureHumidityInterval.String())
		_, err := scheduler.Every(c.TemperatureHumidityInterval).Do(c.publishTemperatureHumidityData)
		if err != nil {
			c.logger.Error("error scheduling temperature and humidity publishing", "error", err)
			return
		}
	}
	scheduler.StartAsync()

	// Shutdown gracefully on Ctrl+C
	wg := &sync.WaitGroup{}
	wg.Add(1)
	signal.Notify(c.quit, os.Interrupt, syscall.SIGTERM)
	var shutdownStart time.Time
	go func() {
		<-c.quit
		shutdownStart = clock.Now()
		c.logger.Info("gracefully shutting down controller")

		scheduler.Stop()

		// Disconnect mqttClient
		c.logger.Info("disconnecting MQTT Client")
		c.mqttClient.Disconnect(1000)
		wg.Done()
	}()

	if c.EnableUI {
		if err := c.app.Run(); err != nil {
			panic(err)
		}
	} else {
		wg.Wait()
	}
	c.logger.Info("controller shutdown gracefully", "time_elapsed", time.Since(shutdownStart))
}

// Stop shuts down the controller
func (c *Controller) Stop() {
	c.quit <- os.Interrupt
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

	c.subLogger = c.Config.LogConfig.NewLoggerWithWriter(tview.ANSIWriter(left))
	c.pubLogger = c.Config.LogConfig.NewLoggerWithWriter(tview.ANSIWriter(right))

	header := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(c.TopicPrefix)

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

// publishHealthData publishes an InfluxDB line to record that the controller is alive and active
func (c *Controller) publishHealthData() {
	topic := fmt.Sprintf("%s/data/health", c.TopicPrefix)
	healthLogger := c.pubLogger.With("topic", topic)
	healthLogger.Info("publishing health data")
	err := c.mqttClient.Publish(topic, fmt.Appendf(nil, "health garden=\"%s\"", c.TopicPrefix))
	if err != nil {
		healthLogger.Error("unable to publish health data", "error", err)
	}
}

func (c *Controller) publishTemperatureHumidityData() {
	temperatureTopic := fmt.Sprintf("%s/data/temperature", c.TopicPrefix)
	humidityTopic := fmt.Sprintf("%s/data/humidity", c.TopicPrefix)

	temperature := c.TemperatureValue
	humidity := c.HumidityValue
	if !c.TemperatureHumidityDisableNoise {
		temperature = addNoise(temperature, 3)
		humidity = addNoise(humidity, 3)
	}

	logger := c.pubLogger.With(
		"temperature", temperature,
		"humidity", humidity,
	)
	logger.Info("publishing temperature and humidity data")

	err := c.mqttClient.Publish(temperatureTopic, fmt.Appendf(nil, "temperature value=%f", temperature))
	if err != nil {
		logger.Error("unable to publish temperature data", "error", err)
	}

	err = c.mqttClient.Publish(humidityTopic, fmt.Appendf(nil, "humidity value=%f", humidity))
	if err != nil {
		logger.Error("unable to publish humidity data", "error", err)
	}
}

// PublishStartupLog publishes the message that controllers use to signal that they started up
func (c *Controller) PublishStartupLog(topicPrefix string) error {
	topic := fmt.Sprintf("%s/data/logs", topicPrefix)
	msg := "logs message=\"garden-controller setup complete\""

	err := c.mqttClient.Publish(topic, []byte(msg))
	if err != nil {
		return fmt.Errorf("error publishing startup log %w", err)
	}

	return nil
}

// addNoise will take a base value and introduce some += variance based on the provided percentage range. This will
// produce sensor data that is relatively consistent but not totally flat
func addNoise(baseValue float64, percentRange float64) float64 {
	// nolint:gosec
	diff := percentRange - (rand.Float64() * percentRange * 2)
	return baseValue + diff
}

// publishWaterEvent publishes completed water events
func (c *Controller) publishWaterEvent(waterMsg action.WaterMessage, cmdTopic string) {
	if !c.PublishWaterEvent {
		c.pubLogger.Debug("publishing water events is disabled")
		return
	}

	// Incoming topic is "{{.TopicPrefix}}/command/water" but we need to publish on "{{.TopicPrefix}}/data/water"
	dataTopic := strings.ReplaceAll(cmdTopic, "command", "data")
	waterEventLogger := c.pubLogger.With(
		"topic", dataTopic,
		"zone_position", waterMsg.Position,
		"duration", waterMsg.Duration,
		"event_id", waterMsg.EventID,
	)
	waterEventLogger.Info("publishing watering event for Zone")

	startMsg := fmt.Sprintf("water,status=start,zone=%d,id=%s,zone_id=%s millis=0", waterMsg.Position, waterMsg.EventID, waterMsg.ZoneID)
	err := c.mqttClient.Publish(dataTopic, []byte(startMsg))
	if err != nil {
		waterEventLogger.Error("unable to publish watering started event", "error", err)
	}

	go func() {
		time.Sleep(time.Duration(waterMsg.Duration) * time.Millisecond)
		doneMsg := fmt.Sprintf("water,status=complete,zone=%d,id=%s,zone_id=%s millis=%d", waterMsg.Position, waterMsg.EventID, waterMsg.ZoneID, waterMsg.Duration)
		err = c.mqttClient.Publish(dataTopic, []byte(doneMsg))
		if err != nil {
			waterEventLogger.Error("unable to publish watering event", "error", err)
		}
	}()
}

// getHandlerForTopic provides a different MessageHandler function for each of the expected
// topics to be able to handle them in different ways
func (c *Controller) getHandlerForTopic(topic string) paho.MessageHandler {
	switch t := strings.Split(topic, "/")[2]; t {
	case "water":
		return c.waterHandler(topic)
	case "stop":
		return c.stopHandler(topic)
	case "stop_all":
		return c.stopAllHandler(topic)
	case "light":
		return c.lightHandler(topic)
	default:
		return paho.MessageHandler(func(_ paho.Client, msg paho.Message) {
			c.subLogger.With(
				"topic", msg.Topic(),
				"message", string(msg.Payload()),
			).Info("received message on unexpected topic")
		})
	}
}

// topics returns a list of topics based on the Config values and provided TopicPrefix
func (c *Controller) topics() ([]string, error) {
	topics := []string{}
	templateFuncs := []func(string) (string, error){
		mqtt.WaterTopic,
		mqtt.StopTopic,
		mqtt.StopAllTopic,
		mqtt.LightTopic,
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
