package server

import (
	"errors"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/go-co-op/gocron"
)

// addWateringSchedule will schedule watering actions for the Plant based off the CreatedAt date,
// WateringStrategy time, and Interval. The scheduled Job is tagged with the Plant's ID so it can
// easily be removed
func (pr PlantsResource) addWateringSchedule(g *pkg.Garden, p *pkg.Plant) error {
	logger.Infof("Creating scheduled Job for watering Plant %s", p.ID.String())

	// Read Plant's Interval string into a Duration
	duration, err := time.ParseDuration(p.WateringStrategy.Interval)
	if err != nil {
		return err
	}

	// Parse Plant's WateringStrategy.Time (has no "date")
	waterTime, err := time.Parse(pkg.WaterTimeFormat, p.WateringStrategy.StartTime)
	if err != nil {
		return err
	}

	// Create startDate using the CreatedAt date with the WateringStrategy's timestamp
	startDate := time.Date(
		p.CreatedAt.Year(),
		p.CreatedAt.Month(),
		p.CreatedAt.Day(),
		waterTime.Hour(),
		waterTime.Minute(),
		waterTime.Second(),
		0,
		waterTime.Location(),
	)

	// Schedule the WaterAction execution
	action := p.WateringAction()
	_, err = pr.scheduler.
		Every(duration).
		StartAt(startDate).
		Tag(p.ID.String()).
		Do(func() {
			influxdbClient := influxdb.NewClient(pr.config.InfluxDBConfig)
			defer influxdbClient.Close()

			logger.Infof("Executing WateringAction to water Plant %s for %d ms", p.ID.String(), action.Duration)
			err = action.Execute(g, p, pr.mqttClient, influxdbClient)
			if err != nil {
				logger.Error("Error executing scheduled plant watering action: ", err)
			}
			err = pr.storageClient.SavePlant(p)
			if err != nil {
				logger.Error("Error saving plant after watering: ", err)
			}
		})
	return err
}

// removeWateringSchedule is used to remove the Plant's scheduled watering Job
// from the scheduler.
func (pr PlantsResource) removeWateringSchedule(p *pkg.Plant) error {
	return pr.scheduler.RemoveByTag(p.ID.String())
}

// resetWateringSchedule will simply remove the existing Job and create a new one
func (pr PlantsResource) resetWateringSchedule(g *pkg.Garden, p *pkg.Plant) error {
	if err := pr.removeWateringSchedule(p); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	return pr.addWateringSchedule(g, p)
}

// getNextWateringTime determines the next scheduled watering time for a given Plant using tags
func (pr PlantsResource) getNextWateringTime(p *pkg.Plant) *time.Time {
	for _, job := range pr.scheduler.Jobs() {
		for _, tag := range job.Tags() {
			if tag == p.ID.String() {
				result := job.NextRun()
				return &result
			}
		}
	}
	return nil
}
