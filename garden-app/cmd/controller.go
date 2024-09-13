package cmd

import (
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	topicPrefix                 string
	numZones                    int
	publishWaterEvent           bool
	publishHealth               bool
	healthInterval              time.Duration
	enableUI                    bool
	publishTemperatureHumidity  bool
	temperatureHumidityInterval time.Duration
	temperatureValue            float64
	humidityValue               float64

	controllerCommand = &cobra.Command{
		Use:     "controller",
		Aliases: []string{"controller run"},
		Short:   "Run a mock garden-controller",
		Long:    `Subscribes on a MQTT topic to act as a mock garden-controller for testing purposes`,
		Run:     runController,
	}
)

func init() {
	controllerCommand.PersistentFlags().StringVarP(&topicPrefix, "topic", "t", "test-garden", "MQTT topic prefix of the garden-controller")
	viper.BindPFlag("controller.topic_prefix", controllerCommand.PersistentFlags().Lookup("topic"))

	controllerCommand.PersistentFlags().IntVarP(&numZones, "zones", "z", 0, "Number of Zones")
	viper.BindPFlag("controller.num_zones", controllerCommand.PersistentFlags().Lookup("zones"))

	controllerCommand.PersistentFlags().BoolVar(&publishWaterEvent, "publish-water-event", true, "Whether or not watering events should be published for logging")
	viper.BindPFlag("controller.publish_water_event", controllerCommand.PersistentFlags().Lookup("publish-water-event"))

	controllerCommand.PersistentFlags().BoolVar(&publishHealth, "publish-health", true, "Whether or not to publish health data every minute")
	viper.BindPFlag("controller.publish_health", controllerCommand.PersistentFlags().Lookup("publish-health"))

	controllerCommand.PersistentFlags().DurationVar(&healthInterval, "health-interval", time.Minute, "Interval between health data publishing")
	viper.BindPFlag("controller.health_interval", controllerCommand.PersistentFlags().Lookup("health-interval"))

	controllerCommand.PersistentFlags().BoolVar(&enableUI, "enable-ui", true, "Enable tview UI for nicer output")
	viper.BindPFlag("controller.enable_ui", controllerCommand.PersistentFlags().Lookup("enable-ui"))

	controllerCommand.PersistentFlags().BoolVar(&publishTemperatureHumidity, "publish-temperature-humidity", false, "Whether or not to publish temperature and humidity data")
	viper.BindPFlag("controller.publish_temperature_humidity", controllerCommand.PersistentFlags().Lookup("publish-temperature-humidity"))

	controllerCommand.PersistentFlags().DurationVar(&temperatureHumidityInterval, "temperature-humidity-interval", time.Minute, "Interval for temperature and humidity publishing")
	viper.BindPFlag("controller.temperature_humidity_interval", controllerCommand.PersistentFlags().Lookup("temperature-humidity-interval"))

	controllerCommand.PersistentFlags().Float64Var(&temperatureValue, "temperature-value", 100, "The value to use for temperature data publishing")
	viper.BindPFlag("controller.temperature_value", controllerCommand.PersistentFlags().Lookup("temperature-value"))

	controllerCommand.PersistentFlags().Float64Var(&humidityValue, "humidity-value", 100, "The value to use for humidity data publishing")
	viper.BindPFlag("controller.humidity_value", controllerCommand.PersistentFlags().Lookup("humidity-value"))
}

// runController will start up the mock garden-controller
func runController(cmd *cobra.Command, _ []string) {
	var config controller.Config
	if err := viper.Unmarshal(&config); err != nil {
		cmd.PrintErrln("unable to read config from file:", err)
		return
	}

	controller, err := controller.NewController(config)
	if err != nil {
		cmd.PrintErrln("error creating Controller:", err)
		return
	}
	controller.Start()
}
