package server

import (
	"errors"
	"fmt"
	"sort"
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

	// Schedule the WaterAction execution
	action := p.WateringAction()
	_, err = pr.scheduler.
		Every(duration).
		StartAt(*p.WaterSchedule.StartTime).
		Tag(p.ID.String()).
		Do(func() {
			defer pr.influxdbClient.Close()

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
				return &result
			}
		}
	}
	return nil
}

// getNextLightTime returns the next time that the Garden's light will be turned to the specified state
func (gr GardensResource) getNextLightTime(g *pkg.Garden, state string) *time.Time {
	sort.Sort(gr.scheduler)
	for _, job := range gr.scheduler.Jobs() {
		matchedID := false
		matchedState := false
		for _, tag := range job.Tags() {
			if tag == g.ID.String() {
				matchedID = true
			}
			if tag == fmt.Sprintf("%s-%s", g.ID.String(), state) {
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

// scheduleLightActions will schedule LightActions to turn the light on and off based off the CreatedAt date,
// LightingSchedule time, and Interval. The scheduled Jobs are tagged with the Garden's ID so they can
// easily be removed
func (gr GardensResource) scheduleLightActions(g *pkg.Garden) error {
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
		Tag(g.ID.String(), fmt.Sprintf("%s-%s", g.ID.String(), pkg.StateOn)).
		Do(executeLightAction, onAction)
	if err != nil {
		return err
	}
	_, err = gr.scheduler.
		Every(lightingInterval).
		StartAt(startDate.Add(duration)).
		Tag(g.ID.String(), fmt.Sprintf("%s-%s", g.ID.String(), pkg.StateOff)).
		Do(executeLightAction, offAction)
	if err != nil {
		return err
	}

	// If AdhocOnTime is defined (and is in the future), schedule it
	if g.LightSchedule.AdhocOnTime != nil {
		// If AdhocOnTime is in the past, reset it and return
		if g.LightSchedule.AdhocOnTime.Before(time.Now()) {
			g.LightSchedule.AdhocOnTime = nil
			return gr.storageClient.SaveGarden(g)
		}

		// If nextOnTime is before AdhocOnTime, remove it
		nextOnTime := gr.getNextLightTime(g, pkg.StateOn)
		if nextOnTime.Before(*g.LightSchedule.AdhocOnTime) {
			if err := gr.scheduler.RemoveByTag(fmt.Sprintf("%s-%s", g.ID.String(), pkg.StateOn)); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
				return err
			}
		}

		// Schedule one-time watering
		if err = gr.scheduleAdhocLightAction(g); err != nil {
			return err
		}
	}
	return nil
}

// scheduleAdhocLightAction schedules a one-time action to turn a light on based on the LightSchedule.AdhocOnTime
func (gr GardensResource) scheduleAdhocLightAction(g *pkg.Garden) error {
	if g.LightSchedule.AdhocOnTime == nil {
		return errors.New("unable to schedule adhoc light schedule without LightSchedule.AdhocOnTime")
	}
	logger.Infof("Creating one-time scheduled Job for lighting Garden %s", g.ID.String())

	executeLightAction := func(action *pkg.LightAction) {
		logger.Infof("Executing LightAction for Garden %s with state %s", g.ID.String(), action.State)
		err := action.Execute(g, gr.mqttClient)
		if err != nil {
			logger.Error("Error executing scheduled LightAction: ", err)
		}
		// Now set AdhocOnTime to nil and save
		g.LightSchedule.AdhocOnTime = nil
		err = gr.storageClient.SaveGarden(g)
		if err != nil {
			logger.Error("Error saving Garden after removing AdhocOnTime: ", err)
		}
	}

	// Schedule the LightAction execution for ON and OFF
	onAction := &pkg.LightAction{State: pkg.StateOn}
	_, err := gr.scheduler.
		Every("1m"). // Every is required even though it's not needed for this Job
		LimitRunsTo(1).
		StartAt(*g.LightSchedule.AdhocOnTime).
		WaitForSchedule().
		Tag(g.ID.String(), fmt.Sprintf("%s-%s", g.ID.String(), pkg.StateOn), "ADHOC").
		Do(executeLightAction, onAction)

	return err
}

// rescheduleLightOnAction is used to reschedule only the ON action after it is removed as part of adhoc light delay
func (gr GardensResource) rescheduleLightOnAction(g *pkg.Garden) error {
	logger.Infof("Creating new scheduled Jobs for turning ON Garden %s", g.ID.String())

	// Parse Gardens's LightSchedule.Time (has no "date")
	lightTime, err := time.Parse(pkg.LightTimeFormat, g.LightSchedule.StartTime)
	if err != nil {
		return err
	}

	// Create startDate using 24 hours from today with the WaterSchedule's timestamp
	now := time.Now()
	startDate := time.Date(
		now.Year(),
		now.Month(),
		now.Day()+1,
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
	_, err = gr.scheduler.
		Every(lightingInterval).
		StartAt(startDate).
		Tag(g.ID.String(), fmt.Sprintf("%s-%s", g.ID.String(), pkg.StateOn)).
		Do(executeLightAction, onAction)
	return err
}

// resetWateringSchedule will simply remove the existing Job and create a new one
func (gr GardensResource) resetLightingSchedule(g *pkg.Garden) error {
	if err := gr.scheduler.RemoveByTag(g.ID.String()); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	return gr.scheduleLightActions(g)
}
