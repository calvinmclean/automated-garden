package sql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/babyapi"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationClientURLMigration(t *testing.T) {
	db, err := sql.Open("sqlite", "file:migrateTest?mode=memory&cache=shared")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE notification_clients (
			id VARCHAR(20) PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			options JSON NOT NULL
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO notification_clients (id, name, type, options)
		VALUES ('pushover_client_01', 'My Pushover', 'pushover', '{"app_token":"myapptoken123","recipient_token":"myusertoken456"}')
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO notification_clients (id, name, type, options)
		VALUES ('fake_client_01', 'My Fake', 'fake', '{}')
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO notification_clients (id, name, type, options)
		VALUES ('pushover_incomplete', 'Incomplete Pushover', 'pushover', '{"app_token":"onlytoken"}')
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO notification_clients (id, name, type, options)
		VALUES ('other_type_client', 'Other', 'discord', '{"webhook":"https://example.com"}')
	`)
	require.NoError(t, err)

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	require.NoError(t, err)

	migrations, err := iofs.New(migrationsFS, "migrations")
	require.NoError(t, err)

	m, err := migrate.NewWithInstance("iofs", migrations, "sqlite3", driver)
	require.NoError(t, err)

	err = m.Up()
	require.NoError(t, err)

	var url string
	var name string

	err = db.QueryRow("SELECT name, url FROM notification_clients WHERE id = 'pushover_client_01'").Scan(&name, &url)
	require.NoError(t, err)
	assert.Equal(t, "My Pushover", name)
	assert.Equal(t, "pushover://shoutrrr:myapptoken123@myusertoken456/", url)

	err = db.QueryRow("SELECT name, url FROM notification_clients WHERE id = 'fake_client_01'").Scan(&name, &url)
	require.NoError(t, err)
	assert.Equal(t, "My Fake", name)
	assert.Equal(t, "fake://", url)

	err = db.QueryRow("SELECT name, url FROM notification_clients WHERE id = 'pushover_incomplete'").Scan(&name, &url)
	require.NoError(t, err)
	assert.Equal(t, "Incomplete Pushover", name)
	assert.Equal(t, "", url)

	err = db.QueryRow("SELECT name, url FROM notification_clients WHERE id = 'other_type_client'").Scan(&name, &url)
	require.NoError(t, err)
	assert.Equal(t, "Other", name)
	assert.Equal(t, "", url)

	var colCount int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('notification_clients') WHERE name IN ('type', 'options')").Scan(&colCount)
	require.NoError(t, err)
	assert.Equal(t, 0, colCount, "type and options columns should be dropped")
}

func TestNotificationClientStorage(t *testing.T) {
	ctx := context.Background()

	sqlClient, err := NewClient(Config{
		DataSourceName: ":memory:",
	})
	require.NoError(t, err)

	nc := &notifications.Client{
		ID:   babyapi.NewID(),
		Name: "TestClient",
		URL:  "pushover://shoutrrr:testtoken@testuser/",
	}

	err = sqlClient.NotificationClientConfigs.Set(ctx, nc)
	require.NoError(t, err)

	got, err := sqlClient.NotificationClientConfigs.Get(ctx, nc.GetID())
	require.NoError(t, err)
	assert.Equal(t, nc.ID, got.ID)
	assert.Equal(t, "TestClient", got.Name)
	assert.Equal(t, "pushover://shoutrrr:testtoken@testuser/", got.URL)

	all, err := sqlClient.NotificationClientConfigs.Search(ctx, "", nil)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	err = sqlClient.NotificationClientConfigs.Delete(ctx, nc.GetID())
	require.NoError(t, err)

	_, err = sqlClient.NotificationClientConfigs.Get(ctx, nc.GetID())
	assert.Error(t, err)
}
