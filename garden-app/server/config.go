package server

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
)

// Config holds all the options and sub-configs for the server
type Config struct {
	WebConfig      `mapstructure:"web_server"`
	InfluxDBConfig influxdb.Config `mapstructure:"influxdb"`
	MQTTConfig     mqtt.Config     `mapstructure:"mqtt"`
	StorageConfig  storage.Config  `mapstructure:"storage"`
	LogConfig      LogConfig       `mapstructure:"log"`
}

// WebConfig is used to allow reading the "web_server" section into the main Config struct
type WebConfig struct {
	Port     int  `mapstructure:"port"`
	ReadOnly bool `mapstructure:"readonly"`
}
