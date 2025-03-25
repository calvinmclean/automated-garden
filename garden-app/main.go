package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/calvinmclean/automated-garden/garden-app/cmd"
	"github.com/calvinmclean/automated-garden/garden-app/server"

	"gopkg.in/yaml.v3"
)

func main() {
	cmd.Execute()
}

// Run is used for compatibility with Goblin
func Run(ctx context.Context, ipAddress string) error {
	api := server.NewAPI()

	cfgFile, err := os.ReadFile("config.yaml")
	if err != nil {
		return fmt.Errorf("unable to read config from file: %w", err)
	}

	var config server.Config
	err = yaml.Unmarshal(cfgFile, &config)
	if err != nil {
		return fmt.Errorf("unable to read config from file: %w", err)
	}

	err = api.Setup(config, true)
	if err != nil {
		return fmt.Errorf("error setting up API: %w", err)
	}

	api.WithContext(ctx).SetAddress(net.JoinHostPort(ipAddress, "8080"))
	return api.Serve()
}
