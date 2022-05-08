package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	configFilename string
	logLevel       string
	parsedLogLevel log.Level

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
	cobra.OnInitialize(initConfig, parseLogLevel)

	rootCommand.PersistentFlags().StringVar(&configFilename, "config", "config.yaml", "path to config file")

	rootCommand.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "level of logging to display")
	rootCommand.RegisterFlagCompletionFunc("log-level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var levels []string
		for _, l := range log.AllLevels {
			levels = append(levels, l.String())
		}
		return levels, cobra.ShellCompDirectiveDefault
	})
}

func initConfig() {
	if configFilename != "" {
		viper.SetConfigFile(configFilename)
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
	}
}

func parseLogLevel() {
	var err error
	parsedLogLevel, err = log.ParseLevel(logLevel)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
