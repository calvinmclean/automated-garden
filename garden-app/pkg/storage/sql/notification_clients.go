package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage/sql/db"
	"github.com/calvinmclean/babyapi"
)

// NotificationClientStorage implements babyapi.Storage interface for NotificationClient using SQL
type NotificationClientStorage struct {
	q *db.Queries
}

var _ babyapi.Storage[*notifications.Client] = &NotificationClientStorage{}

// NewNotificationClientStorage creates a new NotificationClientStorage instance
func NewNotificationClientStorage(sqlDB *sql.DB) *NotificationClientStorage {
	return &NotificationClientStorage{
		q: db.New(sqlDB),
	}
}

// Get retrieves a NotificationClient from storage by ID
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

// Search returns all NotificationClients from storage
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

// Set saves a NotificationClient to storage (creates or updates)
func (s *NotificationClientStorage) Set(ctx context.Context, notificationClient *notifications.Client) error {
	options, err := json.Marshal(notificationClient.Options)
	if err != nil {
		return fmt.Errorf("error marshaling options: %w", err)
	}

	return s.q.UpsertNotificationClient(ctx, db.UpsertNotificationClientParams{
		ID:      notificationClient.ID.String(),
		Name:    notificationClient.Name,
		Type:    notificationClient.Type,
		Options: options,
	})
}

// Delete removes a NotificationClient from storage
func (s *NotificationClientStorage) Delete(ctx context.Context, id string) error {
	return s.q.DeleteNotificationClient(ctx, id)
}

func dbNotificationClientToNotificationClient(dbNotificationClient db.NotificationClient) (*notifications.Client, error) {
	notificationClientID, err := parseID(dbNotificationClient.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid notification client ID: %w", err)
	}

	notificationClient := &notifications.Client{
		ID:   notificationClientID,
		Name: dbNotificationClient.Name,
		Type: dbNotificationClient.Type,
	}

	if len(dbNotificationClient.Options) > 0 {
		var options map[string]any
		err := json.Unmarshal(dbNotificationClient.Options, &options)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling options: %w", err)
		}
		notificationClient.Options = options
	}

	return notificationClient, nil
}
