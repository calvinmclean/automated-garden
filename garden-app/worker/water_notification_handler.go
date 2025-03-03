package worker

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func (w *Worker) handleWaterCompleteMessage(_ mqtt.Client, msg mqtt.Message) {
	err := w.doWaterCompleteMessage(msg.Topic(), msg.Payload())
	if err != nil {
		w.logger.With("topic", msg.Topic(), "error", err).Error("error handling message")
	}
}

func (w *Worker) doWaterCompleteMessage(topic string, payload []byte) error {
	logger := w.logger.With("topic", topic)
	logger.Info("received message", "message", string(payload))

	waterMessage, err := parseWaterMessage(payload)
	if err != nil {
		return fmt.Errorf("error parsing message: %w", err)
	}

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

	zone, err := w.storageClient.Zones.Get(context.Background(), waterMessage.ZoneID)
	if err != nil {
		return fmt.Errorf("error getting zone %s: %w", waterMessage.ZoneID, err)
	}
	logger.Info("found zone", "zone_id", zone.GetID())

	title := fmt.Sprintf("%s finished watering", zone.Name)
	dur := time.Duration(waterMessage.Duration) * time.Millisecond
	message := fmt.Sprintf("watered for %s", dur.String())
	return w.sendNotificationForGarden(garden, title, message)
}

func parseWaterMessage(msg []byte) (action.WaterMessage, error) {
	p := parser{data: bytes.TrimPrefix(msg, []byte("water,"))}

	result := action.WaterMessage{}

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
				return action.WaterMessage{}, fmt.Errorf("invalid integer for position: %w", err)
			}
			result.Position = uint(zonePos)
		case "millis":
			dur, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return action.WaterMessage{}, fmt.Errorf("invalid integer for millis: %w", err)
			}
			result.Duration = dur
		case "id":
			result.EventID = strings.Trim(val, `"`)
		case "zone_id":
			result.ZoneID = strings.Trim(val, `"`)
		}

		part, err = p.readNextPair()
	}
	if err != nil {
		return action.WaterMessage{}, fmt.Errorf("error reading next pair: %w", err)
	}

	return result, nil
}
