package server

import (
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
)

// Config holds all the options and sub-configs for the server
type Config struct {
	WebConfig      WebConfig       `mapstructure:"web_server" yaml:"web_server"`
	InfluxDBConfig influxdb.Config `mapstructure:"influxdb" yaml:"influxdb"`
	MQTTConfig     mqtt.Config     `mapstructure:"mqtt" yaml:"mqtt"`
	StorageConfig  storage.Config  `mapstructure:"storage" yaml:"storage"`
	LogConfig      LogConfig       `mapstructure:"log" yaml:"log"`
}

// WebConfig is used to allow reading the "web_server" section into the main Config struct
type WebConfig struct {
	Port           int  `mapstructure:"port" yaml:"port"`
	ReadOnly       bool `mapstructure:"readonly" yaml:"readonly"`
	DisableMetrics bool `mapstructure:"disable_metrics" yaml:"disable_metrics"`
}
