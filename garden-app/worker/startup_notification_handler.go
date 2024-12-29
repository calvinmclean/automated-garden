package worker

import (
	"fmt"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func (w *Worker) handleGardenStartupMessage(_ mqtt.Client, msg mqtt.Message) {
	err := w.getGardenAndSendMessage(msg.Topic(), string(msg.Payload()))
	if err != nil {
		w.logger.With("topic", msg.Topic(), "error", err).Error("error handling message")
	}
}

func (w *Worker) getGardenAndSendMessage(topic string, payload string) error {
	logger := w.logger.With("topic", topic)

	msg := parseStartupMessage(payload)
	if msg != "garden-controller setup complete" {
		logger.Warn("unexpected message from controller", "message", payload)
		return nil
	}
	logger.Info("received message", "message", msg)

	garden, err := w.getGardenForTopic(topic)
	if err != nil {
		return err
	}
	logger = logger.With("garden_id", garden.GetID())
	logger.Info("found garden with topic-prefix")

	return w.sendGardenStartupMessage(garden, topic, msg)
}

func (w *Worker) sendGardenStartupMessage(garden *pkg.Garden, topic string, msg string) error {
	logger := w.logger.With("topic", topic)

	if !garden.GetNotificationSettings().ControllerStartup {
		logger.Warn("garden does not have controller_startup notification enabled", "garden_id", garden.GetID())
		return nil
	}

	title := fmt.Sprintf("%s connected", garden.Name)
	return w.sendNotificationForGarden(garden, title, msg, logger)
}

func parseStartupMessage(msg string) string {
	return strings.TrimSuffix(strings.TrimPrefix(msg, "logs message=\""), "\"")
}
