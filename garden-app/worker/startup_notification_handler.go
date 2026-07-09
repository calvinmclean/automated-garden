package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	lineprotocol "github.com/influxdata/line-protocol"
)

func (w *Worker) handleControllerLogMessage(_ mqtt.Client, msg mqtt.Message) {
	err := w.getGardenAndHandleLogMessage(msg.Topic(), string(msg.Payload()))
	if err != nil {
		w.logger.With("topic", msg.Topic(), "error", err).Error("error handling controller log message")
	}
}

// controllerLog represents a log message published by a garden controller.
type controllerLog struct {
	Level       string
	Source      string
	Message     string
	ResetReason string
}

func (w *Worker) getGardenAndHandleLogMessage(topic string, payload string) error {
	logger := w.logger.With("topic", topic)

	log, err := parseControllerLogMessage(payload)
	if err != nil {
		logger.Warn("unexpected controller log message", "message", payload, "error", err)
		return nil
	}

	garden, err := w.getGardenForTopic(topic)
	if err != nil {
		return err
	}
	logger = logger.With("garden_id", garden.GetID())

	ctx := context.Background()

	if log.Message == "garden-controller setup complete" {
		return w.handleStartupLog(ctx, garden, topic, log, logger)
	}

	return w.handleGenericLog(ctx, garden, log, logger)
}

func (w *Worker) handleStartupLog(ctx context.Context, garden *pkg.Garden, topic string, log *controllerLog, logger *slog.Logger) error {
	err := w.setExpectedLightState(ctx, garden)
	if err != nil {
		logger.Warn("unable to set expected LightState", "error", err.Error())
		log.Message += fmt.Sprintf(" Error setting LightState: %v", err)
	}

	err = w.setExpectedFanState(ctx, garden)
	if err != nil {
		logger.Warn("unable to set expected FanState", "error", err.Error())
		log.Message += fmt.Sprintf(" Error setting FanState: %v", err)
	}

	if log.ResetReason != "" {
		log.Message += fmt.Sprintf(" reset_reason=%s", log.ResetReason)
	}

	return w.sendGardenStartupMessage(ctx, garden, topic, log.Message)
}

func (w *Worker) handleGenericLog(ctx context.Context, garden *pkg.Garden, log *controllerLog, logger *slog.Logger) error {
	logArgs := []any{
		"level", log.Level,
		"source", log.Source,
		"message", log.Message,
	}

	switch log.Level {
	case "error":
		logger.Error("controller error", logArgs...)
	case "warn":
		logger.Warn("controller warning", logArgs...)
	case "debug":
		logger.Debug("controller debug", logArgs...)
	default:
		logger.Info("controller log", logArgs...)
	}

	if log.Level == "error" && garden.GetNotificationSettings().ControllerErrors {
		title := fmt.Sprintf("%s: Controller Error", garden.Name)
		msg := log.Message
		if log.Source != "" {
			msg = fmt.Sprintf("[%s] %s", log.Source, msg)
		}
		return w.sendNotificationForGarden(ctx, garden, title, msg)
	}

	return nil
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

// parseControllerLogMessage parses an InfluxDB line protocol message with the measurement "logs".
// It supports tags "level" and "source" and the string field "message". If level is omitted, it
// defaults to "info".
func parseControllerLogMessage(msg string) (*controllerLog, error) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return nil, errors.New("empty message")
	}

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
	if m.Name() != "logs" {
		return nil, fmt.Errorf("unexpected measurement %q", m.Name())
	}

	log := &controllerLog{
		Level: "info",
	}
	for _, tag := range m.TagList() {
		switch tag.Key {
		case "level":
			log.Level = tag.Value
		case "source":
			log.Source = tag.Value
		}
	}

	for _, field := range m.FieldList() {
		if s, ok := field.Value.(string); ok {
			switch field.Key {
			case "message":
				log.Message = s
			case "reset_reason":
				log.ResetReason = s
			}
		}
	}

	if log.Message == "" {
		return nil, errors.New("missing message field")
	}

	return log, nil
}
