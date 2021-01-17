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

	duration, err := time.ParseDuration(p.Interval)
	if err != nil {
		return err
	}
	action := &actions.WaterAction{Duration: p.WateringAmount}
	_, err = scheduler.
		Every(uint64(duration.Minutes())).
		StartAt(*p.StartDate).
		Minutes().
		SetTag([]string{p.ID.String()}).
		Do(action.Execute, p)
	return err
}

// removeWateringSchedule is used to remove the Plant's scheduled watering Job
// from the scheduler.
// TODO: Fix this
//       Currently, the RemoveJobByTag will not remove a scheduled Job. It will
//       remove it from the list of Jobs but it continues to execute. Instead, I
//       am reinitializing the Scheduler after end-dating the Plant
func removeWateringSchedule(p *api.Plant) error {
	// return scheduler.RemoveJobByTag(p.ID.String()); err != nil {

	// Stop and restart the scheduler
	scheduler.Stop()
	initializeScheduler()
	return nil
}
