package cmd

import (
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	configFilename string
	logLevel       string

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
	viper.SetEnvPrefix("GARDEN_APP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	cobra.OnInitialize(initConfig)

	rootCommand.PersistentFlags().StringVar(&configFilename, "config", "config.yaml", "path to config file")

	rootCommand.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "level of logging to display")
	rootCommand.RegisterFlagCompletionFunc("log-level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"debug", "info", "warn", "error",
		}, cobra.ShellCompDirectiveDefault
	})
	viper.BindPFlag("log.level", rootCommand.PersistentFlags().Lookup("log-level"))
}

func initConfig() {
	if configFilename != "" {
		viper.SetConfigFile(configFilename)
	}

	if err := viper.ReadInConfig(); err == nil {
		slog.Debug("using config file", "config_file", viper.ConfigFileUsed())
	}
}
