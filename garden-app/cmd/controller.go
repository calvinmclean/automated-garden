package cmd

import (
	"github.com/calvinmclean/automated-garden/garden-app/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	gardenName           string
	plantsWithMoisture   []int
	publishWateringEvent bool

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

	controllerCommand.Flags().BoolVar(&publishWateringEvent, "publish-watering-event", false, "Whether or not watering events should be published for logging")
	viper.BindPFlag("publish_watering_event", controllerCommand.Flags().Lookup("publish-watering-event"))

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
