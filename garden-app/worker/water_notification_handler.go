package worker

import (
	"fmt"
	"time"

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

	zonePosition, waterDuration, err := parseWaterMessage(payload)
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

	zone, err := w.getZone(garden.GetID(), zonePosition)
	if err != nil {
		return fmt.Errorf("error getting zone with position %d: %w", zonePosition, err)
	}
	logger.Info("found zone with position", "zone_position", zonePosition, "zone_id", zone.GetID())

	title := fmt.Sprintf("%s finished watering", zone.Name)
	message := fmt.Sprintf("watered for %s", waterDuration.String())
	return w.sendNotificationForGarden(garden, title, message, logger)
}

func parseWaterMessage(msg []byte) (int, time.Duration, error) {
	p := &parser{msg, 0}
	zonePosition, err := p.readNextInt()
	if err != nil {
		return 0, 0, fmt.Errorf("error parsing zone position: %w", err)
	}

	waterMillis, err := p.readNextInt()
	if err != nil {
		return 0, 0, fmt.Errorf("error parsing watering time: %w", err)
	}
	waterDuration := time.Duration(waterMillis) * time.Millisecond

	return zonePosition, waterDuration, nil
}
