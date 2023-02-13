package server

import (
	"github.com/sirupsen/logrus"
)

// LogConfig holds settings for logger
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// GetFormatter returns a logrus formatter based on the input. Valid values are "json", otherwise default text is used
func (c LogConfig) GetFormatter() logrus.Formatter {
	switch c.Format {
	case "json":
		return &logrus.JSONFormatter{}
	default:
		return &logrus.TextFormatter{
			DisableColors: false,
			ForceColors:   true,
			FullTimestamp: true,
		}
	}
}

// GetLogLevel returns the logrus Level based on parsed string. Defaults to InfoLevel instead of error
func (c LogConfig) GetLogLevel() logrus.Level {
	parsedLevel, err := logrus.ParseLevel(c.Level)
	if err != nil {
		logrus.Warnf("unable to parse log level, defaulting to INFO: %v", err)
		parsedLevel = logrus.InfoLevel
	}
	return parsedLevel
}
