package cmd

import (
	"github.com/calvinmclean/automated-garden/garden-app/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	wifiSSID string

	generateConfigCommand = &cobra.Command{
		Use:   "generate-config",
		Short: "Generate config.h and wifi_config.h files for garden-controller",
		Long:  `Uses information from a config file and from an interactive command prompt to generate garden-controller configurations`,
		Run:   GenerateConfig,
	}
)

func init() {
	generateConfigCommand.Flags().StringVar(&wifiSSID, "ssid", "", "SSID for your WiFi network")
	viper.BindPFlag("controller.wifi.ssid", generateConfigCommand.Flags().Lookup("ssid"))

	controllerCommand.AddCommand(generateConfigCommand)
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
