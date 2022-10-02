package cmd

import (
	"github.com/calvinmclean/automated-garden/garden-app/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	serverCommand = &cobra.Command{
		Use:     "server",
		Aliases: []string{"run"},
		Short:   "Run the webserver",
		Long:    `Runs the main webserver application`,
		Run:     Server,
	}
)

func init() {
	serverCommand.Flags().Int("port", 80, "port to run Application server on")
	viper.BindPFlag("web_server.port", serverCommand.Flags().Lookup("port"))

	rootCommand.AddCommand(serverCommand)
}

// Server will execute the Run function provided by the `server` package for running the webserver
func Server(cmd *cobra.Command, args []string) {
	var config server.Config
	if err := viper.Unmarshal(&config); err != nil {
		cmd.PrintErrln("unable to read config from file:", err)
		return
	}
	config.LogLevel = parsedLogLevel

	cmd.Printf("Starting garden-app webserver on port %d...\n", config.Port)
	server, err := server.NewServer(config)
	if err != nil {
		cmd.PrintErrln("error creating HTTP Server:", err)
		return
	}
	server.Start()
}
