package cmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFilename string
	logLevel       string
)

func Execute() {
	api := server.NewAPI()
	command := api.Command()

	command.AddCommand(controllerCommand)

	viper.SetEnvPrefix("GARDEN_APP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	cobra.OnInitialize(initConfig)

	command.PersistentFlags().StringVar(&configFilename, "config", "config.yaml", "path to config file")

	command.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "level of logging to display")
	err := command.RegisterFlagCompletionFunc("log-level", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"debug", "info", "warn", "error",
		}, cobra.ShellCompDirectiveDefault
	})
	if err != nil {
		panic(err)
	}

	viper.BindPFlag("log.level", command.PersistentFlags().Lookup("log-level"))

	command.PersistentPreRunE = func(c *cobra.Command, _ []string) error {
		if c.Name() != "serve" {
			return nil
		}

		var config server.Config
		err := viper.Unmarshal(&config)
		if err != nil {
			return fmt.Errorf("unable to read config from file: %w", err)
		}

		err = api.Setup(config, true)
		if err != nil {
			return fmt.Errorf("error setting up API: %w", err)
		}

		return nil
	}

	for _, c := range command.Commands() {
		if c.Name() != "serve" {
			continue
		}

		c.Flags().Int("port", 80, "port to run Application server on")
		viper.BindPFlag("web_server.port", c.Flags().Lookup("port"))

		c.Flags().Bool("readonly", false, "run in read-only mode so server will only allow GET requests")
		viper.BindPFlag("web_server.readonly", c.Flags().Lookup("readonly"))
	}

	err = command.Execute()
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}

func init() {
	viper.SetEnvPrefix("GARDEN_APP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	cobra.OnInitialize(initConfig)
}

func initConfig() {
	if configFilename != "" {
		viper.SetConfigFile(configFilename)
	}

	if err := viper.ReadInConfig(); err == nil {
		slog.Debug("using config file", "config_file", viper.ConfigFileUsed())
	}
}
