package http

import (
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/api"
	"github.com/calvinmclean/automated-garden/garden-app/api/actions"
	"github.com/go-co-op/gocron"
)

// initializeScheduler will create a new Scheduler and schedule Jobs for each Plant
func initializeScheduler() {
	scheduler = gocron.NewScheduler(time.Local)
	for _, p := range storageClient.GetPlants(false) {
		if err := addWateringSchedule(p); err != nil {
			logger.Errorf("Unable to add watering Job for Plant %s: %v", p.ID.String(), err)
		}
	}
	scheduler.StartAsync()
}

// addWateringSchedule will schedule watering actions for the Plant based off the StartDate
// and Interval. The scheduled Job is tagged with the Plant's ID so it can easily be removed
func addWateringSchedule(p *api.Plant) error {
	logger.Infof("Creating scheduled Job for watering Plant %s", p.ID.String())

	// Read Plant's Interval string into a Duration
	duration, err := time.ParseDuration(p.Interval)
	if err != nil {
		return err
	}

	// Schedule the WaterAction execution
	action := &actions.WaterAction{Duration: p.WateringAmount}
	_, err = scheduler.
		Every(duration).
		StartAt(*p.StartDate).
		Tag(p.ID.String()).
		Do(action.Execute, p)
	return err
}

// removeWateringSchedule is used to remove the Plant's scheduled watering Job
// from the scheduler.
func removeWateringSchedule(p *api.Plant) error {
	return scheduler.RemoveByTag(p.ID.String())
}
