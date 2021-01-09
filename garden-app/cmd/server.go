package cmd

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/http"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	packageName string
	parentName  string

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

// Run ...
func Run(cmd *cobra.Command, args []string) {
	port := viper.GetInt("web_server.port")

	fmt.Printf("Starting garden-app webserver on port %d...\n", port)
	http.Run(port)
}
