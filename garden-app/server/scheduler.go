package server

import (
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/api"
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
	duration, err := time.ParseDuration(p.WateringStrategy.Interval)
	if err != nil {
		return err
	}

	// Schedule the WaterAction execution
	action := p.WateringAction()
	_, err = scheduler.
		Every(duration).
		StartAt(*p.StartDate).
		Tag(p.ID.String()).
		Do(func() {
			err = action.Execute(p)
			if err != nil {
				logger.Error("Error executing scheduled plant watering action: ", err)
			}
			err = storageClient.SavePlant(p)
			if err != nil {
				logger.Error("Error saving plant after watering: ", err)
			}
		})
	return err
}

// removeWateringSchedule is used to remove the Plant's scheduled watering Job
// from the scheduler.
func removeWateringSchedule(p *api.Plant) error {
	return scheduler.RemoveByTag(p.ID.String())
}

// resetWateringSchedule will simply remove the existing Job and create a new one
func resetWateringSchedule(p *api.Plant) error {
	if err := removeWateringSchedule(p); err != nil {
		return err
	}
	return addWateringSchedule(p)
}

// getNextWateringTime determines the next scheduled watering time for a given Plant using tags
func getNextWateringTime(p *api.Plant) *time.Time {
	for _, job := range scheduler.Jobs() {
		for _, tag := range job.Tags() {
			if tag == p.ID.String() {
				result := job.NextRun()
				return &result
			}
		}
	}
	return nil
}
