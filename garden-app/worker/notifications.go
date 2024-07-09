package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

func (w *Worker) sendLightActionNotification(g *pkg.Garden, state pkg.LightState, logger *slog.Logger) {
	if g.LightSchedule.GetNotificationClientID() == "" {
		return
	}

	title := fmt.Sprintf("%s: Light %s", g.Name, state.String())
	w.sendNotificationWithClientID(g.LightSchedule.GetNotificationClientID(), title, "Successfully executed LightAction", logger)
}

func (w *Worker) sendNotification(title, msg string, logger *slog.Logger) {
	// TODO: this might end up getting client from garden or zone config instead of using all
	notificationClients, err := w.storageClient.NotificationClientConfigs.GetAll(context.Background(), nil)
	if err != nil {
		logger.Error("error getting all notification clients", "error", err)
		return
	}

	for _, nc := range notificationClients {
		w.sendNotificationWithClientID(nc.GetID(), title, msg, logger)
	}
}

func (w *Worker) sendNotificationWithClientID(clientID, title, msg string, logger *slog.Logger) {
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
