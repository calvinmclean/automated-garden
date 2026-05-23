package worker

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/notifications/fake"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/weather"
	"github.com/calvinmclean/babyapi"
	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	id, _  = xid.FromString("c5cvhpcbcv45e8bp16dg")
	id2, _ = xid.FromString("chkodpg3lcj13q82mq40")
)

func createExampleGarden() *pkg.Garden {
	two := uint(2)
	createdAt, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	startTime, _ := pkg.StartTimeFromString("22:00:01-07:00")
	return &pkg.Garden{
		Name:        "test-garden",
		TopicPrefix: "test-garden",
		MaxZones:    &two,
		ID:          babyapi.ID{ID: id},
		CreatedAt:   &createdAt,
		LightSchedule: &pkg.LightSchedule{
			Duration:  &pkg.Duration{Duration: 15 * time.Hour},
			StartTime: startTime,
		},
	}
}

func createExampleZone() *pkg.Zone {
	createdAt, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	p := uint(0)
	return &pkg.Zone{
		Name:             "test zone",
		ID:               babyapi.ID{ID: id},
		CreatedAt:        &createdAt,
		Position:         &p,
		WaterScheduleIDs: []xid.ID{id},
		GardenID:         id,
	}
}

func createExampleWaterSchedule() *pkg.WaterSchedule {
	createdAt, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	startDate := pkg.NewDate(createdAt)
	return &pkg.WaterSchedule{
		ID:        babyapi.ID{ID: id},
		Duration:  &pkg.Duration{Duration: time.Second},
		Interval:  &pkg.Duration{Duration: time.Hour * 24},
		StartTime: pkg.NewStartTime(createdAt),
		StartDate: &startDate,
	}
}

func TestScheduleWaterActionStorageError(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	assert.NoError(t, err)

	garden := createExampleGarden()
	zone := createExampleZone()

	err = storageClient.Gardens.Set(context.Background(), garden)
	assert.NoError(t, err)

	err = storageClient.Zones.Set(context.Background(), zone)
	assert.NoError(t, err)

	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)

	// Allow light action sync during StartAsync (test doesn't care about light behavior)
	mqttClient.On("Publish", "test-garden/command/light", mock.Anything).Return(nil)
	mqttClient.On("Disconnect", uint(100)).Return()
	influxdbClient.On("Close").Return()

	worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
	worker.StartAsync()

	ws := createExampleWaterSchedule()

	// Set StartTime to the near future
	startTime := pkg.NewStartTime(clock.Now().Add(250 * time.Millisecond))
	ws.StartTime = startTime

	err = worker.ScheduleWaterAction(ws)
	assert.NoError(t, err)

	time.Sleep(1000 * time.Millisecond)

	worker.Stop()
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}

func TestScheduleWaterAction(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	assert.NoError(t, err)
	defer weather.ResetCache()

	clock.Reset()
	defer clock.Reset()

	garden := createExampleGarden()
	zone := createExampleZone()

	err = storageClient.Gardens.Set(context.Background(), garden)
	assert.NoError(t, err)

	err = storageClient.Zones.Set(context.Background(), zone)
	assert.NoError(t, err)

	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)

	// Allow light action sync during StartAsync (test doesn't care about light behavior)
	mqttClient.On("Publish", "test-garden/command/light", mock.Anything).Return(nil)
	mqttClient.On("Publish", "test-garden/command/water", mock.Anything).Return(nil)
	mqttClient.On("Disconnect", uint(100)).Return()
	influxdbClient.On("Close").Return()

	worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
	worker.StartAsync()

	ws := createExampleWaterSchedule()

	wsNotInStorage := createExampleWaterSchedule()
	wsNotInStorage.ID = babyapi.ID{ID: id2}

	wsNotActive := createExampleWaterSchedule()
	wsNotActive.ID = babyapi.NewID()
	currentTime := clock.Now()
	wsNotActive.ActivePeriod = &pkg.ActivePeriod{
		StartMonth: currentTime.AddDate(0, 1, 0).String(),
		EndMonth:   currentTime.AddDate(0, 2, 0).String(),
	}

	// Set StartTime to the near future
	startTime := pkg.NewStartTime(clock.Now().Add(1 * time.Second))
	ws.StartTime = startTime
	wsNotInStorage.StartTime = startTime
	wsNotActive.StartTime = startTime

	err = storageClient.WaterSchedules.Set(context.Background(), ws)
	assert.NoError(t, err)

	err = storageClient.WaterSchedules.Set(context.Background(), wsNotActive)
	assert.NoError(t, err)

	err = worker.ScheduleWaterAction(ws)
	assert.NoError(t, err)

	err = worker.ScheduleWaterAction(wsNotInStorage)
	assert.NoError(t, err)

	err = worker.ScheduleWaterAction(wsNotActive)
	assert.NoError(t, err)

	time.Sleep(1000 * time.Millisecond)

	worker.Stop()
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}

func TestScheduleWaterActionGardenHealthNotification(t *testing.T) {
	tests := []struct {
		name             string
		lastContact      time.Time
		expectedMessages []fake.Message
	}{
		{
			"GardenUp",
			clock.Now(),
			[]fake.Message{},
		},
		{
			"GardenDown",
			clock.Now().Add(-10 * time.Minute),
			[]fake.Message{
				{
					Title:   "test-garden: DOWN",
					Message: `Attempting to execute Water Action, but last contact was \d\d\d\d-\d\d-\d\d \d\d:\d\d:\d\d.\nDetails: last contact from Garden was 10m\d.\d+s ago`,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake.Reset()

			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)
			defer weather.ResetCache()

			garden := createExampleGarden()
			zone := createExampleZone()

			notificationClient := &notifications.Client{
				ID:   babyapi.NewID(),
				Name: "TestClient",
				URL:  "fake://",
			}
			err = storageClient.NotificationClientConfigs.Set(context.Background(), notificationClient)
			assert.NoError(t, err)

			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			err = storageClient.Zones.Set(context.Background(), zone)
			assert.NoError(t, err)

			mqttClient := new(mqtt.MockClient)
			// Allow light action sync during StartAsync (test doesn't care about light behavior)
			mqttClient.On("Publish", "test-garden/command/light", mock.Anything).Return(nil)
			mqttClient.On("Publish", "test-garden/command/water", mock.Anything).Return(nil)
			mqttClient.On("Disconnect", uint(100)).Return()

			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(tt.lastContact, nil)
			influxdbClient.On("Close").Return()

			worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
			worker.StartAsync()

			ws := createExampleWaterSchedule()
			ws.Name = "MyWaterSchedule"
			// Set StartTime to the near future
			ws.StartTime = pkg.NewStartTime(clock.Now().Add(1 * time.Second))
			ncID := notificationClient.GetID()
			ws.NotificationClientID = &ncID

			err = storageClient.WaterSchedules.Set(context.Background(), ws)
			assert.NoError(t, err)

			err = worker.ScheduleWaterAction(ws)
			assert.NoError(t, err)

			time.Sleep(1000 * time.Millisecond)

			worker.Stop()
			influxdbClient.AssertExpectations(t)
			mqttClient.AssertExpectations(t)

			for i, msg := range fake.Messages() {
				require.Equal(t, tt.expectedMessages[i].Title, msg.Title)
				require.Regexp(t, tt.expectedMessages[i].Message, msg.Message)
			}
			if len(tt.expectedMessages) == 0 {
				require.Empty(t, fake.Messages())
			}
		})
	}
}

func TestScheduleWaterActionWithErrorNotification(t *testing.T) {
	tests := []struct {
		name               string
		enableNotification bool
		expectedTitle      string
	}{
		{"NotificationsEnabled", true, "MyWaterSchedule: Water Action Error"},
		{"NotificationsDisabled", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake.Reset()

			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)
			defer weather.ResetCache()

			garden := createExampleGarden()
			zone := createExampleZone()

			notificationClient := &notifications.Client{
				ID:   babyapi.NewID(),
				Name: "TestClient",
				URL:  "fake://",
			}
			err = storageClient.NotificationClientConfigs.Set(context.Background(), notificationClient)
			assert.NoError(t, err)

			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			err = storageClient.Zones.Set(context.Background(), zone)
			assert.NoError(t, err)

			mqttClient := new(mqtt.MockClient)
			// Allow light action sync during StartAsync (test doesn't care about light behavior)
			mqttClient.On("Publish", "test-garden/command/light", mock.Anything).Return(nil)
			mqttClient.On("Publish", "test-garden/command/water", mock.Anything).Return(errors.New("publish error"))
			mqttClient.On("Disconnect", uint(100)).Return()

			influxdbClient := new(influxdb.MockClient)
			if tt.enableNotification {
				influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(clock.Now(), nil)
			}
			influxdbClient.On("Close").Return()

			worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
			worker.StartAsync()

			ws := createExampleWaterSchedule()
			ws.Name = "MyWaterSchedule"
			// Set StartTime to the near future
			ws.StartTime = pkg.NewStartTime(clock.Now().Add(1 * time.Second))
			if tt.enableNotification {
				ncID := notificationClient.GetID()
				ws.NotificationClientID = &ncID
			}

			err = storageClient.WaterSchedules.Set(context.Background(), ws)
			assert.NoError(t, err)

			err = worker.ScheduleWaterAction(ws)
			assert.NoError(t, err)

			time.Sleep(1000 * time.Millisecond)

			worker.Stop()
			influxdbClient.AssertExpectations(t)
			mqttClient.AssertExpectations(t)

			assert.Equal(t, tt.expectedTitle, fake.LastMessage().Title)
		})
	}
}

func TestResetNextWaterTime(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	assert.NoError(t, err)
	defer weather.ResetCache()

	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)
	mqttClient.On("Disconnect", uint(100)).Return()
	influxdbClient.On("Close").Return()

	worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
	worker.StartAsync()

	ws := createExampleWaterSchedule()
	// Set Zone's WaterSchedule.StartTime to a time that won't cause it to run
	startTime := pkg.NewStartTime(clock.Now().Add(-1 * time.Hour))
	ws.StartTime = startTime
	err = worker.ScheduleWaterAction(ws)
	assert.NoError(t, err)

	// Change WaterSchedule and restart
	newTime := pkg.NewStartTime(startTime.Time.Add(-30 * time.Minute))
	ws.StartTime = newTime
	err = worker.ResetWaterSchedule(ws)
	assert.NoError(t, err)

	nextWaterTime := worker.GetNextWaterTime(ws).In(startTime.Time.Location())
	expected := startTime.Time.Add(-30 * time.Minute).Add(24 * time.Hour).Truncate(time.Second)
	assert.Equal(t, expected, nextWaterTime)

	worker.Stop()
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}

func TestGetNextWaterTime(t *testing.T) {
	mockClock := clock.MockTime()
	now := mockClock.Now()
	t.Cleanup(clock.Reset)

	tests := []struct {
		name      string
		startTime time.Time
		interval  time.Duration
		expected  time.Time
	}{
		{
			"RunTomorrow",
			// Set start time to before now so it runs at next interval
			now.Add(-1 * time.Hour),
			24 * time.Hour,
			now.Add(23 * time.Hour).Truncate(time.Second),
		},
		{
			"RunIn5Days",
			// Set start time to before now so it runs at next interval
			now.Add(-1 * time.Hour),
			5 * 24 * time.Hour,
			now.Add(5*24*time.Hour - 1*time.Hour).Truncate(time.Second),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)
			defer weather.ResetCache()

			influxdbClient := new(influxdb.MockClient)
			mqttClient := new(mqtt.MockClient)
			mqttClient.On("Disconnect", uint(100)).Return()
			influxdbClient.On("Close").Return()

			worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
			worker.StartAsync()

			ws := createExampleWaterSchedule()
			ws.StartTime = pkg.NewStartTime(tt.startTime)
			startDate := pkg.NewDate(now)
			ws.StartDate = &startDate
			ws.Interval = &pkg.Duration{Duration: tt.interval}

			err = worker.ScheduleWaterAction(ws)
			assert.NoError(t, err)

			nextWaterTime := worker.GetNextWaterTime(ws).In(tt.startTime.Location())
			assert.Equal(t, tt.expected, nextWaterTime)

			worker.Stop()
			influxdbClient.AssertExpectations(t)
			mqttClient.AssertExpectations(t)
		})
	}
}

func TestGetNextWaterTimeWithInterval(t *testing.T) {
	tests := []struct {
		name            string
		startDateOffset time.Duration
		interval        time.Duration
		expectedOffset  time.Duration
	}{
		{
			"RunTomorrow",
			0, // Start today so first run is now
			24 * time.Hour,
			24 * time.Hour,
		},
		{
			"RunIn5Days",
			0, // Start today so first run is now
			5 * 24 * time.Hour,
			5 * 24 * time.Hour, // First run is now, next is 5 days later
		},
		{
			// This tests the scenario where the server is restarted in-between an interval and
			// relies on persistent state to reschedule
			"Every5DaysButStartedAFewDaysAgo",
			-3 * 24 * time.Hour, // Started 3 days ago
			5 * 24 * time.Hour,
			2 * 24 * time.Hour, // Next run is in 2 days (5 - 3 = 2)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			assert.NoError(t, err)
			defer weather.ResetCache()

			influxdbClient := new(influxdb.MockClient)
			mqttClient := new(mqtt.MockClient)
			mqttClient.On("Disconnect", uint(100)).Return()
			influxdbClient.On("Close").Return()

			worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
			worker.StartAsync()

			// Set time to near future so it can execute and we can see the next interval
			delay := 100 * time.Millisecond
			now := clock.Now()
			startTime := now.Add(delay)

			ws := createExampleWaterSchedule()
			ws.StartTime = pkg.NewStartTime(startTime)
			// Use local time with offset, NewDate will handle UTC conversion internally
			startDate := pkg.NewDate(now.Add(tt.startDateOffset))
			ws.StartDate = &startDate
			ws.Interval = &pkg.Duration{Duration: tt.interval}

			err = storageClient.WaterSchedules.Set(context.Background(), ws)
			assert.NoError(t, err)

			err = worker.ScheduleWaterAction(ws)
			assert.NoError(t, err)

			// Wait for first execution so we can be sure the interval works
			time.Sleep(delay + 100*time.Millisecond)

			expected := startTime.Add(tt.expectedOffset).Truncate(time.Second)

			nextWaterTime := worker.GetNextWaterTime(ws).In(startTime.Location())
			assert.Equal(t, expected, nextWaterTime)

			worker.Stop()
			influxdbClient.AssertExpectations(t)
			mqttClient.AssertExpectations(t)
		})
	}
}

func TestScheduleLightActions(t *testing.T) {
	t.Run("ScheduledLightActionCreatesNotification", func(t *testing.T) {
		tests := []struct {
			name               string
			url                string
			off                bool
			enableNotification bool
			mqttPublishError   error
			expectedOnMessage  string
			expectedOffMessage string
		}{
			{
				"SuccessfulOnAndOff",
				"fake://",
				true,
				true,
				nil,
				"test-garden: Light ON",
				"test-garden: Light OFF",
			},
			{
				"NoNotificationWhenDisabledSuccessfulOnAndOff",
				"fake://",
				true,
				false,
				nil,
				"",
				"",
			},
			{
				"ErrorCreatingClient",
				"fake://?create_error=error",
				false,
				true,
				nil,
				"",
				"",
			},
			{
				"ErrorSendingMessage",
				"fake://?send_message_error=error",
				false,
				true,
				nil,
				"",
				"",
			},
			{
				"ErrorNotification",
				"fake://",
				false,
				true,
				errors.New("publish error"),
				"test-garden: Light Action Error",
				"",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fake.Reset()

				storageClient, err := storage.NewClient(storage.Config{
					ConnectionString: ":memory:",
				})
				assert.NoError(t, err)

				mqttClient := new(mqtt.MockClient)
				mqttClient.On("Publish", "test-garden/command/light", mock.Anything).Return(tt.mqttPublishError)
				mqttClient.On("Disconnect", uint(100)).Return()

				influxdbClient := new(influxdb.MockClient)
				influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(clock.Now(), nil)
				influxdbClient.On("Close").Return()

				notificationClient := &notifications.Client{
					ID:   babyapi.NewID(),
					Name: "TestClient",
					URL:  tt.url,
				}
				err = storageClient.NotificationClientConfigs.Set(context.Background(), notificationClient)
				assert.NoError(t, err)

				worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
				worker.StartAsync()
				defer worker.Stop()

				// Create new LightSchedule that turns on in 1 second for only 1 second
				now := clock.Now().UTC()
				later := now.Add(1 * time.Second).Truncate(time.Second)
				g := createExampleGarden()
				g.LightSchedule.StartTime = pkg.NewStartTime(later)
				g.LightSchedule.Duration = &pkg.Duration{Duration: time.Second}
				if tt.enableNotification {
					ncID := notificationClient.GetID()
					g.NotificationClientID = &ncID
					g.NotificationSettings = &pkg.NotificationSettings{
						LightSchedule: true,
					}
				}

				err = worker.ScheduleLightActions(g)
				assert.NoError(t, err)

				time.Sleep(1 * time.Second)
				assert.Equal(t, tt.expectedOnMessage, fake.LastMessage().Title)

				if tt.off {
					time.Sleep(1 * time.Second)
					assert.Equal(t, tt.expectedOffMessage, fake.LastMessage().Title)
				}
			})
		}
	})

	t.Run("ScheduledLightActionGardenDownNotification", func(t *testing.T) {
		tests := []struct {
			name             string
			lastContact      time.Time
			expectedMessages []fake.Message
		}{
			{
				"GardenUp",
				clock.Now(),
				[]fake.Message{
					{Title: "test-garden: Light ON", Message: "Successfully executed LightAction"},
				},
			},
			{
				"GardenDown",
				clock.Now().Add(-10 * time.Minute),
				[]fake.Message{
					{
						Title:   "test-garden: DOWN",
						Message: `Attempting to execute Light Action, but last contact was \d\d\d\d-\d\d-\d\d \d\d:\d\d:\d\d.\nDetails: last contact from Garden was 10m\d.\d+s ago`,
					},
					{Title: "test-garden: Light ON", Message: "Successfully executed LightAction"},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fake.Reset()

				storageClient, err := storage.NewClient(storage.Config{
					ConnectionString: ":memory:",
				})
				assert.NoError(t, err)

				mqttClient := new(mqtt.MockClient)
				mqttClient.On("Publish", "test-garden/command/light", mock.Anything).Return(nil)
				mqttClient.On("Disconnect", uint(100)).Return()

				influxdbClient := new(influxdb.MockClient)
				influxdbClient.On("GetLastContact", mock.Anything, mock.Anything).Return(tt.lastContact, nil)
				influxdbClient.On("Close").Return()

				notificationClient := &notifications.Client{
					ID:   babyapi.NewID(),
					Name: "TestClient",
					URL:  "fake://",
				}
				err = storageClient.NotificationClientConfigs.Set(context.Background(), notificationClient)
				assert.NoError(t, err)

				worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
				worker.StartAsync()
				defer worker.Stop()

				// Create new LightSchedule that turns on in 1 second for only 1 second
				now := clock.Now().UTC()
				later := now.Add(1 * time.Second).Truncate(time.Second)
				g := createExampleGarden()
				g.LightSchedule.StartTime = pkg.NewStartTime(later)
				g.LightSchedule.Duration = &pkg.Duration{Duration: time.Second}
				ncID := notificationClient.GetID()
				g.NotificationClientID = &ncID

				err = worker.ScheduleLightActions(g)
				assert.NoError(t, err)

				time.Sleep(1 * time.Second)
				for i, msg := range fake.Messages() {
					require.Equal(t, tt.expectedMessages[i].Title, msg.Title)
					require.Regexp(t, tt.expectedMessages[i].Message, msg.Message)
				}
			})
		}
	})
}

func TestRemoveJobsByID(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	assert.NoError(t, err)
	defer weather.ResetCache()

	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)
	mqttClient.On("Disconnect", uint(100)).Return()
	influxdbClient.On("Close").Return()

	worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
	worker.StartAsync()

	ws := createExampleWaterSchedule()
	// Set Zone's WaterSchedule.StartTime to a time that won't cause it to run
	startTime := pkg.NewStartTime(clock.Now().Add(-1 * time.Hour))
	ws.StartTime = startTime
	err = worker.ScheduleWaterAction(ws)
	assert.NoError(t, err)

	err = worker.RemoveJobsByID(ws.ID.String())
	assert.NoError(t, err)

	// This also gets coverage for GetNextWaterTime when no Job exists
	nextWaterTime := worker.GetNextWaterTime(ws)
	if nextWaterTime != nil {
		t.Errorf("Expected nil but got: %v", nextWaterTime)
	}

	worker.Stop()
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}

func TestGetNextWaterScheduleWithMultiple(t *testing.T) {
	worker := NewWorker(nil, nil, nil, slog.Default())
	worker.scheduler.StartAsync()

	now := clock.Now()
	addTime := func(add time.Duration) *pkg.StartTime {
		return pkg.NewStartTime(now.Add(add))
	}

	ws1 := &pkg.WaterSchedule{
		Name:      "ws1",
		ID:        babyapi.NewID(),
		Duration:  &pkg.Duration{Duration: time.Second},
		Interval:  &pkg.Duration{Duration: 24 * time.Hour},
		StartTime: addTime(5 * time.Minute),
	}
	ws2 := &pkg.WaterSchedule{
		Name:      "ws2",
		ID:        babyapi.NewID(),
		Duration:  &pkg.Duration{Duration: time.Second},
		Interval:  &pkg.Duration{Duration: 24 * time.Hour},
		StartTime: addTime(1 * time.Minute),
	}
	ws3 := &pkg.WaterSchedule{
		Name:      "ws3",
		ID:        babyapi.NewID(),
		Duration:  &pkg.Duration{Duration: time.Second},
		Interval:  &pkg.Duration{Duration: 24 * time.Hour},
		StartTime: addTime(3 * time.Minute),
	}
	ws4 := &pkg.WaterSchedule{
		Name:      "ws4",
		ID:        babyapi.NewID(),
		Duration:  &pkg.Duration{Duration: time.Second},
		Interval:  &pkg.Duration{Duration: 24 * time.Hour},
		StartTime: addTime(2 * time.Minute),
	}
	unscheduled := &pkg.WaterSchedule{
		Name:      "unscheduled",
		ID:        babyapi.NewID(),
		Duration:  &pkg.Duration{Duration: time.Second},
		Interval:  &pkg.Duration{Duration: 24 * time.Hour},
		StartTime: addTime(2 * time.Minute),
	}
	inactive := &pkg.WaterSchedule{
		Name:      "inactive",
		ID:        babyapi.NewID(),
		Duration:  &pkg.Duration{Duration: time.Second},
		Interval:  &pkg.Duration{Duration: 24 * time.Hour},
		StartTime: addTime(2 * time.Minute),
		ActivePeriod: &pkg.ActivePeriod{
			StartMonth: now.AddDate(0, 1, 0).Month().String(),
			EndMonth:   now.AddDate(0, 2, 0).Month().String(),
		},
	}

	err := worker.ScheduleWaterAction(ws1)
	assert.NoError(t, err)
	err = worker.ScheduleWaterAction(ws2)
	assert.NoError(t, err)
	err = worker.ScheduleWaterAction(ws3)
	assert.NoError(t, err)
	err = worker.ScheduleWaterAction(ws4)
	assert.NoError(t, err)
	err = worker.ScheduleWaterAction(inactive)
	assert.NoError(t, err)

	next := worker.GetNextActiveWaterSchedule([]*pkg.WaterSchedule{ws1, ws2, ws3, ws4, unscheduled, inactive})
	assert.Equal(t, "ws2", next.Name)

	next = worker.GetNextActiveWaterSchedule([]*pkg.WaterSchedule{})
	assert.Nil(t, next)
}

// TestScheduleLightActions_NextRun tests that gocron NextRun values are computed
// correctly for different mock times relative to the LightSchedule's ON/OFF times.
// This documents gocron's behavior where StartAt in the past is skipped to the next interval.
func TestScheduleLightActions_NextRun(t *testing.T) {
	// Use a schedule similar to the user's: 19:00 UTC-7 ON, 14h duration
	// ON  = 02:00 UTC, OFF = 16:00 UTC
	tz := time.FixedZone("UTC-7", -7*60*60)

	tests := []struct {
		name            string
		now             time.Time
		expectedOnNext  time.Time
		expectedOffNext time.Time
	}{
		{
			name:            "BeforeOnTime",
			now:             time.Date(2026, 5, 22, 1, 0, 0, 0, time.UTC),
			expectedOnNext:  time.Date(2026, 5, 22, 2, 0, 0, 0, time.UTC),
			expectedOffNext: time.Date(2026, 5, 22, 16, 0, 0, 0, time.UTC),
		},
		{
			name:            "DuringOnPeriod",
			now:             time.Date(2026, 5, 22, 3, 0, 0, 0, time.UTC),
			expectedOnNext:  time.Date(2026, 5, 23, 2, 0, 0, 0, time.UTC),
			expectedOffNext: time.Date(2026, 5, 22, 16, 0, 0, 0, time.UTC),
		},
		{
			name:            "AfterOffTime",
			now:             time.Date(2026, 5, 22, 17, 0, 0, 0, time.UTC),
			expectedOnNext:  time.Date(2026, 5, 23, 2, 0, 0, 0, time.UTC),
			expectedOffNext: time.Date(2026, 5, 23, 16, 0, 0, 0, time.UTC),
		},
		{
			name:            "AtExactOnTime",
			now:             time.Date(2026, 5, 22, 2, 0, 0, 0, time.UTC),
			expectedOnNext:  time.Date(2026, 5, 23, 2, 0, 0, 0, time.UTC),
			expectedOffNext: time.Date(2026, 5, 22, 16, 0, 0, 0, time.UTC),
		},
		{
			name:            "AtExactOffTime",
			now:             time.Date(2026, 5, 22, 16, 0, 0, 0, time.UTC),
			expectedOnNext:  time.Date(2026, 5, 23, 2, 0, 0, 0, time.UTC),
			expectedOffNext: time.Date(2026, 5, 23, 16, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClock := clock.MockTime()
			mockClock.Set(tt.now)
			t.Cleanup(clock.Reset)

			storageClient, err := storage.NewClient(storage.Config{
				ConnectionString: ":memory:",
			})
			require.NoError(t, err)
			defer weather.ResetCache()

			influxdbClient := new(influxdb.MockClient)
			mqttClient := new(mqtt.MockClient)
			mqttClient.On("Disconnect", uint(100)).Return()
			influxdbClient.On("Close").Return()

			worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
			worker.StartAsync()

			g := &pkg.Garden{
				ID:          babyapi.NewID(),
				Name:        "test-garden",
				TopicPrefix: "test-garden",
				LightSchedule: &pkg.LightSchedule{
					Duration:  &pkg.Duration{Duration: 14 * time.Hour},
					StartTime: &pkg.StartTime{Time: time.Date(0, 0, 0, 19, 0, 0, 0, tz)},
				},
			}

			err = worker.ScheduleLightActions(g)
			require.NoError(t, err)

			// Find jobs by tags
			var onJob, offJob interface{ NextRun() time.Time }
			for _, job := range worker.scheduler.Jobs() {
				tags := job.Tags()
				for _, tag := range tags {
					if tag == g.ID.String() {
						for _, t := range tags {
							if t == pkg.LightStateOn.String() {
								onJob = job
							}
							if t == pkg.LightStateOff.String() {
								offJob = job
							}
						}
					}
				}
			}

			require.NotNil(t, onJob, "ON job not found")
			require.NotNil(t, offJob, "OFF job not found")

			assert.True(t, tt.expectedOnNext.Equal(onJob.NextRun()), "ON NextRun mismatch: expected %v, got %v", tt.expectedOnNext, onJob.NextRun())
			assert.True(t, tt.expectedOffNext.Equal(offJob.NextRun()), "OFF NextRun mismatch: expected %v, got %v", tt.expectedOffNext, offJob.NextRun())

			worker.Stop()
			influxdbClient.AssertExpectations(t)
			mqttClient.AssertExpectations(t)
		})
	}
}

// TestScheduleLightActions_OffInOneMinute tests the exact scenario reported by the user:
// Schedule: 11 PM ON, 12h duration. Now is 10:59 AM.
// Light is currently ON (since 11 PM yesterday). OFF is in 1 minute at 11 AM today.
func TestScheduleLightActions_OffInOneMinute(t *testing.T) {
	mockClock := clock.MockTime()
	// 10:59 AM UTC
	mockClock.Set(time.Date(2026, 5, 22, 10, 59, 0, 0, time.UTC))
	t.Cleanup(clock.Reset)

	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	require.NoError(t, err)
	defer weather.ResetCache()

	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)
	mqttClient.On("Disconnect", uint(100)).Return()
	influxdbClient.On("Close").Return()

	worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
	worker.StartAsync()

	g := &pkg.Garden{
		ID:          babyapi.NewID(),
		Name:        "test-garden",
		TopicPrefix: "test-garden",
		LightSchedule: &pkg.LightSchedule{
			Duration:  &pkg.Duration{Duration: 12 * time.Hour},
			StartTime: &pkg.StartTime{Time: time.Date(0, 0, 0, 23, 0, 0, 0, time.UTC)},
		},
	}

	err = worker.ScheduleLightActions(g)
	require.NoError(t, err)

	// Find jobs by tags
	var onJob, offJob interface{ NextRun() time.Time }
	for _, job := range worker.scheduler.Jobs() {
		tags := job.Tags()
		for _, tag := range tags {
			if tag == g.ID.String() {
				for _, t := range tags {
					if t == pkg.LightStateOn.String() {
						onJob = job
					}
					if t == pkg.LightStateOff.String() {
						offJob = job
					}
				}
			}
		}
	}

	require.NotNil(t, onJob, "ON job not found")
	require.NotNil(t, offJob, "OFF job not found")

	// OFF should run at 11:00 AM today (in 1 minute)
	expectedOffNext := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	assert.True(t, expectedOffNext.Equal(offJob.NextRun()), "OFF NextRun mismatch: expected %v, got %v", expectedOffNext, offJob.NextRun())

	// ON should run at 11:00 PM today (12 hours after OFF)
	expectedOnNext := time.Date(2026, 5, 22, 23, 0, 0, 0, time.UTC)
	assert.True(t, expectedOnNext.Equal(onJob.NextRun()), "ON NextRun mismatch: expected %v, got %v", expectedOnNext, onJob.NextRun())

	worker.Stop()
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}

// TestResetLightSchedule_SyncsStateDuringOnPeriod tests that when ResetLightSchedule is
// called while the light should be ON (but the scheduled ON job is in the past and would be
// skipped by gocron), an immediate ON action is executed to sync the state.
func TestResetLightSchedule_SyncsStateDuringOnPeriod(t *testing.T) {
	mockClock := clock.MockTime()
	// Set to 20:00 UTC-7 (May 21) = 03:00 UTC (May 22), which is during the ON period
	mockClock.Set(time.Date(2026, 5, 22, 3, 0, 0, 0, time.UTC))
	t.Cleanup(clock.Reset)

	storageClient, err := storage.NewClient(storage.Config{
		ConnectionString: ":memory:",
	})
	require.NoError(t, err)
	defer weather.ResetCache()

	tz := time.FixedZone("UTC-7", -7*60*60)
	g := &pkg.Garden{
		ID:          babyapi.NewID(),
		Name:        "test-garden",
		TopicPrefix: "test-garden",
		LightSchedule: &pkg.LightSchedule{
			Duration:  &pkg.Duration{Duration: 14 * time.Hour},
			StartTime: &pkg.StartTime{Time: time.Date(0, 0, 0, 19, 0, 0, 0, tz)},
		},
	}

	mqttClient := new(mqtt.MockClient)
	// Expect immediate ON sync because we're in the ON period
	mqttClient.On("Publish", "test-garden/command/light", []byte(`{"state":"ON"}`)).Return(nil)
	mqttClient.On("Disconnect", uint(100)).Return()

	influxdbClient := new(influxdb.MockClient)
	influxdbClient.On("Close").Return()

	worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
	worker.StartAsync()

	err = worker.ResetLightSchedule(g)
	require.NoError(t, err)

	// Verify ON job was skipped to tomorrow (gocron behavior when StartAt is in the past)
	var onJob interface{ NextRun() time.Time }
	for _, job := range worker.scheduler.Jobs() {
		tags := job.Tags()
		for _, tag := range tags {
			if tag == g.ID.String() {
				for _, t := range tags {
					if t == pkg.LightStateOn.String() {
						onJob = job
					}
				}
			}
		}
	}
	require.NotNil(t, onJob, "ON job not found")
	expectedOnNext := time.Date(2026, 5, 23, 2, 0, 0, 0, time.UTC)
	assert.True(t, expectedOnNext.Equal(onJob.NextRun()), "ON NextRun should be tomorrow: expected %v, got %v", expectedOnNext, onJob.NextRun())

	worker.Stop()
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}
