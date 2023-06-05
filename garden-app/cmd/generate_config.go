package cmd

import (
	"github.com/calvinmclean/automated-garden/garden-app/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	wifiSSID    string
	writeFile   bool
	mainConfig  bool
	wifiConfig  bool
	overwrite   bool
	interactive bool

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

	generateConfigCommand.Flags().BoolVarP(&writeFile, "write", "w", false, "write results to file instead of stdout")
	generateConfigCommand.Flags().BoolVar(&wifiConfig, "wifi-config", true, "enable generating 'wifi_config.h'")
	generateConfigCommand.Flags().BoolVar(&mainConfig, "main-config", true, "enable generating 'config.h'")
	generateConfigCommand.Flags().BoolVarP(&overwrite, "force", "f", false, "overwrite files if they already exist")
	generateConfigCommand.Flags().BoolVarP(&interactive, "interactive", "i", false, "guided prompts help you setup the configuration")

	controllerCommand.AddCommand(generateConfigCommand)
}

// GenerateConfig is used to help in the creation of garden-controller Arduino configuration files
func GenerateConfig(cmd *cobra.Command, _ []string) {
	var config controller.Config
	if err := viper.Unmarshal(&config); err != nil {
		cmd.PrintErrln("unable to read config from file: ", err)
		return
	}

	controller.GenerateConfig(config, writeFile, wifiConfig, mainConfig, overwrite, interactive)
}
