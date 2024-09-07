package controller

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"regexp"
	"time"

	"github.com/AlecAivazis/survey/v2"
)

const (
	configTemplate = `#ifndef config_h
#define config_h

#define TOPIC_PREFIX "{{ .TopicPrefix }}"

#define QUEUE_SIZE 10

#define ENABLE_WIFI
#ifdef ENABLE_WIFI
#define MQTT_ADDRESS "{{ .MQTTConfig.Broker }}"
#define MQTT_PORT {{ .MQTTConfig.Port }}

#endif

#define NUM_ZONES {{ len .Zones }}
#define VALVES { {{ range $index, $z := .Zones }}{{if $index}}, {{end}}{{ $z.ValvePin }}{{ end }} }
#define PUMPS { {{ range $index, $z := .Zones }}{{if $index}}, {{end}}{{ $z.PumpPin }}{{ end }} }

{{ if .LightPin }}
#define LIGHT_ENABLED true
#define LIGHT_PIN {{ .LightPin }}
{{ else }}
#define LIGHT_ENABLED false
#define LIGHT_PIN GPIO_NUM_MAX
{{ end }}

{{ if .PublishTemperatureHumidity }}
#define ENABLE_DHT22 true
#define DHT22_PIN {{ .TemperatureHumidityPin }}
#define DHT22_INTERVAL {{ milliseconds .TemperatureHumidityInterval }}
{{ else }}
#define ENABLE_DHT22 false
#define DHT22_PIN GPIO_NUM_MAX
#define DHT22_INTERVAL 0
{{ end -}}
#endif
`
	wifiConfigTemplate = `#ifndef wifi_config_h
#define wifi_config_h

#define SSID "{{ .SSID }}"
#define PASSWORD "{{ .Password }}"

#endif
`
)

// WifiConfig holds WiFi connection details
type WifiConfig struct {
	SSID     string `mapstructure:"ssid"`
	Password string `mapstructure:"password"`
}

// ZoneConfig has the configuration details for controlling hardware pins
type ZoneConfig struct {
	PumpPin  string `mapstructure:"pump_pin" survey:"pump_pin"`
	ValvePin string `mapstructure:"valve_pin" survey:"valve_pin"`
}

// GenerateConfig will create config.h and wifi_config.h based on the provided configurations. It can optionally write to files
// instead of stdout
func GenerateConfig(config Config, writeFile, genWifiConfig, genMainConfig, overwrite, interactive bool) {
	logger := config.LogConfig.NewLogger()

	if interactive {
		err := survey.AskOne(&survey.Confirm{
			Message: "Generate 'config.h'?",
			Default: genMainConfig,
		}, &genMainConfig)
		if err != nil {
			logger.Error("survey error", "error", err)
			return
		}
	}

	if genMainConfig {
		logger.Debug("generating 'config.h'")
		mainConfig, err := generateMainConfig(config, interactive)
		if err != nil {
			logger.Error("error generating 'config.h'", "error", err)
			return
		}
		err = writeOutput(logger, mainConfig, "config.h", writeFile, overwrite, interactive)
		if err != nil {
			logger.Error("error generating 'config.h'", "error", err)
			return
		}
	}

	if interactive {
		err := survey.AskOne(&survey.Confirm{
			Message: "Generate 'wifi_config.h'?",
			Default: genWifiConfig,
		}, &genWifiConfig)
		if err != nil {
			logger.Error("survey error", "error", err)
			return
		}
	}

	if genWifiConfig {
		logger.Debug("generating 'wifi_config.h'")
		wifiConfig, err := generateWiFiConfig(config.WifiConfig, interactive)
		if err != nil {
			logger.Error("error generating 'wifi_config.h'", "error", err)
			return
		}
		err = writeOutput(logger, wifiConfig, "wifi_config.h", writeFile, overwrite, interactive)
		if err != nil {
			logger.Error("error generating 'wifi_config.h'", "error", err)
			return
		}
	}
}

func writeOutput(logger *slog.Logger, content, filename string, writeFile, overwrite, interactive bool) error {
	logger.With(
		"filename", filename,
		"write_file", writeFile,
		"overwrite_file", overwrite,
	).Debug("writing output to file")

	if interactive {
		err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Write generated config to %q?", filename),
			Default: writeFile,
		}, &writeFile)
		if err != nil {
			return err
		}
	}

	file := os.Stdout
	// if configured to write to a file, replace os.Stdout with file
	if writeFile {
		// if overwrite is false, first check if file exists and error if it does
		if !overwrite {
			_, err := os.Stat(filename)
			if err == nil {
				if interactive {
					err := survey.AskOne(&survey.Confirm{
						Message: fmt.Sprintf("Overwrite existing %q?", filename),
						Default: overwrite,
					}, &overwrite)
					if err != nil {
						return err
					}
				}
				if !overwrite {
					return fmt.Errorf("file %q exists, use --force to overwrite", filename)
				}
			}
		}

		var err error
		file, err = os.Create(filename)
		if err != nil {
			return err
		}
	}

	_, err := file.WriteString(content)
	if err != nil {
		return err
	}
	return nil
}

func generateMainConfig(config Config, interactive bool) (string, error) {
	if interactive {
		err := configPrompts(&config)
		if err != nil {
			return "", err
		}
	}

	milliseconds := func(interval time.Duration) string {
		return fmt.Sprintf("%d", interval.Milliseconds())
	}
	t := template.Must(template.
		New("config.h").
		Funcs(template.FuncMap{"milliseconds": milliseconds}).
		Parse(configTemplate))

	var result bytes.Buffer
	data := config
	err := t.Execute(&result, data)
	if err != nil {
		return "", err
	}
	return removeExtraNewlines(result.String()), nil
}

func generateWiFiConfig(config WifiConfig, interactive bool) (string, error) {
	qs := []*survey.Question{
		{
			Name: "ssid",
			Prompt: &survey.Input{
				Message: "WiFi SSID",
				Default: config.SSID,
				Help:    "this is the name of your WiFi network",
			},
			Validate: survey.Required,
		},
		{
			Name: "password",
			Prompt: &survey.Password{
				Message: "Password",
				Help:    "this is your WiFi password",
			},
			Validate: survey.Required,
		},
	}

	// if not interactive, but password is missing, turn interactive with password question only
	if config.Password == "" && !interactive {
		qs = qs[1:]
		interactive = true
	}

	if interactive {
		err := survey.Ask(qs, &config)
		if err != nil {
			return "", fmt.Errorf("error in survey response: %w", err)
		}
	}

	t := template.Must(template.New("wifi_config.h").Parse(wifiConfigTemplate))
	var result bytes.Buffer
	err := t.Execute(&result, config)
	if err != nil {
		return "", err
	}
	return removeExtraNewlines(result.String()), nil
}

func removeExtraNewlines(input string) string {
	return regexp.MustCompile(`(?m)^\n{2,}`).ReplaceAllLiteralString(input, "\n")
}
