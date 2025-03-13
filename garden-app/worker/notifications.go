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
	if health.Status != pkg.HealthStatusUp {
		w.sendNotification(
			clientID,
			fmt.Sprintf("%s: %s", g.Name, health.Status),
			fmt.Sprintf(`Attempting to execute %s Action, but last contact was %s.
Details: %s`, actionName, health.LastContact.Format(time.DateTime), health.Details),
			w.logger,
		)
	}
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
