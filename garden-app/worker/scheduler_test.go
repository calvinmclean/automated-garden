package worker

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
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
	return &pkg.WaterSchedule{
		ID:        babyapi.ID{ID: id},
		Duration:  &pkg.Duration{Duration: time.Second},
		Interval:  &pkg.Duration{Duration: time.Hour * 24},
		StartTime: pkg.NewStartTime(createdAt),
		StartDate: &createdAt,
	}
}

func TestScheduleWaterActionStorageError(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
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

	mqttClient.On("Disconnect", uint(100)).Return()
	influxdbClient.On("Close").Return()

	worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
	worker.StartAsync()

	ws := createExampleWaterSchedule()

	// Set StartTime to the near future
	startTime := pkg.NewStartTime(time.Now().Add(250 * time.Millisecond))
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
		Driver: "hashmap",
	})
	assert.NoError(t, err)
	defer weather.ResetCache()

	garden := createExampleGarden()
	zone := createExampleZone()

	err = storageClient.Gardens.Set(context.Background(), garden)
	assert.NoError(t, err)

	err = storageClient.Zones.Set(context.Background(), zone)
	assert.NoError(t, err)

	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)

	mqttClient.On("WaterTopic", mock.Anything).Return("test-garden/action/water", nil)
	mqttClient.On("Publish", "test-garden/action/water", mock.Anything).Return(nil)
	mqttClient.On("Disconnect", uint(100)).Return()
	influxdbClient.On("Close").Return()

	worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
	worker.StartAsync()

	ws := createExampleWaterSchedule()

	wsNotInStorage := createExampleWaterSchedule()
	wsNotInStorage.ID = babyapi.ID{ID: id2}

	wsNotActive := createExampleWaterSchedule()
	wsNotActive.ID = babyapi.NewID()
	currentTime := time.Now()
	wsNotActive.ActivePeriod = &pkg.ActivePeriod{
		StartMonth: currentTime.AddDate(0, 1, 0).String(),
		EndMonth:   currentTime.AddDate(0, 2, 0).String(),
	}

	// Set StartTime to the near future
	startTime := pkg.NewStartTime(time.Now().Add(1 * time.Second))
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
			fake.ResetLastMessage()

			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)
			defer weather.ResetCache()

			garden := createExampleGarden()
			zone := createExampleZone()

			notificationClient := &notifications.Client{
				ID:      babyapi.NewID(),
				Name:    "TestClient",
				Type:    "fake",
				Options: map[string]any{},
			}
			err = storageClient.NotificationClientConfigs.Set(context.Background(), notificationClient)
			assert.NoError(t, err)

			err = storageClient.Gardens.Set(context.Background(), garden)
			assert.NoError(t, err)

			err = storageClient.Zones.Set(context.Background(), zone)
			assert.NoError(t, err)

			influxdbClient := new(influxdb.MockClient)
			mqttClient := new(mqtt.MockClient)

			mqttClient.On("WaterTopic", mock.Anything).Return("test-garden/action/water", nil)
			mqttClient.On("Publish", "test-garden/action/water", mock.Anything).Return(errors.New("publish error"))
			mqttClient.On("Disconnect", uint(100)).Return()
			influxdbClient.On("Close").Return()

			worker := NewWorker(storageClient, influxdbClient, mqttClient, slog.Default())
			worker.StartAsync()

			ws := createExampleWaterSchedule()
			ws.Name = "MyWaterSchedule"
			// Set StartTime to the near future
			ws.StartTime = pkg.NewStartTime(time.Now().Add(1 * time.Second))
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
		Driver: "hashmap",
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
	startTime := pkg.NewStartTime(time.Now().Add(-1 * time.Hour))
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
	now := time.Now()

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
				Driver: "hashmap",
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
			ws.StartDate = &now
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
			0,
			24 * time.Hour,
			24 * time.Hour,
		},
		{
			"RunIn5Days",
			0,
			5 * 24 * time.Hour,
			5 * 24 * time.Hour,
		},
		{
			// This tests the scenario where the server is restarted in-between an interval and
			// relies on persistent state to reschedule
			"Every5DaysButStartedAFewDaysAgo",
			-3 * 24 * time.Hour,
			5 * 24 * time.Hour,
			2 * 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
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
			now := time.Now()
			startTime := now.Add(delay)

			ws := createExampleWaterSchedule()
			ws.StartTime = pkg.NewStartTime(startTime)
			startDate := now.Add(tt.startDateOffset)
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
	// TODO: this test was consistently failing when running in GitHub Workflow, but worked fine locally until this commit which
	// changed line 199 of `scheduler.go` (ScheduleLightActions) to delete and re-create Job instead of updating. It's interesting
	// because it used to work fine, so I need to double-check that this these tests are actually testing what I think they are.
	// In other words, this test sometimes tests the update job and sometimes doesn't, depending on when it is run
	t.Run("AdhocOnTimeInFutureOverridesScheduled", func(t *testing.T) {
		worker := NewWorker(nil, nil, nil, slog.Default())
		worker.StartAsync()
		defer worker.Stop()

		now := time.Now().UTC()
		later := now.Add(1 * time.Hour).Truncate(time.Second)
		g := createExampleGarden()
		g.LightSchedule.AdhocOnTime = &later
		err := worker.ScheduleLightActions(g)
		assert.NoError(t, err)

		nextOnTime := worker.GetNextLightTime(g, pkg.LightStateOn)
		assert.Equal(t, later, *nextOnTime)
	})

	t.Run("AdhocOnTimeInPastIsNotUsed", func(t *testing.T) {
		storageClient, err := storage.NewClient(storage.Config{
			Driver: "hashmap",
		})
		assert.NoError(t, err)
		defer weather.ResetCache()

		worker := NewWorker(storageClient, nil, nil, slog.Default())
		worker.StartAsync()
		defer worker.Stop()

		now := time.Now().UTC()
		past := now.Add(-1 * time.Hour)
		g := createExampleGarden()
		g.LightSchedule.AdhocOnTime = &past
		err = worker.ScheduleLightActions(g)
		assert.NoError(t, err)
		if g.LightSchedule.AdhocOnTime != nil {
			t.Errorf("Expected nil AdhocOnTime but got: %v", g.LightSchedule.AdhocOnTime)
		}

		expected := time.Date(
			now.Year(),
			now.Month(),
			now.Day(),
			g.LightSchedule.StartTime.Time.UTC().Hour(),
			g.LightSchedule.StartTime.Time.UTC().Minute(),
			g.LightSchedule.StartTime.Time.UTC().Second(),
			0,
			time.UTC,
		)
		// If expected time is before now, it will be tomorrow
		if expected.Before(now) {
			expected = expected.Add(24 * time.Hour)
		}

		nextOnTime := worker.GetNextLightTime(g, pkg.LightStateOn)
		assert.Equal(t, expected, *nextOnTime)
	})

	t.Run("ScheduledLightActionCreatesNotification", func(t *testing.T) {
		tests := []struct {
			name               string
			opts               map[string]any
			off                bool
			enableNotification bool
			mqttPublishError   error
			expectedOnMessage  string
			expectedOffMessage string
		}{
			{
				"SuccessfulOnAndOff",
				map[string]any{},
				true,
				true,
				nil,
				"test-garden: Light ON",
				"test-garden: Light OFF",
			},
			{
				"NoNotificationWhenDisabledSuccessfulOnAndOff",
				map[string]any{},
				true,
				false,
				nil,
				"",
				"",
			},
			{
				"ErrorCreatingClient",
				map[string]any{"create_error": "error"},
				false,
				true,
				nil,
				"",
				"",
			},
			{
				"ErrorSendingMessage",
				map[string]any{"send_message_error": "error"},
				false,
				true,
				nil,
				"",
				"",
			},
			{
				"ErrorNotification",
				map[string]any{},
				false,
				true,
				errors.New("publish error"),
				"test-garden: Light Action Error",
				"",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fake.ResetLastMessage()

				storageClient, err := storage.NewClient(storage.Config{
					Driver: "hashmap",
				})
				assert.NoError(t, err)

				mqttClient := new(mqtt.MockClient)
				mqttClient.On("LightTopic", mock.Anything).Return("test-garden/action/light", nil)
				mqttClient.On("Publish", "test-garden/action/light", mock.Anything).Return(tt.mqttPublishError)
				mqttClient.On("Disconnect", uint(100)).Return()

				notificationClient := &notifications.Client{
					ID:      babyapi.NewID(),
					Name:    "TestClient",
					Type:    "fake",
					Options: tt.opts,
				}
				err = storageClient.NotificationClientConfigs.Set(context.Background(), notificationClient)
				assert.NoError(t, err)

				worker := NewWorker(storageClient, nil, mqttClient, slog.Default())
				worker.StartAsync()
				defer worker.Stop()

				// Create new LightSchedule that turns on in 1 second for only 1 second
				now := time.Now().UTC()
				later := now.Add(1 * time.Second).Truncate(time.Second)
				g := createExampleGarden()
				g.LightSchedule.StartTime = pkg.NewStartTime(later)
				g.LightSchedule.Duration = &pkg.Duration{Duration: time.Second}
				if tt.enableNotification {
					ncID := notificationClient.GetID()
					g.LightSchedule.NotificationClientID = &ncID
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
}

func TestScheduleLightDelay(t *testing.T) {
	tests := []struct {
		name          string
		garden        *pkg.Garden
		actions       []*action.LightAction
		on            bool
		expectedDelay time.Duration
	}{
		{
			"LightAlreadyOn",
			func() *pkg.Garden {
				g := createExampleGarden()
				// Set start time to a bit ago so the light is considered to be ON
				g.LightSchedule.StartTime = pkg.NewStartTime(time.Now().Add(-1 * time.Minute))
				return g
			}(),
			[]*action.LightAction{
				{
					State:       pkg.LightStateOff,
					ForDuration: &pkg.Duration{Duration: 30 * time.Minute},
				},
			},
			true,
			30 * time.Minute,
		},
		{
			"LightAlreadyOnRunTwiceAppends",
			func() *pkg.Garden {
				g := createExampleGarden()
				// Set start time to a bit ago so the light is considered to be ON
				g.LightSchedule.StartTime = pkg.NewStartTime(time.Now().Add(-1 * time.Minute))
				return g
			}(),
			[]*action.LightAction{
				{
					State:       pkg.LightStateOff,
					ForDuration: &pkg.Duration{Duration: 30 * time.Minute},
				},
				{
					State:       pkg.LightStateOff,
					ForDuration: &pkg.Duration{Duration: 30 * time.Minute},
				},
			},
			true,
			60 * time.Minute,
		},
		{
			"LightStillOff",
			func() *pkg.Garden {
				g := createExampleGarden()
				// Set start time to the future so the light is considered to be OFF
				g.LightSchedule.StartTime = pkg.NewStartTime(time.Now().Add(5 * time.Minute))
				return g
			}(),
			[]*action.LightAction{
				{
					State:       pkg.LightStateOff,
					ForDuration: &pkg.Duration{Duration: 30 * time.Minute},
				},
			},
			false,
			30 * time.Minute,
		},
		{
			"LightStillOffRunTwiceAppends",
			func() *pkg.Garden {
				g := createExampleGarden()
				// Set start time to the future so the light is considered to be OFF
				g.LightSchedule.StartTime = pkg.NewStartTime(time.Now().Add(5 * time.Minute))
				return g
			}(),
			[]*action.LightAction{
				{
					State:       pkg.LightStateOff,
					ForDuration: &pkg.Duration{Duration: 30 * time.Minute},
				},
				{
					State:       pkg.LightStateOff,
					ForDuration: &pkg.Duration{Duration: 30 * time.Minute},
				},
			},
			false,
			60 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient, err := storage.NewClient(storage.Config{
				Driver: "hashmap",
			})
			assert.NoError(t, err)
			defer weather.ResetCache()

			worker := NewWorker(storageClient, nil, nil, slog.Default())
			worker.StartAsync()
			defer worker.Stop()

			err = worker.ScheduleLightActions(tt.garden)
			assert.NoError(t, err)

			// Now request delay
			now := time.Now()
			for _, action := range tt.actions {
				err = worker.ScheduleLightDelay(tt.garden, action)
				assert.NoError(t, err)
			}

			var expected time.Time
			if tt.on {
				expected = now.Add(tt.expectedDelay).Truncate(time.Second)
			} else {
				expected = time.Date(
					now.Year(),
					now.Month(),
					now.Day(),
					tt.garden.LightSchedule.StartTime.Time.Hour(),
					tt.garden.LightSchedule.StartTime.Time.Minute(),
					tt.garden.LightSchedule.StartTime.Time.Second(),
					0,
					time.Local,
				).Add(tt.expectedDelay).Truncate(time.Second)
			}

			nextOnTime := worker.GetNextLightTime(tt.garden, pkg.LightStateOn).Truncate(time.Second)
			assert.Equal(t, expected.UTC(), nextOnTime)
		})
	}

	t.Run("ErrorDelayingPastNextOffTime", func(t *testing.T) {
		storageClient, err := storage.NewClient(storage.Config{
			Driver: "hashmap",
		})
		assert.NoError(t, err)
		defer weather.ResetCache()

		worker := NewWorker(storageClient, nil, nil, slog.Default())
		worker.StartAsync()
		defer worker.Stop()

		g := createExampleGarden()
		// Set StartTime and Duration so NextOffTime is soon
		g.LightSchedule.StartTime = pkg.NewStartTime(time.Now().Add(-1 * time.Hour))
		g.LightSchedule.Duration = &pkg.Duration{Duration: 1*time.Hour + 5*time.Minute}

		err = worker.ScheduleLightActions(g)
		assert.NoError(t, err)

		// Now request delay
		err = worker.ScheduleLightDelay(g, &action.LightAction{
			State:       pkg.LightStateOff,
			ForDuration: &pkg.Duration{Duration: 30 * time.Minute},
		})
		if err == nil {
			t.Errorf("Expected error but got nil")
		}
		if err.Error() != "unable to schedule delay that extends past the light turning back on" {
			t.Errorf("Unexpected error string: %v", err)
		}
	})

	t.Run("ErrorDelayingLongerThanLightDuration", func(t *testing.T) {
		storageClient, err := storage.NewClient(storage.Config{
			Driver: "hashmap",
		})
		assert.NoError(t, err)
		defer weather.ResetCache()

		worker := NewWorker(storageClient, nil, nil, slog.Default())
		worker.StartAsync()
		defer worker.Stop()

		g := createExampleGarden()

		err = worker.ScheduleLightActions(g)
		assert.NoError(t, err)

		// Now request delay
		err = worker.ScheduleLightDelay(g, &action.LightAction{
			State:       pkg.LightStateOff,
			ForDuration: &pkg.Duration{Duration: 16 * time.Hour},
		})
		if err == nil {
			t.Errorf("Expected error but got nil")
		}
		if err.Error() != "unable to execute delay that lasts longer than light_schedule" {
			t.Errorf("Unexpected error string: %v", err)
		}
	})

	t.Run("ErrorSettingDelayWithoutOFFState", func(t *testing.T) {
		storageClient, err := storage.NewClient(storage.Config{
			Driver: "hashmap",
		})
		assert.NoError(t, err)
		defer weather.ResetCache()

		worker := NewWorker(storageClient, nil, nil, slog.Default())
		worker.StartAsync()
		defer worker.Stop()

		g := createExampleGarden()

		err = worker.ScheduleLightActions(g)
		assert.NoError(t, err)

		// Now request delay
		err = worker.ScheduleLightDelay(g, &action.LightAction{
			State:       pkg.LightStateOn,
			ForDuration: &pkg.Duration{Duration: 30 * time.Minute},
		})
		if err == nil {
			t.Errorf("Expected error but got nil")
		}
		if err.Error() != "unable to use delay when state is not OFF" {
			t.Errorf("Unexpected error string: %v", err)
		}
	})
}

func TestRemoveJobsByID(t *testing.T) {
	storageClient, err := storage.NewClient(storage.Config{
		Driver: "hashmap",
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
	startTime := pkg.NewStartTime(time.Now().Add(-1 * time.Hour))
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

	now := time.Now()
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
