package worker

import (
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var id, _ = xid.FromString("c5cvhpcbcv45e8bp16dg")

func createExampleGarden() *pkg.Garden {
	two := uint(2)
	createdAt, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	return &pkg.Garden{
		Name:        "test-garden",
		TopicPrefix: "test-garden",
		MaxZones:    &two,
		ID:          id,
		Zones:       map[xid.ID]*pkg.Zone{},
		CreatedAt:   &createdAt,
		LightSchedule: &pkg.LightSchedule{
			Duration:  &pkg.Duration{Duration: 15 * time.Hour},
			StartTime: "22:00:01-07:00",
		},
	}
}

func createExampleZone() *pkg.Zone {
	createdAt, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	p := uint(0)
	return &pkg.Zone{
		Name:             "test zone",
		ID:               id,
		CreatedAt:        &createdAt,
		Position:         &p,
		WaterScheduleIDs: []xid.ID{id},
	}
}

func createExampleWaterSchedule() *pkg.WaterSchedule {
	createdAt, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	return &pkg.WaterSchedule{
		ID:        id,
		Duration:  &pkg.Duration{Duration: time.Second},
		Interval:  &pkg.Duration{Duration: time.Hour * 24},
		StartTime: &createdAt,
	}
}

func TestScheduleWaterAction(t *testing.T) {
	storageClient := new(storage.MockClient)
	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)

	mqttClient.On("WaterTopic", mock.Anything).Return("test-garden/action/water", nil)
	mqttClient.On("Publish", "test-garden/action/water", mock.Anything).Return(nil)
	mqttClient.On("Disconnect", uint(100)).Return()
	influxdbClient.On("Close").Return()
	storageClient.On("GetZonesUsingWaterSchedule", id).Return([]*pkg.ZoneAndGarden{
		{
			Zone:   createExampleZone(),
			Garden: createExampleGarden(),
		},
	}, nil)

	worker := NewWorker(storageClient, influxdbClient, mqttClient, logrus.New())
	worker.StartAsync()

	ws := createExampleWaterSchedule()
	// Set Zone's WaterSchedule.StartTime to the near future
	startTime := time.Now().Add(250 * time.Millisecond)
	ws.StartTime = &startTime
	err := worker.ScheduleWaterAction(ws)
	if err != nil {
		t.Errorf("Unexpected error when scheduling WaterAction: %v", err)
	}

	time.Sleep(1000 * time.Millisecond)

	worker.Stop()
	storageClient.AssertExpectations(t)
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}

func TestResetNextWaterTime(t *testing.T) {
	storageClient := new(storage.MockClient)
	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)
	mqttClient.On("Disconnect", uint(100)).Return()
	influxdbClient.On("Close").Return()

	worker := NewWorker(storageClient, influxdbClient, mqttClient, logrus.New())
	worker.StartAsync()

	ws := createExampleWaterSchedule()
	// Set Zone's WaterSchedule.StartTime to a time that won't cause it to run
	startTime := time.Now().Add(-1 * time.Hour)
	ws.StartTime = &startTime
	err := worker.ScheduleWaterAction(ws)
	if err != nil {
		t.Errorf("Unexpected error when scheduling WaterAction: %v", err)
	}

	// Change WaterSchedule and restart
	newTime := startTime.Add(-30 * time.Minute)
	ws.StartTime = &newTime
	err = worker.ResetWaterSchedule(ws)
	if err != nil {
		t.Errorf("Unexpected error when resetting WaterAction: %v", err)
	}

	nextWaterTime := worker.GetNextWaterTime(ws)
	expected := startTime.Add(-30 * time.Minute).Add(24 * time.Hour)
	if *nextWaterTime != expected {
		t.Errorf("Expected %v but got: %v", nextWaterTime, expected)
	}

	worker.Stop()
	storageClient.AssertExpectations(t)
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}

func TestGetNextWaterTime(t *testing.T) {
	storageClient := new(storage.MockClient)
	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)
	mqttClient.On("Disconnect", uint(100)).Return()
	influxdbClient.On("Close").Return()

	worker := NewWorker(storageClient, influxdbClient, mqttClient, logrus.New())
	worker.StartAsync()

	ws := createExampleWaterSchedule()
	// Set WaterSchedule.StartTime to a time that won't cause it to run
	startTime := time.Now().Add(-1 * time.Hour)
	ws.StartTime = &startTime
	err := worker.ScheduleWaterAction(ws)
	if err != nil {
		t.Errorf("Unexpected error when scheduling WaterAction: %v", err)
	}

	nextWaterTime := worker.GetNextWaterTime(ws)
	expected := startTime.Add(24 * time.Hour)
	if *nextWaterTime != expected {
		t.Errorf("Expected %v but got: %v", nextWaterTime, expected)
	}

	worker.Stop()
	storageClient.AssertExpectations(t)
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}

func TestScheduleLightActions(t *testing.T) {
	// TODO: this test was consistently failing when running in GitHub Workflow, but worked fine locally until this commit which
	// changed line 199 of `scheduler.go` (ScheduleLightActions) to delete and re-create Job instead of updating. It's interesting
	// because it used to work fine, so I need to double-check that this these tests are actually testing what I think they are.
	// In other words, this test sometimes tests the update job and sometimes doesn't, depending on when it is run
	t.Run("AdhocOnTimeInFutureOverridesScheduled", func(t *testing.T) {
		worker := NewWorker(nil, nil, nil, logrus.New())
		worker.StartAsync()
		defer worker.Stop()

		now := time.Now()
		later := now.Add(1 * time.Hour).Truncate(time.Second)
		g := createExampleGarden()
		g.LightSchedule.AdhocOnTime = &later
		err := worker.ScheduleLightActions(g)
		if err != nil {
			t.Errorf("Unexpected error when scheduling LightAction: %v", err)
		}

		nextOnTime := worker.GetNextLightTime(g, pkg.LightStateOn)
		assert.Equal(t, later, *nextOnTime)
	})
	t.Run("AdhocOnTimeInPastIsNotUsed", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		worker := NewWorker(storageClient, nil, nil, logrus.New())
		worker.StartAsync()
		defer worker.Stop()

		now := time.Now()
		past := now.Add(-1 * time.Hour)
		g := createExampleGarden()
		g.LightSchedule.AdhocOnTime = &past
		storageClient.On("SaveGarden", g).Return(nil)
		err := worker.ScheduleLightActions(g)
		if err != nil {
			t.Errorf("Unexpected error when scheduling LightAction: %v", err)
		}
		if g.LightSchedule.AdhocOnTime != nil {
			t.Errorf("Expected nil AdhocOnTime but got: %v", g.LightSchedule.AdhocOnTime)
		}

		lightTime, _ := time.Parse(pkg.LightTimeFormat, g.LightSchedule.StartTime)
		expected := time.Date(
			now.In(lightTime.Location()).Year(),
			now.In(lightTime.Location()).Month(),
			now.In(lightTime.Location()).Day(),
			lightTime.Hour(),
			lightTime.Minute(),
			lightTime.Second(),
			0,
			time.Local,
		)
		// If expected time is before now, it will be tomorrow
		if expected.Before(now) {
			expected = expected.Add(24 * time.Hour)
		}

		nextOnTime := worker.GetNextLightTime(g, pkg.LightStateOn)
		assert.Equal(t, expected, *nextOnTime)
		storageClient.AssertExpectations(t)
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
				g.LightSchedule.StartTime = time.Now().Add(-1 * time.Minute).Format(pkg.LightTimeFormat)
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
				g.LightSchedule.StartTime = time.Now().Add(-1 * time.Minute).Format(pkg.LightTimeFormat)
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
				g.LightSchedule.StartTime = time.Now().Add(5 * time.Minute).Format(pkg.LightTimeFormat)
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
				g.LightSchedule.StartTime = time.Now().Add(5 * time.Minute).Format(pkg.LightTimeFormat)
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
			storageClient := new(storage.MockClient)
			worker := NewWorker(storageClient, nil, nil, logrus.New())
			worker.StartAsync()
			defer worker.Stop()

			storageClient.On("SaveGarden", tt.garden).Return(nil)
			err := worker.ScheduleLightActions(tt.garden)
			if err != nil {
				t.Errorf("Unexpected error when scheduling LightAction: %v", err)
			}

			// Now request delay
			now := time.Now()
			for _, action := range tt.actions {
				err = worker.ScheduleLightDelay(tt.garden, action)
				if err != nil {
					t.Errorf("Unexpected error when scheduling delay: %v", err)
				}
			}

			var expected time.Time
			if tt.on {
				expected = now.Add(tt.expectedDelay).Truncate(time.Second)
			} else {
				lightTime, _ := time.Parse(pkg.LightTimeFormat, tt.garden.LightSchedule.StartTime)
				expected = time.Date(
					now.Year(),
					now.Month(),
					now.Day(),
					lightTime.Hour(),
					lightTime.Minute(),
					lightTime.Second(),
					0,
					time.Local,
				).Add(tt.expectedDelay).Truncate(time.Second)
			}

			nextOnTime := worker.GetNextLightTime(tt.garden, pkg.LightStateOn).Truncate(time.Second)
			assert.Equal(t, expected, nextOnTime)
			storageClient.AssertExpectations(t)
		})
	}

	t.Run("ErrorDelayingPastNextOffTime", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		worker := NewWorker(storageClient, nil, nil, logrus.New())
		worker.StartAsync()
		defer worker.Stop()

		g := createExampleGarden()
		// Set StartTime and Duration so NextOffTime is soon
		g.LightSchedule.StartTime = time.Now().Add(-1 * time.Hour).Format(pkg.LightTimeFormat)
		g.LightSchedule.Duration = &pkg.Duration{Duration: 1*time.Hour + 2*time.Minute}

		err := worker.ScheduleLightActions(g)
		if err != nil {
			t.Errorf("Unexpected error when scheduling LightAction: %v", err)
		}

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
		storageClient.AssertExpectations(t)
	})

	t.Run("ErrorDelayingLongerThanLightDuration", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		worker := NewWorker(storageClient, nil, nil, logrus.New())
		worker.StartAsync()
		defer worker.Stop()

		g := createExampleGarden()

		err := worker.ScheduleLightActions(g)
		if err != nil {
			t.Errorf("Unexpected error when scheduling LightAction: %v", err)
		}

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
		storageClient.AssertExpectations(t)
	})

	t.Run("ErrorSettingDelayWithoutOFFState", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		worker := NewWorker(storageClient, nil, nil, logrus.New())
		worker.StartAsync()
		defer worker.Stop()

		g := createExampleGarden()

		err := worker.ScheduleLightActions(g)
		if err != nil {
			t.Errorf("Unexpected error when scheduling LightAction: %v", err)
		}

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
		storageClient.AssertExpectations(t)
	})
}

func TestRemoveJobsByID(t *testing.T) {
	storageClient := new(storage.MockClient)
	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)
	mqttClient.On("Disconnect", uint(100)).Return()
	influxdbClient.On("Close").Return()

	worker := NewWorker(storageClient, influxdbClient, mqttClient, logrus.New())
	worker.StartAsync()

	ws := createExampleWaterSchedule()
	// Set Zone's WaterSchedule.StartTime to a time that won't cause it to run
	startTime := time.Now().Add(-1 * time.Hour)
	ws.StartTime = &startTime
	err := worker.ScheduleWaterAction(ws)
	if err != nil {
		t.Errorf("Unexpected error when scheduling WaterAction: %v", err)
	}

	err = worker.RemoveJobsByID(ws.ID)
	if err != nil {
		t.Errorf("Unexpected error when removing jobs: %v", err)
	}

	// This also gets coverage for GetNextWaterTime when no Job exists
	nextWaterTime := worker.GetNextWaterTime(ws)
	if nextWaterTime != nil {
		t.Errorf("Expected nil but got: %v", nextWaterTime)
	}

	worker.Stop()
	storageClient.AssertExpectations(t)
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}
