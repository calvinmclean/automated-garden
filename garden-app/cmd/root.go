package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	configFilename string

	rootCommand = &cobra.Command{
		Use:   "garden-app",
		Short: "A command line application for the automated home garden",
		Long:  `This CLI is used to run and interact with this webserver application for your automated home garden`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCommand.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCommand.PersistentFlags().StringVar(&configFilename, "config", "", "path to config file")
}

func initConfig() {
	if configFilename != "" {
		viper.SetConfigFile(configFilename)
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
