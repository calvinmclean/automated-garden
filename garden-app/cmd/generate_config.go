package cmd

import (
	"github.com/calvinmclean/automated-garden/garden-app/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	generateConfigCommand = &cobra.Command{
		Use:   "generate-config",
		Short: "Generate config.h and wifi_config.h files for garden-controller",
		Long:  `Uses information from a config file and from an interactive command prompt to generate garden-controller configurations`,
		Run:   GenerateConfig,
	}
)

func init() {
	generateConfigCommand.Flags().StringVarP(&gardenName, "name", "n", "garden", "TODO")
	viper.BindPFlag("garden_name", generateConfigCommand.Flags().Lookup("name"))

	generateConfigCommand.Flags().IntVarP(&numPlants, "plants", "p", 0, "TODO")
	viper.BindPFlag("num_plants", generateConfigCommand.Flags().Lookup("plants"))

	rootCommand.AddCommand(generateConfigCommand)
}

// GenerateConfig is used to help in the creation of garden-controller Arduino configuration files
func GenerateConfig(cmd *cobra.Command, args []string) {
	var config controller.Config
	if err := viper.Unmarshal(&config); err != nil {
		cmd.PrintErrln("unable to read config from file: ", err)
		return
	}
	config.LogLevel = parsedLogLevel

	controller.GenerateConfig(config)
}
