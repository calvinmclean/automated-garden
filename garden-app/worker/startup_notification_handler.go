package worker

import (
	"fmt"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func (w *Worker) handleGardenStartupMessage(_ mqtt.Client, msg mqtt.Message) {
	err := w.doGardenStartupMessage(msg.Topic(), msg.Payload())
	if err != nil {
		w.logger.With("topic", msg.Topic(), "error", err).Error("error handling message")
	}
}

func (w *Worker) doGardenStartupMessage(topic string, payload []byte) error {
	logger := w.logger.With("topic", topic)

	msg := parseStartupMessage(payload)
	if msg != "garden-controller setup complete" {
		logger.Warn("unexpected message from controller", "message", string(payload))
		return nil
	}
	logger.Info("received message", "message", string(payload))

	garden, err := w.getGardenForTopic(topic)
	if err != nil {
		return err
	}
	logger = logger.With("garden_id", garden.GetID())
	logger.Info("found garden with topic-prefix")

	if !garden.GetNotificationSettings().ControllerStartup {
		logger.Info("garden does not have controller_startup notification enabled", "garden_id", garden.GetID())
		return nil
	}

	title := fmt.Sprintf("%s connected", garden.Name)
	return w.sendNotificationForGarden(garden, title, msg, logger)
}

func parseStartupMessage(msg []byte) string {
	return strings.TrimSuffix(strings.TrimPrefix(string(msg), "logs message=\""), "\"")
}
