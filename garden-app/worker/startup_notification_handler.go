package worker

import (
	"context"
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
	logger.Debug("found garden with topic-prefix")

	ctx := context.Background()

	err = w.setExpectedLightState(ctx, garden)
	if err != nil {
		logger.Warn("unable to set expected LightState", "error", err.Error())
		msg += fmt.Sprintf(" Error setting LightState: %v", err)
	}

	err = w.setExpectedFanState(ctx, garden)
	if err != nil {
		logger.Warn("unable to set expected FanState", "error", err.Error())
		msg += fmt.Sprintf(" Error setting FanState: %v", err)
	}

	return w.sendGardenStartupMessage(ctx, garden, topic, msg)
}

// setExpectedLightState is used when a GardenController connects/starts up. It sets the current
// expected light state in case the last toggle was missed during downtime or turned off after crashing.
// It is also called by syncLightState when the server schedules/resets a LightSchedule.
func (w *Worker) setExpectedLightState(ctx context.Context, garden *pkg.Garden) error {
	if garden == nil {
		return errors.New("nil Garden")
	}

	if garden.LightSchedule == nil || w.mqttClient == nil {
		return nil
	}

	state := garden.LightSchedule.ExpectedStateAtTime(clock.Now())
	err := w.ExecuteLightAction(ctx, garden, &action.LightAction{
		State: state,
	})
	if err != nil {
		return fmt.Errorf("error executing LightAction: %w", err)
	}

	return nil
}

// setExpectedFanState is used when a GardenController connects/starts up. It runs the fan for the
// remaining duration of the current ON period if the schedule says it should be active now.
func (w *Worker) setExpectedFanState(ctx context.Context, garden *pkg.Garden) error {
	if garden == nil {
		return errors.New("nil Garden")
	}

	if garden.FanSchedule == nil || w.mqttClient == nil {
		return nil
	}

	if !garden.FanSchedule.IsActiveAtTime(clock.Now()) {
		return nil
	}

	if garden.FanSchedule.OnlyWithLight && garden.LightSchedule != nil {
		if garden.LightSchedule.ExpectedStateAtTime(clock.Now()) != pkg.LightStateOn {
			return nil
		}
	}

	nextChange, _ := garden.FanSchedule.NextChange(clock.Now())
	if nextChange.IsZero() {
		return nil
	}

	remainingDuration := nextChange.Sub(clock.Now())
	if remainingDuration <= 0 {
		return nil
	}

	err := w.ExecuteFanAction(ctx, garden, &action.FanAction{
		Duration: remainingDuration.Milliseconds(),
		Power:    garden.FanSchedule.PowerToPWM(),
	})
	if err != nil {
		return fmt.Errorf("error executing FanAction: %w", err)
	}

	return nil
}

func (w *Worker) sendGardenStartupMessage(ctx context.Context, garden *pkg.Garden, topic string, msg string) error {
	if garden == nil {
		return errors.New("nil Garden")
	}
	logger := w.logger.With("garden_id", garden.GetID(), "topic", topic)

	if !garden.GetNotificationSettings().ControllerStartup {
		logger.Warn("garden does not have controller_startup notification enabled")
		return nil
	}

	title := fmt.Sprintf("%s connected", garden.Name)
	return w.sendNotificationForGarden(ctx, garden, title, msg)
}

func parseStartupMessage(msg string) string {
	return strings.TrimSuffix(strings.TrimPrefix(msg, "logs message=\""), "\"")
}
