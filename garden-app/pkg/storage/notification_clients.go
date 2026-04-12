package storage

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/db"
	"github.com/calvinmclean/babyapi"
)

type NotificationClientStorage struct {
	q *db.Queries
}

var _ babyapi.Storage[*notifications.Client] = &NotificationClientStorage{}

func NewNotificationClientStorage(sqlDB *sql.DB) *NotificationClientStorage {
	return &NotificationClientStorage{
		q: db.New(sqlDB),
	}
}

func (s *NotificationClientStorage) Get(ctx context.Context, id string) (*notifications.Client, error) {
	dbNotificationClient, err := s.q.GetNotificationClient(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, babyapi.ErrNotFound
		}
		return nil, fmt.Errorf("error getting notification client: %w", err)
	}

	return dbNotificationClientToNotificationClient(dbNotificationClient)
}

func (s *NotificationClientStorage) Search(ctx context.Context, _ string, _ url.Values) ([]*notifications.Client, error) {
	dbNotificationClients, err := s.q.ListNotificationClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing notification clients: %w", err)
	}

	notificationClients := make([]*notifications.Client, len(dbNotificationClients))
	for i, dbNotificationClient := range dbNotificationClients {
		notificationClient, err := dbNotificationClientToNotificationClient(dbNotificationClient)
		if err != nil {
			return nil, fmt.Errorf("invalid notification client: %w", err)
		}

		notificationClients[i] = notificationClient
	}

	return notificationClients, nil
}

func (s *NotificationClientStorage) Set(ctx context.Context, notificationClient *notifications.Client) error {
	return s.q.UpsertNotificationClient(ctx, db.UpsertNotificationClientParams{
		ID:   notificationClient.ID.String(),
		Name: notificationClient.Name,
		URL:  notificationClient.URL,
	})
}

func (s *NotificationClientStorage) Delete(ctx context.Context, id string) error {
	return s.q.DeleteNotificationClient(ctx, id)
}

func dbNotificationClientToNotificationClient(dbNotificationClient db.NotificationClient) (*notifications.Client, error) {
	notificationClientID, err := parseID(dbNotificationClient.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid notification client ID: %w", err)
	}

	return &notifications.Client{
		ID:   notificationClientID,
		Name: dbNotificationClient.Name,
		URL:  dbNotificationClient.URL,
	}, nil
}
