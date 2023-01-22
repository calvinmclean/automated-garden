package worker

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/go-co-op/gocron"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

const (
	lightInterval = 24 * time.Hour
	adhocTag      = "ADHOC"
)

// ScheduleWaterAction will schedule water actions for the Zone based off the CreatedAt date,
// WaterSchedule time, and Interval. The scheduled Job is tagged with the Zone's ID so it can
// easily be removed
func (w *Worker) ScheduleWaterAction(g *pkg.Garden, z *pkg.Zone) error {
	logger := w.contextLogger(g, z)
	logger.Infof("creating scheduled Job for watering Zone: %+v", *z.WaterSchedule)

	// Read Zone's Interval string into a Duration
	interval, err := time.ParseDuration(z.WaterSchedule.Interval)
	if err != nil {
		return err
	}

	// Read Zone's Interval string into a Duration
	duration, err := time.ParseDuration(z.WaterSchedule.Duration)
	if err != nil {
		return err
	}

	// Schedule the WaterAction execution
	scheduleJobsGauge.WithLabelValues(zoneLabels(z)...).Inc()
	waterAction := &action.WaterAction{Duration: duration.Milliseconds()}
	_, err = w.scheduler.
		Every(interval).
		StartAt(*z.WaterSchedule.StartTime).
		Tag("zone").
		Tag(z.ID.String()).
		Do(func(jobLogger *logrus.Entry) {
			defer w.influxdbClient.Close()

			schedulerErrors.WithLabelValues(zoneLabels(z)...).Inc()

			jobLogger.Infof("executing WaterAction for %d ms", waterAction.Duration)
			err = w.ExecuteWaterAction(g, z, waterAction)
			if err != nil {
				jobLogger.Errorf("error executing scheduled zone water action: %v", err)
				schedulerErrors.WithLabelValues(zoneLabels(z)...).Inc()
			}
		}, logger.WithField("source", "scheduled_job"))
	return err
}

// ResetWaterSchedule will simply remove the existing Job and create a new one
func (w *Worker) ResetWaterSchedule(g *pkg.Garden, z *pkg.Zone) error {
	logger := w.contextLogger(g, z)
	logger.Debugf("resetting WaterSchedule")

	if err := w.RemoveJobsByID(z.ID); err != nil {
		return err
	}
	return w.ScheduleWaterAction(g, z)
}

// GetNextWaterTime determines the next scheduled watering time for a given Zone using tags
func (w *Worker) GetNextWaterTime(z *pkg.Zone) *time.Time {
	logger := w.contextLogger(nil, z)
	logger.WithField("zone_id", z.ID).Debugf("getting next water time for zone with ID")

	for _, job := range w.scheduler.Jobs() {
		for _, tag := range job.Tags() {
			if tag == z.ID.String() {
				result := job.NextRun()
				return &result
			}
		}
	}
	return nil
}

// ScheduleLightActions will schedule LightActions to turn the light on and off based off the CreatedAt date,
// LightSchedule time, and Interval. The scheduled Jobs are tagged with the Garden's ID so they can
// easily be removed
func (w *Worker) ScheduleLightActions(g *pkg.Garden) error {
	logger := w.contextLogger(g, nil)
	logger.Infof("creating scheduled Jobs for lighting Garden: %+v", *g.LightSchedule)

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

	executeLightAction := func(input *action.LightAction, actionLogger *logrus.Entry) {
		actionLogger = actionLogger.WithField("state", input.State.String())
		actionLogger.Infof("executing LightAction with state %s", input.State)
		err = w.ExecuteLightAction(g, input)
		if err != nil {
			actionLogger.Errorf("error executing scheduled LightAction: %v", err)
			schedulerErrors.WithLabelValues(gardenLabels(g)...).Inc()
		}
	}

	// Schedule the LightAction execution for ON and OFF
	scheduleJobsGauge.WithLabelValues(gardenLabels(g)...).Add(2)
	onAction := &action.LightAction{State: pkg.LightStateOn}
	offAction := &action.LightAction{State: pkg.LightStateOff}
	_, err = w.scheduler.
		Every(lightInterval).
		StartAt(startDate).
		Tag("garden").
		Tag(g.ID.String()).
		Tag(pkg.LightStateOn.String()).
		Do(executeLightAction, onAction, logger.WithField("source", "scheduled_job"))
	if err != nil {
		return err
	}
	_, err = w.scheduler.
		Every(lightInterval).
		StartAt(startDate.Add(duration)).
		Tag("garden").
		Tag(g.ID.String()).
		Tag(pkg.LightStateOff.String()).
		Do(executeLightAction, offAction, logger.WithField("source", "scheduled_job"))
	if err != nil {
		return err
	}

	// If AdhocOnTime is defined (and is in the future), schedule it
	if g.LightSchedule.AdhocOnTime != nil {
		logger.Debugf("garden has adhoc ON time at %v", g.LightSchedule.AdhocOnTime)
		// If AdhocOnTime is in the past, reset it and return
		if g.LightSchedule.AdhocOnTime.Before(time.Now()) {
			logger.Debug("adhoc ON time is in the past and is being removed")
			g.LightSchedule.AdhocOnTime = nil
			return w.storageClient.SaveGarden(g)
		}

		nextOnJob, err := w.getNextLightJob(g, pkg.LightStateOn, false)
		if err != nil {
			return err
		}

		// If nextOnTime is before AdhocOnTime, delay it by 24 hours
		nextOnTime := nextOnJob.NextRun()
		logger.Debugf("garden's next ON time is %v", nextOnTime)
		if nextOnTime.Before(*g.LightSchedule.AdhocOnTime) {
			logger.Debug("next ON time is before the adhoc time, so delaying it by 24 hours")
			_, err = w.scheduler.Job(nextOnJob).StartAt(nextOnTime.Add(24 * time.Hour)).Update()
			if err != nil {
				return err
			}
		}

		// Schedule one-time watering
		if err = w.scheduleAdhocLightAction(g); err != nil {
			return err
		}
		logger.Debug("successfully scheduled adhoc ON time")
	}
	return nil
}

// ResetLightSchedule will simply remove the existing Job and create a new one
func (w *Worker) ResetLightSchedule(g *pkg.Garden) error {
	logger := w.contextLogger(g, nil)
	logger.Debug("resetting LightSchedule")

	if err := w.RemoveJobsByID(g.ID); err != nil {
		return err
	}
	return w.ScheduleLightActions(g)
}

// GetNextLightTime returns the next time that the Garden's light will be turned to the specified state
func (w *Worker) GetNextLightTime(g *pkg.Garden, state pkg.LightState) *time.Time {
	logger := w.contextLogger(g, nil)
	logger.Debugf("getting next light time for state %s", state.String())

	nextJob, err := w.getNextLightJob(g, state, true)
	if err != nil {
		return nil
	}
	nextRun := nextJob.NextRun()
	return &nextRun
}

// ScheduleLightDelay handles a LightAction that requests delaying turning a light on
func (w *Worker) ScheduleLightDelay(g *pkg.Garden, input *action.LightAction) error {
	logger := w.contextLogger(g, nil)
	logger.Infof("scheduling light delay: %+v", *input)

	// Only allow when action state is OFF
	if input.State != pkg.LightStateOff {
		return errors.New("unable to use delay when state is not OFF")
	}

	// Read delay Duration string into a time.Duration
	delayDuration, err := time.ParseDuration(input.ForDuration)
	if err != nil {
		return err
	}

	lightScheduleDuration, err := time.ParseDuration(g.LightSchedule.Duration)
	if err != nil {
		return err
	}

	// Don't allow delaying longer than LightSchedule.Duration
	if delayDuration > lightScheduleDuration {
		return errors.New("unable to execute delay that lasts longer than light_schedule")
	}

	nextOnTime := w.GetNextLightTime(g, pkg.LightStateOn)
	if nextOnTime == nil {
		return errors.New("unable to get next light-on time")
	}
	logger.Debugf("found next ON time %v", *nextOnTime)

	nextOffTime := w.GetNextLightTime(g, pkg.LightStateOff)
	if nextOffTime == nil {
		return errors.New("unable to get next light-off time")
	}
	logger.Debugf("found next OFF time %v", *nextOffTime)

	var adhocTime time.Time

	// If nextOffTime is before nextOnTime, then the light was probably ON and we need to schedule now + delay to turn back on.
	// No need to change any schedules
	if nextOffTime.Before(*nextOnTime) {
		logger.Debugf("next OFF time is before next ON time; setting schedule to turn light back on in %v", delayDuration)
		now := time.Now()

		// Don't allow a delayDuration that will occur after nextOffTime
		if nextOffTime.Before(now.Add(delayDuration)) {
			return errors.New("unable to schedule delay that extends past the light turning back on")
		}

		adhocTime = now.Add(delayDuration)
	} else {
		// If nextOffTime is after nextOnTime, then light was not ON yet and we need to reschedule the regular ON time
		// and schedule nextOnTime + delay
		logger.Debugf("next OFF time is after next ON time; delaying next ON time by %v", delayDuration)

		nextOnJob, err := w.getNextLightJob(g, pkg.LightStateOn, false)
		if err != nil {
			return err
		}
		logger.Debug("found next ON Job and rescheduling in 24 hours")

		// Delay the original ON Job for 24 hours
		_, err = w.scheduler.Job(nextOnJob).StartAt(nextOnJob.NextRun().Add(24 * time.Hour)).Update()
		if err != nil {
			return err
		}

		// Add new ON schedule with action.Light.ForDuration that executes once
		adhocTime = nextOnTime.Add(delayDuration)
	}
	logger.Debugf("saving adhoc on time to Garden: %v", adhocTime)

	// Add new lightSchedule with AdhocTime and Save Garden
	g.LightSchedule.AdhocOnTime = &adhocTime
	err = w.scheduleAdhocLightAction(g)
	if err != nil {
		return err
	}

	return w.storageClient.SaveGarden(g)
}

// RemoveJobsByID will remove Jobs tagged with the specific xid
func (w *Worker) RemoveJobsByID(id xid.ID) error {
	jobs, err := w.scheduler.FindJobsByTag(id.String())
	if err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	// Remove Jobs from metric
	for _, j := range jobs {
		scheduleJobsGauge.WithLabelValues(j.Tags()[0:2]...).Dec()
	}
	if err := w.scheduler.RemoveByTags(id.String()); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	return nil
}

// getNextLightJob returns the next Job tagged with the gardenID and state. If allowAdhoc is true, return whichever job is soonest,
// otherwise return the first non-adhoc Job
func (w *Worker) getNextLightJob(g *pkg.Garden, state pkg.LightState, allowAdhoc bool) (*gocron.Job, error) {
	logger := w.contextLogger(g, nil)
	logger.Debugf("getting next light Job for state %s, allowAdhoc=%t", state, allowAdhoc)

	sort.Sort(w.scheduler)
	jobs, err := w.scheduler.FindJobsByTag(g.ID.String(), state.String())
	if err != nil {
		return nil, err
	}

	if allowAdhoc {
		logger.Debugf("found %d light jobs, returning the first one", len(jobs))
		return jobs[0], nil
	}

	logger.Debugf("found %d light jobs and now checking to remove any adhoc jobs", len(jobs))
	for _, j := range jobs {
		for _, tag := range j.Tags() {
			if tag == adhocTag {
				continue
			}
			return j, nil
		}
	}
	return nil, fmt.Errorf("unable to find next %s Job for Garden %s", state.String(), g.ID.String())
}

// scheduleAdhocLightAction schedules a one-time action to turn a light on based on the LightSchedule.AdhocOnTime
func (w *Worker) scheduleAdhocLightAction(g *pkg.Garden) error {
	logger := w.contextLogger(g, nil)
	logger.Infof("creating one-time scheduled Job for lighting Garden")

	if g.LightSchedule.AdhocOnTime == nil {
		return errors.New("unable to schedule adhoc light schedule without LightSchedule.AdhocOnTime")
	}

	// Remove existing adhoc Jobs for this Garden
	if err := w.scheduler.RemoveByTags(g.ID.String(), adhocTag); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	logger.Debug("removed existing adhoc light Jobs")

	executeLightAction := func(a *action.LightAction, actionLogger *logrus.Entry) {
		scheduleJobsGauge.WithLabelValues(gardenLabels(g)...).Dec()

		actionLogger = actionLogger.WithFields(logrus.Fields{
			"state": a.State.String(),
			"adhoc": "true",
		})
		actionLogger.Infof("executing adhoc LightAction with state %s", a.State)
		err := w.ExecuteLightAction(g, a)
		if err != nil {
			actionLogger.Errorf("error executing scheduled adhoc LightAction: %v", err)
		}
		actionLogger.Debug("removing AdhocOnTime")
		// Now set AdhocOnTime to nil and save
		g.LightSchedule.AdhocOnTime = nil
		err = w.storageClient.SaveGarden(g)
		if err != nil {
			actionLogger.Errorf("error saving Garden after removing AdhocOnTime: %v", err)
			schedulerErrors.WithLabelValues(gardenLabels(g)...).Inc()
		}
	}

	// Schedule the LightAction execution for ON and OFF
	scheduleJobsGauge.WithLabelValues(gardenLabels(g)...).Inc()
	onAction := &action.LightAction{State: pkg.LightStateOn}
	_, err := w.scheduler.
		Every("1s"). // Every is required even though it's not needed for this Job
		LimitRunsTo(1).
		StartAt(*g.LightSchedule.AdhocOnTime).
		WaitForSchedule().
		Tag("garden").
		Tag(g.ID.String()).
		Tag(pkg.LightStateOn.String()).
		Tag(adhocTag).
		Do(executeLightAction, onAction, logger.WithField("source", "scheduled_job"))

	return err
}

func (w *Worker) contextLogger(g *pkg.Garden, z *pkg.Zone) *logrus.Entry {
	fields := logrus.Fields{}
	if g != nil {
		fields["garden_id"] = g.ID.String()
	}
	if z != nil {
		fields["zone_id"] = z.ID.String()
	}
	return w.logger.WithFields(fields)
}

func zoneLabels(z *pkg.Zone) []string {
	return []string{"zone", z.ID.String()}
}

func gardenLabels(g *pkg.Garden) []string {
	return []string{"garden", g.ID.String()}
}
