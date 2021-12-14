package server

import (
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/go-co-op/gocron"
)

func TestScheduleLightActions(t *testing.T) {
	t.Run("AdhocOnTimeInFutureOverridesScheduled", func(t *testing.T) {
		gr := GardensResource{
			scheduler: gocron.NewScheduler(time.Local),
		}
		gr.scheduler.StartAsync()
		defer gr.scheduler.Stop()

		now := time.Now()
		later := now.Add(1 * time.Hour)
		g := createExampleGarden()
		g.LightSchedule.AdhocOnTime = &later
		err := gr.scheduleLightActions(g)
		if err != nil {
			t.Errorf("Unexpected error when scheduling WateringAction: %v", err)
		}

		nextOnTime := gr.getNextLightTime(g, pkg.StateOn)
		if *nextOnTime != later {
			t.Errorf("Unexpected nextOnTime: expected=%v, actual=%v", later, *nextOnTime)
		}
	})
	t.Run("AdhocOnTimeInPastIsNotUsed", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		gr := GardensResource{
			storageClient: storageClient,
			scheduler:     gocron.NewScheduler(time.Local),
		}
		gr.scheduler.StartAsync()
		defer gr.scheduler.Stop()

		now := time.Now()
		past := now.Add(-1 * time.Hour)
		g := createExampleGarden()
		g.LightSchedule.AdhocOnTime = &past
		storageClient.On("SaveGarden", g).Return(nil)
		err := gr.scheduleLightActions(g)
		if err != nil {
			t.Errorf("Unexpected error when scheduling WateringAction: %v", err)
		}

		lightTime, _ := time.Parse(pkg.LightTimeFormat, g.LightSchedule.StartTime)
		expected := time.Date(
			now.Year(),
			now.Month(),
			now.Day(),
			lightTime.Hour(),
			lightTime.Minute(),
			lightTime.Second(),
			0,
			lightTime.Location(),
		)

		nextOnTime := gr.getNextLightTime(g, pkg.StateOn)
		if nextOnTime.UnixNano() != expected.UnixNano() {
			t.Errorf("Unexpected nextOnTime: expected=%v, actual=%v", expected.UnixNano(), nextOnTime.UnixNano())
		}
		storageClient.AssertExpectations(t)
	})
}

func TestScheduleLightDelay(t *testing.T) {
	tests := []struct {
		name          string
		garden        *pkg.Garden
		actions       []*pkg.LightAction
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
			[]*pkg.LightAction{
				{
					State:       pkg.StateOff,
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
			[]*pkg.LightAction{
				{
					State:       pkg.StateOff,
					ForDuration: "30m",
				},
				{
					State:       pkg.StateOff,
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
			[]*pkg.LightAction{
				{
					State:       pkg.StateOff,
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
			[]*pkg.LightAction{
				{
					State:       pkg.StateOff,
					ForDuration: "30m",
				},
				{
					State:       pkg.StateOff,
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
			gr := GardensResource{
				storageClient: storageClient,
				scheduler:     gocron.NewScheduler(time.Local),
			}
			gr.scheduler.StartAsync()
			defer gr.scheduler.Stop()

			storageClient.On("SaveGarden", tt.garden).Return(nil)
			err := gr.scheduleLightActions(tt.garden)
			if err != nil {
				t.Errorf("Unexpected error when scheduling WateringAction: %v", err)
			}

			// Now request delay
			now := time.Now()
			for _, action := range tt.actions {
				err = gr.scheduleLightDelay(tt.garden, action)
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

			nextOnTime := gr.getNextLightTime(tt.garden, pkg.StateOn).Truncate(time.Second)
			if nextOnTime.UnixNano() != expected.UnixNano() {
				t.Errorf("Unexpected nextOnTime: expected=%v, actual=%v", expected, nextOnTime)
			}
			storageClient.AssertExpectations(t)
		})
	}

	t.Run("ErrorDelayingPastNextOffTime", func(t *testing.T) {
		storageClient := new(storage.MockClient)
		gr := GardensResource{
			storageClient: storageClient,
			scheduler:     gocron.NewScheduler(time.Local),
		}
		gr.scheduler.StartAsync()
		defer gr.scheduler.Stop()

		g := createExampleGarden()
		// Set StartTime and Duration so NextOffTime is soon
		g.LightSchedule.StartTime = time.Now().Add(-1 * time.Hour).Format(pkg.LightTimeFormat)
		g.LightSchedule.Duration = "1h2m"

		err := gr.scheduleLightActions(g)
		if err != nil {
			t.Errorf("Unexpected error when scheduling WateringAction: %v", err)
		}

		// Now request delay
		err = gr.scheduleLightDelay(g, &pkg.LightAction{
			State:       pkg.StateOff,
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
}
