package action

import (
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

func createExampleGarden() *pkg.Garden {
	two := uint(2)
	time, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	id, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	return &pkg.Garden{
		Name:        "test-garden",
		TopicPrefix: "test-garden",
		MaxZones:    &two,
		ID:          id,
		Zones:       map[xid.ID]*pkg.Zone{},
		CreatedAt:   &time,
		LightSchedule: &pkg.LightSchedule{
			Duration:  "15h",
			StartTime: "22:00:01-07:00",
		},
	}
}

func createExampleZone() *pkg.Zone {
	time, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	id, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	p := uint(0)
	return &pkg.Zone{
		Name:      "test zone",
		ID:        id,
		CreatedAt: &time,
		Position:  &p,
		WaterSchedule: &pkg.WaterSchedule{
			Duration:  "1000ms",
			Interval:  "24h",
			StartTime: &time,
		},
	}
}

func TestScheduleWaterAction(t *testing.T) {
	storageClient := new(storage.MockClient)
	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)

	mqttClient.On("WaterTopic", mock.Anything).Return("test-garden/action/water", nil)
	mqttClient.On("Publish", "test-garden/action/water", mock.Anything).Return(nil)
	influxdbClient.On("Close").Return()

	scheduler := NewScheduler(storageClient, influxdbClient, mqttClient, logrus.StandardLogger())
	scheduler.StartAsync()
	defer scheduler.Stop()

	g := createExampleGarden()
	z := createExampleZone()
	// Set Zone's WaterSchedule.StartTime to the near future
	startTime := time.Now().Add(250 * time.Millisecond)
	z.WaterSchedule.StartTime = &startTime
	err := scheduler.ScheduleWaterAction(g, z)
	if err != nil {
		t.Errorf("Unexpected error when scheduling WaterAction: %v", err)
	}

	time.Sleep(1000 * time.Millisecond)

	storageClient.AssertExpectations(t)
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}

func TestResetNextWaterTime(t *testing.T) {
	storageClient := new(storage.MockClient)
	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)

	scheduler := NewScheduler(storageClient, influxdbClient, mqttClient, logrus.StandardLogger())
	scheduler.StartAsync()
	defer scheduler.Stop()

	g := createExampleGarden()
	z := createExampleZone()
	// Set Zone's WaterSchedule.StartTime to a time that won't cause it to run
	startTime := time.Now().Add(-1 * time.Hour)
	z.WaterSchedule.StartTime = &startTime
	err := scheduler.ScheduleWaterAction(g, z)
	if err != nil {
		t.Errorf("Unexpected error when scheduling WaterAction: %v", err)
	}

	// Change WaterSchedule and restart
	newTime := startTime.Add(-30 * time.Minute)
	z.WaterSchedule.StartTime = &newTime
	err = scheduler.ResetWaterSchedule(g, z)
	if err != nil {
		t.Errorf("Unexpected error when resetting WaterAction: %v", err)
	}

	nextWaterTime := scheduler.GetNextWaterTime(z)
	expected := startTime.Add(-30 * time.Minute).Add(24 * time.Hour)
	if *nextWaterTime != expected {
		t.Errorf("Expected %v but got: %v", nextWaterTime, expected)
	}

	storageClient.AssertExpectations(t)
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}

func TestGetNextWaterTime(t *testing.T) {
	storageClient := new(storage.MockClient)
	influxdbClient := new(influxdb.MockClient)
	mqttClient := new(mqtt.MockClient)

	scheduler := NewScheduler(storageClient, influxdbClient, mqttClient, logrus.StandardLogger())
	scheduler.StartAsync()
	defer scheduler.Stop()

	g := createExampleGarden()
	z := createExampleZone()
	// Set Zone's WaterSchedule.StartTime to a time that won't cause it to run
	startTime := time.Now().Add(-1 * time.Hour)
	z.WaterSchedule.StartTime = &startTime
	err := scheduler.ScheduleWaterAction(g, z)
	if err != nil {
		t.Errorf("Unexpected error when scheduling WaterAction: %v", err)
	}

	nextWaterTime := scheduler.GetNextWaterTime(z)
	expected := startTime.Add(24 * time.Hour)
	if *nextWaterTime != expected {
		t.Errorf("Expected %v but got: %v", nextWaterTime, expected)
	}

	storageClient.AssertExpectations(t)
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}

func TestScheduleLightActions(t *testing.T) {
	t.Run("AdhocOnTimeInFutureOverridesScheduled", func(t *testing.T) {
		scheduler := NewScheduler(nil, nil, nil, logrus.StandardLogger())
		scheduler.StartAsync()
		defer scheduler.Stop()

		now := time.Now()
		later := now.Add(1 * time.Hour)
		g := createExampleGarden()
		g.LightSchedule.AdhocOnTime = &later
		err := scheduler.ScheduleLightActions(g)
		if err != nil {
			t.Errorf("Unexpected error when scheduling LightAction: %v", err)
		}

		nextOnTime := scheduler.GetNextLightTime(g, pkg.LightStateOn)
		if *nextOnTime != later {
			t.Errorf("Unexpected nextOnTime: expected=%v, actual=%v", later, *nextOnTime)
		}
	})
	t.Run("AdhocOnTimeInPastIsNotUsed", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		scheduler := NewScheduler(storageClient, nil, nil, logrus.StandardLogger())
		scheduler.StartAsync()
		defer scheduler.Stop()

		now := time.Now()
		past := now.Add(-1 * time.Hour)
		g := createExampleGarden()
		g.LightSchedule.AdhocOnTime = &past
		storageClient.On("SaveGarden", g).Return(nil)
		err := scheduler.ScheduleLightActions(g)
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
			lightTime.Location(),
		)
		// If expected time is before now, it will be tomorrow
		if expected.Before(now) {
			expected = expected.Add(lightInterval)
		}

		nextOnTime := scheduler.GetNextLightTime(g, pkg.LightStateOn)
		if nextOnTime.UnixNano() != expected.UnixNano() {
			t.Errorf("Unexpected nextOnTime: expected=%v, actual=%v", expected, nextOnTime)
		}
		storageClient.AssertExpectations(t)
	})
}

func TestScheduleLightDelay(t *testing.T) {
	tests := []struct {
		name          string
		garden        *pkg.Garden
		actions       []*LightAction
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
			[]*LightAction{
				{
					State:       pkg.LightStateOff,
					ForDuration: "30m",
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
			[]*LightAction{
				{
					State:       pkg.LightStateOff,
					ForDuration: "30m",
				},
				{
					State:       pkg.LightStateOff,
					ForDuration: "30m",
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
			[]*LightAction{
				{
					State:       pkg.LightStateOff,
					ForDuration: "30m",
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
			[]*LightAction{
				{
					State:       pkg.LightStateOff,
					ForDuration: "30m",
				},
				{
					State:       pkg.LightStateOff,
					ForDuration: "30m",
				},
			},
			false,
			60 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storageClient := new(storage.MockClient)
			scheduler := NewScheduler(storageClient, nil, nil, logrus.StandardLogger())
			scheduler.StartAsync()
			defer scheduler.Stop()

			storageClient.On("SaveGarden", tt.garden).Return(nil)
			err := scheduler.ScheduleLightActions(tt.garden)
			if err != nil {
				t.Errorf("Unexpected error when scheduling LightAction: %v", err)
			}

			// Now request delay
			now := time.Now()
			for _, action := range tt.actions {
				err = scheduler.ScheduleLightDelay(tt.garden, action)
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
					lightTime.Location(),
				).Add(tt.expectedDelay).Truncate(time.Second)
			}

			nextOnTime := scheduler.GetNextLightTime(tt.garden, pkg.LightStateOn).Truncate(time.Second)
			if nextOnTime.UnixNano() != expected.UnixNano() {
				t.Errorf("Unexpected nextOnTime: expected=%v, actual=%v", expected, nextOnTime)
			}
			storageClient.AssertExpectations(t)
		})
	}

	t.Run("ErrorDelayingPastNextOffTime", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		scheduler := NewScheduler(storageClient, nil, nil, logrus.StandardLogger())
		scheduler.StartAsync()
		defer scheduler.Stop()

		g := createExampleGarden()
		// Set StartTime and Duration so NextOffTime is soon
		g.LightSchedule.StartTime = time.Now().Add(-1 * time.Hour).Format(pkg.LightTimeFormat)
		g.LightSchedule.Duration = "1h2m"

		err := scheduler.ScheduleLightActions(g)
		if err != nil {
			t.Errorf("Unexpected error when scheduling LightAction: %v", err)
		}

		// Now request delay
		err = scheduler.ScheduleLightDelay(g, &LightAction{
			State:       pkg.LightStateOff,
			ForDuration: "30m",
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
		scheduler := NewScheduler(storageClient, nil, nil, logrus.StandardLogger())
		scheduler.StartAsync()
		defer scheduler.Stop()

		g := createExampleGarden()

		err := scheduler.ScheduleLightActions(g)
		if err != nil {
			t.Errorf("Unexpected error when scheduling LightAction: %v", err)
		}

		// Now request delay
		err = scheduler.ScheduleLightDelay(g, &LightAction{
			State:       pkg.LightStateOff,
			ForDuration: "16h",
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
		scheduler := NewScheduler(storageClient, nil, nil, logrus.StandardLogger())
		scheduler.StartAsync()
		defer scheduler.Stop()

		g := createExampleGarden()

		err := scheduler.ScheduleLightActions(g)
		if err != nil {
			t.Errorf("Unexpected error when scheduling LightAction: %v", err)
		}

		// Now request delay
		err = scheduler.ScheduleLightDelay(g, &LightAction{
			State:       pkg.LightStateOn,
			ForDuration: "30m",
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

	scheduler := NewScheduler(storageClient, influxdbClient, mqttClient, logrus.StandardLogger())
	scheduler.StartAsync()
	defer scheduler.Stop()

	g := createExampleGarden()
	z := createExampleZone()
	// Set Zone's WaterSchedule.StartTime to a time that won't cause it to run
	startTime := time.Now().Add(-1 * time.Hour)
	z.WaterSchedule.StartTime = &startTime
	err := scheduler.ScheduleWaterAction(g, z)
	if err != nil {
		t.Errorf("Unexpected error when scheduling WaterAction: %v", err)
	}

	err = scheduler.RemoveJobsByID(z.ID)
	if err != nil {
		t.Errorf("Unexpected error when removing jobs: %v", err)
	}

	// This also gets coverage for GetNextWaterTime when no Job exists
	nextWaterTime := scheduler.GetNextWaterTime(z)
	if nextWaterTime != nil {
		t.Errorf("Expected nil but got: %v", nextWaterTime)
	}

	storageClient.AssertExpectations(t)
	influxdbClient.AssertExpectations(t)
	mqttClient.AssertExpectations(t)
}
