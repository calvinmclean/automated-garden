package action

import (
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

func createExampleGarden() *pkg.Garden {
	two := uint(2)
	time, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	id, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	return &pkg.Garden{
		Name:        "test-garden",
		TopicPrefix: "test-garden",
		MaxPlants:   &two,
		ID:          id,
		Plants:      map[xid.ID]*pkg.Plant{},
		CreatedAt:   &time,
		LightSchedule: &pkg.LightSchedule{
			Duration:  "15h",
			StartTime: "22:00:01-07:00",
		},
	}
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
			t.Errorf("Unexpected error when scheduling WateringAction: %v", err)
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
			t.Errorf("Unexpected error when scheduling WateringAction: %v", err)
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
			expected = expected.Add(lightingInterval)
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
				t.Errorf("Unexpected error when scheduling WateringAction: %v", err)
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
			t.Errorf("Unexpected error when scheduling WateringAction: %v", err)
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
			t.Errorf("Unexpected error when scheduling WateringAction: %v", err)
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
			t.Errorf("Unexpected error when scheduling WateringAction: %v", err)
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
