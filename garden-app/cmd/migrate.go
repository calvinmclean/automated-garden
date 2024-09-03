package cmd

import (
	"fmt"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var migrateCommand = &cobra.Command{
	Use:   "migrate",
	Short: "Run storage migrations to update all resources",
	RunE: func(cmd *cobra.Command, _ []string) error {
		var config server.Config
		err := viper.Unmarshal(&config)
		if err != nil {
			return fmt.Errorf("unable to read config from file: %w", err)
		}

		storageClient, err := storage.NewClient(config.StorageConfig)
		if err != nil {
			return fmt.Errorf("unable to initialize storage client: %v", err)
		}

		return storageClient.RunMigrations(cmd.Context())
	},
}
