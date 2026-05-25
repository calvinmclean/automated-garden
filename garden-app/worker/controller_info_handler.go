package worker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func (w *Worker) handleControllerInfoMessage(_ mqtt.Client, msg mqtt.Message) {
	err := w.getGardenAndSaveControllerInfo(msg.Topic(), string(msg.Payload()))
	if err != nil {
		w.logger.With("topic", msg.Topic(), "error", err).Error("error handling controller info message")
	}
}

func (w *Worker) getGardenAndSaveControllerInfo(topic string, payload string) error {
	logger := w.logger.With("topic", topic)

	info, err := parseControllerInfoMessage(payload)
	if err != nil {
		logger.Warn("unexpected controller info message", "message", payload, "error", err)
		return nil
	}

	garden, err := w.getGardenForTopic(topic)
	if err != nil {
		return err
	}

	info.GardenID = garden.GetID()
	info.UpdatedAt = &[]time.Time{time.Now()}[0]

	logger = logger.With("garden_id", garden.GetID())
	logger.Info("saving controller info")

	return w.storageClient.ControllerInfo.Upsert(context.Background(), info)
}

// parseControllerInfoMessage parses a message in the format:
// info mac="...",ip="...",version="..."
func parseControllerInfoMessage(msg string) (*pkg.ControllerInfo, error) {
	msg = strings.TrimSpace(msg)
	if !strings.HasPrefix(msg, "info ") {
		return nil, errors.New("message does not start with 'info '")
	}

	data := strings.TrimPrefix(msg, "info ")
	info := &pkg.ControllerInfo{}

	pairs := strings.Split(data, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid key=value pair: %q", pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"`)

		switch key {
		case "mac":
			info.MACAddress = value
		case "ip":
			info.IPAddress = value
		case "version":
			info.FirmwareVersion = value
		}
	}

	return info, nil
}
