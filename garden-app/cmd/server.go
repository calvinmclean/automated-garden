package cmd

import (
	"github.com/calvinmclean/automated-garden/garden-app/http"
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

	serverCommand.Flags().StringVar(&plantFile, "plants", "plants.yaml", "path to plants file")
	viper.BindPFlag("web_server.plants_filename", serverCommand.Flags().Lookup("plants"))
}

// Run will execute the Run function provided by the `http` package for running the webserver
func Run(cmd *cobra.Command, args []string) {
	port := viper.GetInt("web_server.port")
	plantsFilename := viper.GetString("web_server.plants_filename")

	cmd.Printf("Starting garden-app webserver on port %d...\n", port)
	http.Run(port, plantsFilename)
}
