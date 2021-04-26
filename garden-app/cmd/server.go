package cmd

import (
	"github.com/calvinmclean/automated-garden/garden-app/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	plantFile string

	serverCommand = &cobra.Command{
		Use:     "server",
		Aliases: []string{"run"},
		Short:   "Run the webserver",
		Long:    `Runs the main webserver application`,
		Run:     Run,
	}
)

func init() {
	serverCommand.Flags().Int("port", 80, "port to run Application server on")
	viper.BindPFlag("web_server.port", serverCommand.Flags().Lookup("port"))
}

// Run will execute the Run function provided by the `server` package for running the webserver
func Run(cmd *cobra.Command, args []string) {
	var config server.Config
	if err := viper.Unmarshal(&config); err != nil {
		cmd.PrintErrln("unable to read config from file: ", err)
		return
	}

	cmd.Printf("Starting garden-app webserver on port %d...\n", config.Port)
	server.Run(config)
}
