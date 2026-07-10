package worker

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	lineprotocol "github.com/influxdata/line-protocol"
)

func (w *Worker) handleWaterCompleteStatusMessage(_ mqtt.Client, msg mqtt.Message) {
	err := w.doWaterCompleteStatusMessage(msg.Topic(), msg.Payload())
	if err != nil {
		w.logger.With("topic", msg.Topic(), "error", err).Error("error handling message")
	}
}

func (w *Worker) doWaterCompleteStatusMessage(topic string, payload []byte) error {
	logger := w.logger.With("topic", topic)

	waterMessage, err := parseWaterStatusEvent(payload)
	if err != nil {
		return fmt.Errorf("error parsing message: %w", err)
	}
	logger = logger.With(
		"event_id", waterMessage.EventID,
		"zone_id", waterMessage.ZoneID,
		"status", waterMessage.Status,
	)

	garden, err := w.getGardenForTopic(topic)
	if err != nil {
		return err
	}
	logger = logger.With("garden_id", garden.GetID())
	logger.Debug("found garden with topic-prefix")

	if garden.GetNotificationClientID() == "" {
		logger.Debug("garden does not have notification client", "garden_id", garden.GetID())
		return nil
	}

	logger = logger.With(notificationClientIDLogField, garden.GetNotificationClientID())

	if waterMessage.Status == pkg.WaterStatusStarted && !garden.GetNotificationSettings().WateringStarted {
		logger.Debug("skipping message since notification is not enabled for the start")
		return nil
	}
	if waterMessage.Status != pkg.WaterStatusStarted && !garden.GetNotificationSettings().WateringComplete {
		logger.Debug("skipping message since notification is not enabled")
		return nil
	}

	zone, err := w.storageClient.Zones.Get(context.Background(), waterMessage.ZoneID)
	if err != nil {
		return fmt.Errorf("error getting zone %s: %w", waterMessage.ZoneID, err)
	}
	logger.Debug("found zone")

	var title, message string
	switch waterMessage.Status {
	case pkg.WaterStatusStarted:
		title = fmt.Sprintf("%s started watering", zone.Name)
		message = fmt.Sprintf("Garden: %s", garden.Name)
	case pkg.WaterStatusCancelled:
		title = fmt.Sprintf("%s watering cancelled", zone.Name)
		if waterMessage.Duration > 0 {
			dur := time.Duration(waterMessage.Duration) * time.Millisecond
			message = fmt.Sprintf("Watered for %s\nGarden: %s", dur.String(), garden.Name)
		} else {
			message = fmt.Sprintf("Garden: %s", garden.Name)
		}
	default:
		title = fmt.Sprintf("%s finished watering", zone.Name)
		dur := time.Duration(waterMessage.Duration) * time.Millisecond
		message = fmt.Sprintf("Watered for %s\nGarden: %s", dur.String(), garden.Name)
	}

	return w.sendNotificationForGarden(context.Background(), garden, title, message)
}

func parseInt64Field(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("value %d overflows int64", v)
		}
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("value %d overflows int64", v)
		}
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func parseWaterStatusEvent(msg []byte) (action.WaterStatusEvent, error) {
	handler := lineprotocol.NewMetricHandler()
	parser := lineprotocol.NewParser(handler)
	metrics, err := parser.Parse(msg)
	if err != nil {
		return action.WaterStatusEvent{}, fmt.Errorf("error parsing line protocol: %w", err)
	}
	if len(metrics) != 1 {
		return action.WaterStatusEvent{}, fmt.Errorf("expected 1 metric, got %d", len(metrics))
	}

	m := metrics[0]
	if m.Name() != "water" {
		return action.WaterStatusEvent{}, fmt.Errorf("unexpected measurement %q", m.Name())
	}

	result := action.WaterStatusEvent{}
	for _, tag := range m.TagList() {
		switch tag.Key {
		case "status":
			result.Status = pkg.WaterStatus(tag.Value)
		case "zone":
			zonePos, err := strconv.ParseUint(tag.Value, 10, 0)
			if err != nil {
				return action.WaterStatusEvent{}, fmt.Errorf("invalid integer for position: %w", err)
			}
			result.Position = uint(zonePos)
		case "id":
			result.EventID = tag.Value
		case "zone_id":
			result.ZoneID = tag.Value
		}
	}

	for _, field := range m.FieldList() {
		if field.Key == "millis" {
			duration, err := parseInt64Field(field.Value)
			if err != nil {
				return action.WaterStatusEvent{}, fmt.Errorf("invalid integer for millis: %w", err)
			}
			result.Duration = duration
		}
	}

	if result.Status != "" && result.Status != pkg.WaterStatusStarted &&
		result.Status != pkg.WaterStatusCompleted &&
		result.Status != pkg.WaterStatusCancelled {
		return action.WaterStatusEvent{}, fmt.Errorf("invalid status: %q", result.Status)
	}

	return result, nil
}
