package cmd

import (
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	gardenName           string
	plantsWithMoisture   []int
	moistureStrategy     string
	moistureValue        int
	moistureInterval     time.Duration
	publishWateringEvent bool
	publishHealth        bool
	healthInterval       time.Duration

	controllerCommand = &cobra.Command{
		Use:   "controller",
		Short: "Run a mock garden-controller",
		Long:  `Subscribes on a MQTT topic to act as a mock garden-controller for testing purposes`,
		Run:   Controller,
	}
)

func init() {
	controllerCommand.Flags().StringVarP(&gardenName, "name", "n", "garden", "Name of the garden-controller (helps determine which MQTT topic to subscribe to)")
	viper.BindPFlag("garden_name", controllerCommand.Flags().Lookup("name"))

	controllerCommand.Flags().IntSliceVarP(&plantsWithMoisture, "plants", "p", []int{}, "Plant positions for which moisture data should be emulated")
	viper.BindPFlag("plants", controllerCommand.Flags().Lookup("plants"))

	controllerCommand.Flags().StringVar(&moistureStrategy, "moisture-strategy", "random", "Strategy for creating moisture data")
	controllerCommand.RegisterFlagCompletionFunc("moisture-strategy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"random", "constant", "increasing", "decreasing"}, cobra.ShellCompDirectiveDefault
	})
	viper.BindPFlag("moisture_strategy", controllerCommand.Flags().Lookup("moisture-strategy"))

	controllerCommand.Flags().IntVar(&moistureValue, "moisture-value", 100, "The value, or starting value, to use for moisture data publishing")
	viper.BindPFlag("moisture_value", controllerCommand.Flags().Lookup("moisture-value"))

	controllerCommand.Flags().DurationVar(&moistureInterval, "moisture-interval", 10*time.Second, "Interval between moisture data publishing")
	viper.BindPFlag("moisture_interval", controllerCommand.Flags().Lookup("moisture-interval"))

	controllerCommand.Flags().BoolVar(&publishWateringEvent, "publish-watering-event", false, "Whether or not watering events should be published for logging")
	viper.BindPFlag("publish_watering_event", controllerCommand.Flags().Lookup("publish-watering-event"))

	controllerCommand.Flags().BoolVar(&publishHealth, "publish-health", false, "Whether or not to publish health data every minute")
	viper.BindPFlag("publish_health", controllerCommand.Flags().Lookup("publish-health"))

	controllerCommand.Flags().DurationVar(&healthInterval, "health-interval", time.Minute, "Interval between health data publishing")
	viper.BindPFlag("health_interval", controllerCommand.Flags().Lookup("health-interval"))

	rootCommand.AddCommand(controllerCommand)
}

// Controller will start up the mock garden-controller
func Controller(cmd *cobra.Command, args []string) {
	var config controller.Config
	if err := viper.Unmarshal(&config); err != nil {
		cmd.PrintErrln("unable to read config from file: ", err)
		return
	}
	config.LogLevel = parsedLogLevel

	controller.Start(config)
}
