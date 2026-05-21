package worker

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	mqtt "github.com/eclipse/paho.mqtt.golang"
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
	logger.Info("found garden with topic-prefix")

	if garden.GetNotificationClientID() == "" {
		logger.Info("garden does not have notification client", "garden_id", garden.GetID())
		return nil
	}

	logger = logger.With(notificationClientIDLogField, garden.GetNotificationClientID())

	if waterMessage.Status == pkg.WaterStatusStarted && !garden.GetNotificationSettings().WateringStarted {
		logger.Info("skipping message since notification is not enabled for the start")
		return nil
	}
	if waterMessage.Status != pkg.WaterStatusStarted && !garden.GetNotificationSettings().WateringComplete {
		logger.Info("skipping message since notification is not enabled")
		return nil
	}

	zone, err := w.storageClient.Zones.Get(context.Background(), waterMessage.ZoneID)
	if err != nil {
		return fmt.Errorf("error getting zone %s: %w", waterMessage.ZoneID, err)
	}
	logger.Info("found zone")

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

	return w.sendNotificationForGarden(garden, title, message)
}

func parseWaterStatusEvent(msg []byte) (action.WaterStatusEvent, error) {
	p := parser{data: bytes.TrimPrefix(msg, []byte("water,"))}

	result := action.WaterStatusEvent{}

	part, err := p.readNextPair()
	for part != "" && err == nil {
		key, val, found := strings.Cut(part, "=")
		if !found {
			continue
		}

		switch key {
		case "zone":
			zonePos, err := strconv.ParseUint(val, 10, 0)
			if err != nil {
				return action.WaterStatusEvent{}, fmt.Errorf("invalid integer for position: %w", err)
			}
			result.Position = uint(zonePos)
		case "millis":
			dur, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return action.WaterStatusEvent{}, fmt.Errorf("invalid integer for millis: %w", err)
			}
			result.Duration = dur
		case "id":
			result.EventID = strings.Trim(val, `"`)
		case "zone_id":
			result.ZoneID = strings.Trim(val, `"`)
		case "status":
			result.Status = pkg.WaterStatus(val)
		}

		part, err = p.readNextPair()
	}
	if err != nil {
		return action.WaterStatusEvent{}, fmt.Errorf("error reading next pair: %w", err)
	}

	if result.Status != "" && result.Status != pkg.WaterStatusStarted &&
		result.Status != pkg.WaterStatusCompleted &&
		result.Status != pkg.WaterStatusCancelled {
		return action.WaterStatusEvent{}, fmt.Errorf("invalid status: %q", result.Status)
	}

	return result, nil
}
