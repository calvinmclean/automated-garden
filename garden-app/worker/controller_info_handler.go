package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	lineprotocol "github.com/influxdata/line-protocol"
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

// parseControllerInfoMessage parses an InfluxDB line protocol message with the
// measurement "info" and string fields "mac", "ip", and "version".
func parseControllerInfoMessage(msg string) (*pkg.ControllerInfo, error) {
	handler := lineprotocol.NewMetricHandler()
	parser := lineprotocol.NewParser(handler)
	metrics, err := parser.Parse([]byte(msg))
	if err != nil {
		return nil, fmt.Errorf("error parsing line protocol: %w", err)
	}
	if len(metrics) != 1 {
		return nil, fmt.Errorf("expected 1 metric, got %d", len(metrics))
	}

	m := metrics[0]
	if m.Name() != "info" {
		return nil, fmt.Errorf("unexpected measurement %q", m.Name())
	}

	info := &pkg.ControllerInfo{}
	for _, field := range m.FieldList() {
		if s, ok := field.Value.(string); ok {
			switch field.Key {
			case "mac":
				info.MACAddress = s
			case "ip":
				info.IPAddress = s
			case "version":
				info.FirmwareVersion = s
			}
		}
	}

	return info, nil
}
