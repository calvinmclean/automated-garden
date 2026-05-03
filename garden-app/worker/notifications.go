package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

func (w *Worker) sendLightActionNotification(g *pkg.Garden, state pkg.LightState, logger *slog.Logger) {
	if g.GetNotificationClientID() == "" {
		return
	}

	if !g.GetNotificationSettings().LightSchedule {
		logger.Info("garden does not have light_schedule notification enabled")
		return
	}

	title := fmt.Sprintf("%s: Light %s", g.Name, state.String())
	w.sendNotification(g.GetNotificationClientID(), title, "Successfully executed LightAction", logger)
}

func (w *Worker) sendDownNotification(g *pkg.Garden, clientID, actionName string) {
	health := w.GetGardenHealth(context.Background(), g)
	if health == nil || health.LastContact == nil {
		return
	}
	if health.Status == pkg.HealthStatusUp {
		return
	}
	w.sendNotification(
		clientID,
		fmt.Sprintf("%s: %s", g.Name, health.Status),
		fmt.Sprintf(`Attempting to execute %s Action, but last contact was %s.
Details: %s`, actionName, health.LastContact.Format(time.DateTime), health.Details),
		w.logger,
	)
}

func (w *Worker) sendWateringReminder(ws *pkg.WaterSchedule, duration time.Duration, zoneCount int, logger *slog.Logger) {
	if ws.GetNotificationClientID() == "" || ws.SendReminder == nil || !*ws.SendReminder {
		return
	}

	title, message := generateWateringNotificationContent(ws, duration, zoneCount)
	w.sendNotification(ws.GetNotificationClientID(), title, message, logger)
}

func generateWateringNotificationContent(ws *pkg.WaterSchedule, duration time.Duration, zoneCount int) (string, string) {
	var title, message string
	if zoneCount > 0 {
		title = fmt.Sprintf("Watering %d Zone", zoneCount)
		if zoneCount > 1 {
			title += "s"
		}
		if ws.Name != "" {
			title += fmt.Sprintf(": %s", ws.Name)
		}
	} else {
		title = "Watering Reminder"
		if ws.Name != "" {
			title += fmt.Sprintf(": %s", ws.Name)
		}
	}

	if duration == 0 {
		message = "Weather conditions suggest skipping watering today"
	} else {
		baseDuration := ws.Duration.Duration
		message = fmt.Sprintf("Duration: %s", duration)
		if duration != baseDuration {
			scaleFactor := float64(duration) / float64(baseDuration)
			message += fmt.Sprintf(" (base: %s, scaled %.2fx)", baseDuration, scaleFactor)
		}
	}

	return title, message
}

func (w *Worker) sendNotification(clientID, title, msg string, logger *slog.Logger) {
	ncLogger := logger.With("notification_client_id", clientID)

	notificationClient, err := w.storageClient.NotificationClientConfigs.Get(context.Background(), clientID)
	if err != nil {
		ncLogger.Error("error getting notification client", "error", err)
		return
	}

	err = notificationClient.SendMessage(title, msg)
	if err != nil {
		ncLogger.Error("error sending message", "error", err)
		return
	}

	ncLogger.Info("successfully send notification")
}
