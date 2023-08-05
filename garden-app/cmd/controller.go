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
	moistureStrategy            string
	moistureValue               int
	moistureInterval            time.Duration
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
		Run:     Controller,
	}
)

func init() {
	controllerCommand.PersistentFlags().StringVarP(&topicPrefix, "topic", "t", "test-garden", "MQTT topic prefix of the garden-controller")
	viper.BindPFlag("controller.topic_prefix", controllerCommand.PersistentFlags().Lookup("topic"))

	controllerCommand.PersistentFlags().IntVarP(&numZones, "zones", "z", 0, "Number of Zones for which moisture data should be emulated")
	viper.BindPFlag("controller.num_zones", controllerCommand.PersistentFlags().Lookup("zones"))

	controllerCommand.PersistentFlags().StringVar(&moistureStrategy, "moisture-strategy", "random", "Strategy for creating moisture data")
	controllerCommand.RegisterFlagCompletionFunc("moisture-strategy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"random", "constant", "increasing", "decreasing"}, cobra.ShellCompDirectiveDefault
	})
	viper.BindPFlag("controller.moisture_strategy", controllerCommand.PersistentFlags().Lookup("moisture-strategy"))

	controllerCommand.PersistentFlags().IntVar(&moistureValue, "moisture-value", 100, "The value, or starting value, to use for moisture data publishing")
	viper.BindPFlag("controller.moisture_value", controllerCommand.PersistentFlags().Lookup("moisture-value"))

	controllerCommand.PersistentFlags().DurationVar(&moistureInterval, "moisture-interval", 10*time.Second, "Interval between moisture data publishing")
	viper.BindPFlag("controller.moisture_interval", controllerCommand.PersistentFlags().Lookup("moisture-interval"))

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

	rootCommand.AddCommand(controllerCommand)
}

// Controller will start up the mock garden-controller
func Controller(cmd *cobra.Command, _ []string) {
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
