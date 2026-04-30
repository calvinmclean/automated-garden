package storage

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
	defer func() { _ = db.Close() }()

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

func TestWeatherScalerMigration(t *testing.T) {
	db, err := sql.Open("sqlite", "file:weatherMigrateTest?mode=memory&cache=shared")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Create water_schedules table with schema as it was before migration 5
	_, err = db.Exec(`
		CREATE TABLE water_schedules (
			id VARCHAR(20) PRIMARY KEY,
			name TEXT,
			description TEXT,
			duration INT NOT NULL,
			interval INT NOT NULL,
			start_date DATETIME NOT NULL,
			start_time VARCHAR(14) NOT NULL,
			end_date DATETIME,
			active_period_start_month TEXT,
			active_period_end_month TEXT,
			weather_control TEXT,
			notification_client_id VARCHAR(20)
		)
	`)
	require.NoError(t, err)

	// Create weather_clients table (schema before migration 2 adds 'name' column)
	_, err = db.Exec(`
		CREATE TABLE weather_clients (
			id VARCHAR(20) PRIMARY KEY,
			type TEXT NOT NULL,
			options JSON NOT NULL
		)
	`)
	require.NoError(t, err)

	// Insert test data with old ScaleControl format
	// Rain control: baseline_value=0, factor=0, range=25.4, client_id=client1
	// Expected: input_min=0, input_max=25.4, factor_min=1.0, factor_max=1.0
	_, err = db.Exec(`
		INSERT INTO water_schedules (id, duration, interval, start_date, start_time, weather_control)
		VALUES ('ws_rain_only', 3600, 86400, '2024-01-01', '08:00:00Z',
			'{"rain_control":{"baseline_value":0,"factor":0,"range":25.4,"client_id":"client1"}}')
	`)
	require.NoError(t, err)

	// Insert test data with old ScaleControl format
	// Temperature control: baseline_value=30, factor=0.5, range=10, client_id=client2
	// Expected: input_min=20, input_max=40, factor_min=0.5, factor_max=1.5
	_, err = db.Exec(`
		INSERT INTO water_schedules (id, duration, interval, start_date, start_time, weather_control)
		VALUES ('ws_temp_only', 3600, 86400, '2024-01-01', '08:00:00Z',
			'{"temperature_control":{"baseline_value":30,"factor":0.5,"range":10,"client_id":"client2"}}')
	`)
	require.NoError(t, err)

	// Insert test data with both controls
	_, err = db.Exec(`
		INSERT INTO water_schedules (id, duration, interval, start_date, start_time, weather_control)
		VALUES ('ws_both', 3600, 86400, '2024-01-01', '08:00:00Z',
			'{"rain_control":{"baseline_value":5,"factor":0.2,"range":20,"client_id":"client1"},"temperature_control":{"baseline_value":25,"factor":0.3,"range":8,"client_id":"client2"}}')
	`)
	require.NoError(t, err)

	// Insert test data with NULL weather_control
	_, err = db.Exec(`
		INSERT INTO water_schedules (id, duration, interval, start_date, start_time, weather_control)
		VALUES ('ws_no_weather', 3600, 86400, '2024-01-01', '08:00:00Z', NULL)
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

	// Verify rain-only conversion
	var weatherControl string
	err = db.QueryRow("SELECT weather_control FROM water_schedules WHERE id = 'ws_rain_only'").Scan(&weatherControl)
	require.NoError(t, err)
	assert.Contains(t, weatherControl, `"interpolation":"linear"`)
	assert.Contains(t, weatherControl, `"input_min":0`)
	assert.Contains(t, weatherControl, `"input_max":25.4`)
	assert.Contains(t, weatherControl, `"factor_min":1`) // 1.0 - 0.0 = 1.0
	assert.Contains(t, weatherControl, `"factor_max":1`)
	assert.Contains(t, weatherControl, `"client_id":"client1"`)
	// Old fields should not exist
	assert.NotContains(t, weatherControl, `"baseline_value"`)
	assert.NotContains(t, weatherControl, `"factor":`)
	assert.NotContains(t, weatherControl, `"range":`)

	// Verify temperature-only conversion
	err = db.QueryRow("SELECT weather_control FROM water_schedules WHERE id = 'ws_temp_only'").Scan(&weatherControl)
	require.NoError(t, err)
	assert.Contains(t, weatherControl, `"interpolation":"linear"`)
	assert.Contains(t, weatherControl, `"input_min":20`)   // 30 - 10 = 20
	assert.Contains(t, weatherControl, `"input_max":40`)   // 30 + 10 = 40
	assert.Contains(t, weatherControl, `"factor_min":0.5`) // 1.0 - 0.5 = 0.5
	assert.Contains(t, weatherControl, `"factor_max":1.5`) // 1.0 + 0.5 = 1.5
	assert.Contains(t, weatherControl, `"client_id":"client2"`)

	// Verify both controls conversion
	err = db.QueryRow("SELECT weather_control FROM water_schedules WHERE id = 'ws_both'").Scan(&weatherControl)
	require.NoError(t, err)
	// Rain: baseline=5, range=20 -> input_min=5, input_max=25, factor_min=0.8, factor_max=1.0
	assert.Contains(t, weatherControl, `"input_min":5`)
	assert.Contains(t, weatherControl, `"input_max":25`)
	assert.Contains(t, weatherControl, `"factor_min":0.8`) // 1.0 - 0.2 = 0.8
	// Temp: baseline=25, range=8 -> input_min=17, input_max=33, factor_min=0.7, factor_max=1.3
	assert.Contains(t, weatherControl, `"input_min":17`)   // 25 - 8 = 17
	assert.Contains(t, weatherControl, `"input_max":33`)   // 25 + 8 = 33
	assert.Contains(t, weatherControl, `"factor_min":0.7`) // 1.0 - 0.3 = 0.7
	assert.Contains(t, weatherControl, `"factor_max":1.3`) // 1.0 + 0.3 = 1.3

	// Verify NULL weather_control remains NULL
	var nullControl sql.NullString
	err = db.QueryRow("SELECT weather_control FROM water_schedules WHERE id = 'ws_no_weather'").Scan(&nullControl)
	require.NoError(t, err)
	assert.False(t, nullControl.Valid)

	// Test down migration - step back just one migration
	err = m.Steps(-1)
	require.NoError(t, err)

	// Verify rain-only reversion
	err = db.QueryRow("SELECT weather_control FROM water_schedules WHERE id = 'ws_rain_only'").Scan(&weatherControl)
	require.NoError(t, err)
	assert.Contains(t, weatherControl, `"baseline_value":0`)
	assert.Contains(t, weatherControl, `"range":25.4`)
	assert.Contains(t, weatherControl, `"factor":0`) // 1.0 - 1.0 = 0
	assert.Contains(t, weatherControl, `"client_id":"client1"`)

	// Verify temperature-only reversion
	err = db.QueryRow("SELECT weather_control FROM water_schedules WHERE id = 'ws_temp_only'").Scan(&weatherControl)
	require.NoError(t, err)
	assert.Contains(t, weatherControl, `"baseline_value":30`) // (20 + 40) / 2 = 30
	assert.Contains(t, weatherControl, `"range":10`)          // (40 - 20) / 2 = 10
	assert.Contains(t, weatherControl, `"factor":0.5`)        // 1.5 - 1.0 = 0.5
	assert.Contains(t, weatherControl, `"client_id":"client2"`)
}

func TestNotificationClientStorage(t *testing.T) {
	ctx := context.Background()

	sqlClient, err := NewClient(Config{
		ConnectionString: ":memory:",
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
