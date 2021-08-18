package cmd

import (
	"github.com/calvinmclean/automated-garden/garden-app/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	numPlants         int
	gardenName        string
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

	rootCommand.AddCommand(controllerCommand)
}

// Controller will start up the mock garden-controller
func Controller(cmd *cobra.Command, args []string) {
	var config controller.Config
	if err := viper.Unmarshal(&config); err != nil {
		cmd.PrintErrln("unable to read config from file: ", err)
		return
	}

	controller.Start(config)
}
