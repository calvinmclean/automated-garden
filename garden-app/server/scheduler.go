package server

import (
	"errors"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/go-co-op/gocron"
)

const (
	lightingInterval = 24 * time.Hour
)

// addWateringSchedule will schedule watering actions for the Plant based off the CreatedAt date,
// WaterSchedule time, and Interval. The scheduled Job is tagged with the Plant's ID so it can
// easily be removed
func (pr PlantsResource) addWateringSchedule(g *pkg.Garden, p *pkg.Plant) error {
	logger.Infof("Creating scheduled Job for watering Plant %s", p.ID.String())

	// Read Plant's Interval string into a Duration
	duration, err := time.ParseDuration(p.WaterSchedule.Interval)
	if err != nil {
		return err
	}

	// Parse Plant's WaterSchedule.Time (has no "date")
	waterTime, err := time.Parse(pkg.WaterTimeFormat, p.WaterSchedule.StartTime)
	if err != nil {
		return err
	}

	// Create startDate using the CreatedAt date with the WaterSchedule's timestamp
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
			defer pr.influxdbClient.Close()

			if p.SkipCount != nil && *p.SkipCount > 0 {
				*p.SkipCount--

				err = pr.storageClient.SavePlant(p)
				if err != nil {
					logger.Error("Error saving plant after watering: ", err)
				}
				return
			}

			logger.Infof("Executing WateringAction to water Plant %s for %d ms", p.ID.String(), action.Duration)
			err = action.Execute(g, p, pr.mqttClient, pr.influxdbClient)
			if err != nil {
				logger.Error("Error executing scheduled plant watering action: ", err)
			}
		})
	return err
}

// resetWateringSchedule will simply remove the existing Job and create a new one
func (pr PlantsResource) resetWateringSchedule(g *pkg.Garden, p *pkg.Plant) error {
	if err := pr.scheduler.RemoveByTag(p.ID.String()); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
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
				if p.SkipCount != nil {
					interval, _ := time.ParseDuration(p.WaterSchedule.Interval)
					for i := uint(0); i < *p.SkipCount; i++ {
						result = result.Add(interval)
					}
				}
				return &result
			}
		}
	}
	return nil
}

// getNextLightOnTime returns the next time that the Garden's light will be turned to the specified state
func (gr GardensResource) getNextLightTime(g *pkg.Garden, state string) *time.Time {
	for _, job := range gr.scheduler.Jobs() {
		matchedID := false
		matchedState := false
		for _, tag := range job.Tags() {
			if tag == g.ID.String() {
				matchedID = true
			}
			if tag == state {
				matchedState = true
			}
		}
		if matchedID && matchedState {
			result := job.NextRun()
			return &result
		}
	}
	return nil
}

// addLightSchedule will schedule LightActions to turn the light on and off based off the CreatedAt date,
// LightingSchedule time, and Interval. The scheduled Jobs are tagged with the Garden's ID so they can
// easily be removed
func (gr GardensResource) addLightSchedule(g *pkg.Garden) error {
	logger.Infof("Creating scheduled Jobs for lighting Garden %s", g.ID.String())

	// Read Garden's Duration string into a time.Duration
	duration, err := time.ParseDuration(g.LightSchedule.Duration)
	if err != nil {
		return err
	}

	// Parse Gardens's LightSchedule.Time (has no "date")
	lightTime, err := time.Parse(pkg.LightTimeFormat, g.LightSchedule.StartTime)
	if err != nil {
		return err
	}

	// Create startDate using the CreatedAt date with the WaterSchedule's timestamp
	startDate := time.Date(
		g.CreatedAt.Year(),
		g.CreatedAt.Month(),
		g.CreatedAt.Day(),
		lightTime.Hour(),
		lightTime.Minute(),
		lightTime.Second(),
		0,
		lightTime.Location(),
	)

	executeLightAction := func(action *pkg.LightAction) {
		logger.Infof("Executing LightAction for Garden %s with state %s", g.ID.String(), action.State)
		err = action.Execute(g, gr.mqttClient)
		if err != nil {
			logger.Error("Error executing scheduled LightAction: ", err)
		}
	}

	// Schedule the LightAction execution for ON and OFF
	onAction := &pkg.LightAction{State: pkg.StateOn}
	offAction := &pkg.LightAction{State: pkg.StateOff}
	_, err = gr.scheduler.
		Every(lightingInterval).
		StartAt(startDate).
		Tag(g.ID.String(), pkg.StateOn).
		Do(executeLightAction, onAction)
	_, err = gr.scheduler.
		Every(lightingInterval).
		StartAt(startDate.Add(duration)).
		Tag(g.ID.String(), pkg.StateOff).
		Do(executeLightAction, offAction)
	return err
}

// resetWateringSchedule will simply remove the existing Job and create a new one
func (gr GardensResource) resetLightingSchedule(g *pkg.Garden) error {
	if err := gr.scheduler.RemoveByTag(g.ID.String()); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	return gr.addLightSchedule(g)
}
