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
	adhocTag = "ADHOC"
)

// ScheduleWaterAction will schedule water actions for the Zone based off the CreatedAt date,
// WaterSchedule time, and Interval. The scheduled Job is tagged with the Zone's ID so it can
// easily be removed
func (w *Worker) ScheduleWaterAction(ws *pkg.WaterSchedule) error {
	logger := w.contextLogger(nil, nil, ws)
	logger.Infof("creating scheduled Job for WaterSchedule: %+v", *ws)

	// Schedule the WaterAction execution
	scheduleJobsGauge.WithLabelValues(waterScheduleLabels(ws)...).Inc()
	_, err := ws.Interval.SchedulerFunc(w.scheduler).
		StartAt(*ws.StartTime).
		Tag("water_schedule").
		Tag(ws.ID.String()).
		Do(func(jobLogger *logrus.Entry) {
			if !ws.IsActive() {
				jobLogger.Infof("skipping WaterSchedule %q because current time is outside of ActivePeriod: %+v", ws.ID, *ws.ActivePeriod)
				return
			}

			zonesAndGardens, err := w.storageClient.GetZonesUsingWaterSchedule(ws.ID)

			if err != nil {
				jobLogger.Errorf("error getting Zones for WaterSchedule when executing scheduled Job: %v", err)
				schedulerErrors.WithLabelValues(waterScheduleLabels(ws)...).Inc()
				return
			}

			for _, zg := range zonesAndGardens {
				err = w.ExecuteScheduledWaterAction(zg.Garden, zg.Zone, ws)
				if err != nil {
					jobLogger.WithField("zone_id", zg.Zone.ID.String()).Errorf("error executing scheduled water action: %v", err)
					schedulerErrors.WithLabelValues(zoneLabels(zg.Zone)...).Inc()
				}
			}
		}, logger.WithField("source", "scheduled_job"))
	return err
}

// ResetWaterSchedule will simply remove the existing Job and create a new one
func (w *Worker) ResetWaterSchedule(ws *pkg.WaterSchedule) error {
	logger := w.contextLogger(nil, nil, ws)
	logger.Debugf("resetting WaterSchedule")

	if err := w.RemoveJobsByID(ws.ID); err != nil {
		return err
	}
	return w.ScheduleWaterAction(ws)
}

// GetNextWaterSchedule determines the WaterSchedule that is going to be used for the next watering time
func (w *Worker) GetNextWaterSchedule(waterSchedules []*pkg.WaterSchedule) *pkg.WaterSchedule {
	w.logger.Debugf("getting next water schedule for water_schedules: %+v", waterSchedules)

	type nextRunData struct {
		ws      *pkg.WaterSchedule
		nextRun *time.Time
	}

	nextRuns := []nextRunData{}
	for _, ws := range waterSchedules {
		if !ws.IsActive() {
			continue
		}
		nextRuns = append(nextRuns, nextRunData{
			ws:      ws,
			nextRun: w.GetNextWaterTime(ws),
		})
	}

	if len(nextRuns) == 0 {
		return nil
	}

	nextRun := nextRuns[0]
	for i := 1; i < len(nextRuns); i++ {
		if nextRuns[i].nextRun == nil {
			continue
		}
		if nextRuns[i].nextRun.Before(*nextRun.nextRun) {
			nextRun = nextRuns[i]
		}
	}

	return nextRun.ws
}

// GetNextWaterTime determines the next scheduled watering time for a given Zone using tags
func (w *Worker) GetNextWaterTime(ws *pkg.WaterSchedule) *time.Time {
	if ws == nil {
		return nil
	}

	logger := w.contextLogger(nil, nil, ws)
	logger.Debugf("getting next water time for water_schedule")

	for _, job := range w.scheduler.Jobs() {
		for _, tag := range job.Tags() {
			if tag == ws.ID.String() {
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
	logger := w.contextLogger(g, nil, nil)
	logger.Infof("creating scheduled Jobs for lighting Garden: %+v", *g.LightSchedule)

	// Parse Gardens's LightSchedule.Time (has no "date")
	lightTime, err := time.Parse(pkg.LightTimeFormat, g.LightSchedule.StartTime)
	if err != nil {
		return err
	}

	now := time.Now().In(lightTime.Location())
	// Create onStartDate using the CreatedAt date with the WaterSchedule's timestamp
	onStartDate := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		lightTime.Hour(),
		lightTime.Minute(),
		lightTime.Second(),
		0,
		lightTime.Location(),
	)
	offStartDate := onStartDate.Add(g.LightSchedule.Duration.Duration)

	// Schedule the LightAction execution for ON and OFF
	scheduleJobsGauge.WithLabelValues(gardenLabels(g)...).Add(2)
	onAction := &action.LightAction{State: pkg.LightStateOn}
	offAction := &action.LightAction{State: pkg.LightStateOff}
	_, err = w.scheduler.
		Every(1).Day().At(onStartDate).
		StartAt(onStartDate).
		Tag("garden").
		Tag(g.ID.String()).
		Tag(pkg.LightStateOn.String()).
		Do(w.executeLightActionInScheduledJob, g, onAction, logger.WithField("source", "scheduled_job"))
	if err != nil {
		return err
	}

	_, err = w.scheduler.
		Every(1).Day().At(offStartDate).
		StartAt(offStartDate).
		Tag("garden").
		Tag(g.ID.String()).
		Tag(pkg.LightStateOff.String()).
		Do(w.executeLightActionInScheduledJob, g, offAction, logger.WithField("source", "scheduled_job"))
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
			err = w.rescheduleLightOn(nextOnJob, g, nextOnTime.Add(24*time.Hour), logger)
			if err != nil {
				return fmt.Errorf("error rescheduling next on job: %w", err)
			}
			if err != nil {
				return err
			}
		}

		// Schedule one-time watering
		if err = w.scheduleAdhocLightAction(g); err != nil {
			return fmt.Errorf("error scheduling ad-hoc light action: %w", err)
		}
		logger.Debug("successfully scheduled adhoc ON time")
	}
	return nil
}

// ResetLightSchedule will simply remove the existing Job and create a new one
func (w *Worker) ResetLightSchedule(g *pkg.Garden) error {
	logger := w.contextLogger(g, nil, nil)
	logger.Debug("resetting LightSchedule")

	if err := w.RemoveJobsByID(g.ID); err != nil {
		return err
	}
	return w.ScheduleLightActions(g)
}

// GetNextLightTime returns the next time that the Garden's light will be turned to the specified state
func (w *Worker) GetNextLightTime(g *pkg.Garden, state pkg.LightState) *time.Time {
	logger := w.contextLogger(g, nil, nil)
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
	logger := w.contextLogger(g, nil, nil)
	logger.Infof("scheduling light delay: %+v", *input)

	// Only allow when action state is OFF
	if input.State != pkg.LightStateOff {
		return errors.New("unable to use delay when state is not OFF")
	}

	// Don't allow delaying longer than LightSchedule.Duration
	if input.ForDuration.Duration > g.LightSchedule.Duration.Duration {
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
		logger.Debugf("next OFF time is before next ON time; setting schedule to turn light back on in %v", input.ForDuration.Duration)
		now := time.Now()

		// Don't allow a delayDuration that will occur after nextOffTime
		if nextOffTime.Before(now.Add(input.ForDuration.Duration)) {
			return errors.New("unable to schedule delay that extends past the light turning back on")
		}

		adhocTime = now.Add(input.ForDuration.Duration)
	} else {
		// If nextOffTime is after nextOnTime, then light was not ON yet and we need to reschedule the regular ON time
		// and schedule nextOnTime + delay
		logger.Debugf("next OFF time is after next ON time; delaying next ON time by %v", input.ForDuration.Duration)

		nextOnJob, err := w.getNextLightJob(g, pkg.LightStateOn, false)
		if err != nil {
			return err
		}
		logger.Debug("found next ON Job and rescheduling in 24 hours")

		err = w.rescheduleLightOn(nextOnJob, g, nextOnJob.NextRun().Add(24*time.Hour), logger)
		if err != nil {
			return fmt.Errorf("error rescheduling next on job: %w", err)
		}

		// Add new ON schedule with action.Light.ForDuration that executes once
		adhocTime = nextOnTime.Add(input.ForDuration.Duration)
	}
	logger.Debugf("saving adhoc on time to Garden: %v", adhocTime)

	// Add new lightSchedule with AdhocTime and Save Garden
	g.LightSchedule.AdhocOnTime = &adhocTime
	err := w.scheduleAdhocLightAction(g)
	if err != nil {
		return fmt.Errorf("error scheduling ad-hoc light action: %w", err)
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
	logger := w.contextLogger(g, nil, nil)
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
		isAdhoc := false
		for _, tag := range j.Tags() {
			if tag == adhocTag {
				isAdhoc = true
				break
			}
		}
		if !isAdhoc {
			return j, nil
		}
	}
	return nil, fmt.Errorf("unable to find next %s Job for Garden %s", state.String(), g.ID.String())
}

// scheduleAdhocLightAction schedules a one-time action to turn a light on based on the LightSchedule.AdhocOnTime
func (w *Worker) scheduleAdhocLightAction(g *pkg.Garden) error {
	logger := w.contextLogger(g, nil, nil)
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
		Every(1).Day(). // Every is required even though it's not needed for this Job
		At(*g.LightSchedule.AdhocOnTime).
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

func (w *Worker) contextLogger(g *pkg.Garden, z *pkg.Zone, ws *pkg.WaterSchedule) *logrus.Entry {
	fields := logrus.Fields{}
	if g != nil {
		fields["garden_id"] = g.ID.String()
	}
	if z != nil {
		fields["zone_id"] = z.ID.String()
	}
	if ws != nil {
		fields["water_schedule_id"] = ws.ID.String()
	}
	return w.logger.WithFields(fields)
}

func zoneLabels(z *pkg.Zone) []string {
	return []string{"zone", z.ID.String()}
}

func gardenLabels(g *pkg.Garden) []string {
	return []string{"garden", g.ID.String()}
}

func waterScheduleLabels(ws *pkg.WaterSchedule) []string {
	return []string{"water_schedule", ws.ID.String()}
}

func (w *Worker) executeLightActionInScheduledJob(g *pkg.Garden, input *action.LightAction, actionLogger *logrus.Entry) {
	actionLogger = actionLogger.WithField("state", input.State.String())
	actionLogger.Infof("executing LightAction with state %s", input.State)
	err := w.ExecuteLightAction(g, input)
	if err != nil {
		actionLogger.Errorf("error executing scheduled LightAction: %v", err)
		schedulerErrors.WithLabelValues(gardenLabels(g)...).Inc()
	}
}

// TODO: go back to using Update() function when bug is fixed
func (w *Worker) rescheduleLightOn(nextJob *gocron.Job, g *pkg.Garden, newTime time.Time, logger *logrus.Entry) error {
	onAction := &action.LightAction{State: pkg.LightStateOn}
	_, err := w.scheduler.
		Every(1).Day().
		StartAt(newTime).
		Tag("garden").
		Tag(g.ID.String()).
		Tag(pkg.LightStateOn.String()).
		Do(w.executeLightActionInScheduledJob, g, onAction, logger.WithField("source", "scheduled_job"))
	if err != nil {
		return fmt.Errorf("unable to create new LightOn job: %w", err)
	}
	w.scheduler.RemoveByReference(nextJob)

	return nil
}
