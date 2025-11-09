package worker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func (w *Worker) handleGardenStartupMessage(_ mqtt.Client, msg mqtt.Message) {
	err := w.getGardenAndSendStartupMessage(msg.Topic(), string(msg.Payload()))
	if err != nil {
		w.logger.With("topic", msg.Topic(), "error", err).Error("error handling message")
	}
}

func (w *Worker) getGardenAndSendStartupMessage(topic string, payload string) error {
	logger := w.logger.With("topic", topic)

	msg := parseStartupMessage(payload)
	if msg != "garden-controller setup complete" {
		logger.Warn("unexpected message from controller", "message", payload)
		return nil
	}

	garden, err := w.getGardenForTopic(topic)
	if err != nil {
		return err
	}
	logger = logger.With("garden_id", garden.GetID())
	logger.Info("found garden with topic-prefix")

	err = w.setExpectedLightState(garden)
	if err != nil {
		logger.Warn("unable to set expected LightState", "error", err.Error())
		msg += fmt.Sprintf(" Error setting LightState: %v", err)
	}

	return w.sendGardenStartupMessage(garden, topic, msg)
}

// setExpectedLightState is used when a GardenController connects/starts up. It sets the current
// expected light state in case the last toggle was missed during downtime or turned off after crashing
func (w *Worker) setExpectedLightState(garden *pkg.Garden) error {
	if garden == nil {
		return errors.New("nil Garden")
	}

	if garden.LightSchedule == nil {
		return nil
	}

	state := garden.LightSchedule.ExpectedStateAtTime(clock.Now())
	err := w.ExecuteLightAction(garden, &action.LightAction{
		State: state,
	})
	if err != nil {
		return fmt.Errorf("error executing LigthAction: %w", err)
	}

	return nil
}

func (w *Worker) sendGardenStartupMessage(garden *pkg.Garden, topic string, msg string) error {
	if garden == nil {
		return errors.New("nil Garden")
	}
	logger := w.logger.With("garden_id", garden.GetID(), "topic", topic)

	if !garden.GetNotificationSettings().ControllerStartup {
		logger.Warn("garden does not have controller_startup notification enabled")
		return nil
	}

	title := fmt.Sprintf("%s connected", garden.Name)
	return w.sendNotificationForGarden(garden, title, msg)
}

func parseStartupMessage(msg string) string {
	return strings.TrimSuffix(strings.TrimPrefix(msg, "logs message=\""), "\"")
}
