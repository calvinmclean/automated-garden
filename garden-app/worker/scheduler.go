package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/go-co-op/gocron"
)

const (
	lightInterval = 24 * time.Hour
	adhocTag      = "ADHOC"
)

// sortableJobs is a type that makes a slice of gocron Jobs sortable
type sortableJobs []*gocron.Job

func (jobs sortableJobs) Len() int {
	return len(jobs)
}

func (jobs sortableJobs) Less(i, j int) bool {
	return jobs[i].NextRun().Before(jobs[j].NextRun())
}

// Swap swaps the elements with indexes i and j.
func (jobs sortableJobs) Swap(i, j int) {
	jobs[i], jobs[j] = jobs[j], jobs[i]
}

// ScheduleWaterAction will schedule water actions for the Zone based off the CreatedAt date,
// WaterSchedule time, and Interval. The scheduled Job is tagged with the Zone's ID so it can
// easily be removed
func (w *Worker) ScheduleWaterAction(waterSchedule *pkg.WaterSchedule) error {
	logger := w.contextLogger(nil, nil, waterSchedule)
	logger.Info("creating scheduled Job for WaterSchedule")

	startTime := waterSchedule.StartTime.Time.UTC()

	// Schedule the WaterAction execution
	scheduleJobsGauge.WithLabelValues(waterScheduleLabels(waterSchedule)...).Inc()
	_, err := waterSchedule.Interval.SchedulerFunc(w.scheduler).
		StartAt(timeAtDate(waterSchedule.StartDate, startTime)).
		Tag("water_schedule").
		Tag(waterSchedule.ID.String()).
		Do(func(jobLogger *slog.Logger) {
			err := func() error {
				// Get WaterSchedule from storage in case the ActivePeriod or WeatherControl are changed
				ws, err := w.storageClient.WaterSchedules.Get(context.Background(), waterSchedule.ID.String())
				if err != nil {
					return fmt.Errorf("error getting WaterSchedule when executing scheduled Job: %w", err)
				}
				if ws == nil {
					return errors.New("WaterSchedule not found")
				}

				if !ws.IsActive(clock.Now()) {
					jobLogger.Info("skipping WaterSchedule because current time is outside of ActivePeriod", "active_period", *ws.ActivePeriod)
					return nil
				}

				zonesAndGardens, err := w.storageClient.GetZonesUsingWaterSchedule(ws.ID.String())
				if err != nil {
					return fmt.Errorf("error getting Zones for WaterSchedule when executing scheduled Job: %w", err)
				}

				for _, zg := range zonesAndGardens {
					err = w.ExecuteScheduledWaterAction(zg.Garden, zg.Zone, ws)
					if err != nil {
						jobLogger.Error("error executing scheduled water action", "error", err, "zone_id", zg.Zone.ID.String())
						schedulerErrors.WithLabelValues(zoneLabels(zg.Zone)...).Inc()
						if ws.GetNotificationClientID() != "" {
							go w.sendNotification(
								ws.GetNotificationClientID(),
								fmt.Sprintf("%s: Water Action Error", ws.Name),
								err.Error(),
								jobLogger,
							)
						}
					}
				}
				return nil
			}()
			if err != nil {
				jobLogger.Error("error executing schedule WaterAction", "error", err)
				schedulerErrors.WithLabelValues(waterScheduleLabels(waterSchedule)...).Inc()
				if waterSchedule.GetNotificationClientID() != "" {
					w.sendNotification(
						waterSchedule.GetNotificationClientID(),
						fmt.Sprintf("%s: Water Action Error", waterSchedule.Name),
						err.Error(),
						jobLogger,
					)
				}
			}
		}, logger.With("source", "scheduled_job"))
	return err
}

// ResetWaterSchedule will simply remove the existing Job and create a new one
func (w *Worker) ResetWaterSchedule(ws *pkg.WaterSchedule) error {
	logger := w.contextLogger(nil, nil, ws)
	logger.Debug("resetting WaterSchedule")

	if err := w.RemoveJobsByID(ws.ID.String()); err != nil {
		return err
	}
	return w.ScheduleWaterAction(ws)
}

// GetNextActiveWaterSchedule determines the WaterSchedule that is going to be used for the next watering time
func (w *Worker) GetNextActiveWaterSchedule(waterSchedules []*pkg.WaterSchedule) *pkg.WaterSchedule {
	w.logger.Debug("getting next water schedule for water_schedules", "water_schedules", waterSchedules)

	type nextRunData struct {
		ws      *pkg.WaterSchedule
		nextRun *time.Time
	}

	nextRuns := []nextRunData{}
	for _, ws := range waterSchedules {
		nextWaterTime := w.GetNextWaterTime(ws)
		if nextWaterTime == nil {
			continue
		}

		if !ws.IsActive(*nextWaterTime) {
			continue
		}

		nextRuns = append(nextRuns, nextRunData{
			ws:      ws,
			nextRun: nextWaterTime,
		})
	}

	if len(nextRuns) == 0 {
		return nil
	}

	nextRun := nextRuns[0]
	for i := 1; i < len(nextRuns); i++ {
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
	logger.Debug("getting next water time for water_schedule")

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
	logger.Info("creating scheduled Jobs for lighting Garden", "light_schedule", *g.LightSchedule)

	lightTime := g.LightSchedule.StartTime.Time.UTC()

	now := clock.Now()
	onStartDate := timeAtDate(&now, lightTime)
	offStartDate := onStartDate.Add(g.LightSchedule.Duration.Duration)

	// Schedule the LightAction execution for ON and OFF
	scheduleJobsGauge.WithLabelValues(gardenLabels(g)...).Add(2)
	onAction := &action.LightAction{State: pkg.LightStateOn}
	offAction := &action.LightAction{State: pkg.LightStateOff}
	_, err := w.scheduler.
		Every(lightInterval).
		StartAt(onStartDate).
		Tag("garden").
		Tag(g.ID.String()).
		Tag(pkg.LightStateOn.String()).
		Do(w.executeLightActionInScheduledJob, g, onAction, logger.With("source", "scheduled_job"))
	if err != nil {
		return err
	}

	_, err = w.scheduler.
		Every(lightInterval).
		StartAt(offStartDate).
		Tag("garden").
		Tag(g.ID.String()).
		Tag(pkg.LightStateOff.String()).
		Do(w.executeLightActionInScheduledJob, g, offAction, logger.With("source", "scheduled_job"))
	if err != nil {
		return err
	}

	// If AdhocOnTime is defined (and is in the future), schedule it
	if g.LightSchedule.AdhocOnTime != nil {
		logger.Debug("garden has adhoc ON time", "adhoc_on_time", g.LightSchedule.AdhocOnTime)
		// If AdhocOnTime is in the past, reset it and return
		if g.LightSchedule.AdhocOnTime.Before(clock.Now().UTC()) {
			logger.Debug("adhoc ON time is in the past and is being removed")
			g.LightSchedule.AdhocOnTime = nil
			return w.storageClient.Gardens.Set(context.Background(), g)
		}

		nextOnJob, err := w.getNextLightJob(g, pkg.LightStateOn, false)
		if err != nil {
			return err
		}

		// If nextOnTime is before AdhocOnTime, delay it by 24 hours
		nextOnTime := nextOnJob.NextRun()
		logger.Debug("garden's next ON time", "next_on_time", nextOnTime)
		if nextOnTime.Before(*g.LightSchedule.AdhocOnTime) {
			logger.Debug("next ON time is before the adhoc time, so delaying it by 24 hours")
			_, err = w.scheduler.Job(nextOnJob).StartAt(nextOnTime.Add(lightInterval)).Update()
			if err != nil {
				return fmt.Errorf("error rescheduling next on job: %w", err)
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

	if err := w.RemoveJobsByID(g.ID.String()); err != nil {
		return err
	}
	return w.ScheduleLightActions(g)
}

// GetNextLightTime returns the next time that the Garden's light will be turned to the specified state
func (w *Worker) GetNextLightTime(g *pkg.Garden, state pkg.LightState) *time.Time {
	logger := w.contextLogger(g, nil, nil)
	logger.Debug("getting next light time for state", "state", state.String())

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
	logger.Info("scheduling light delay", "input", *input)

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
	logger.Debug("found next ON time", "next_on_time", *nextOnTime)

	nextOffTime := w.GetNextLightTime(g, pkg.LightStateOff)
	if nextOffTime == nil {
		return errors.New("unable to get next light-off time")
	}
	logger.Debug("found next OFF time", "next_off_time", *nextOffTime)

	var adhocTime time.Time

	// If nextOffTime is before nextOnTime, then the light was probably ON and we need to schedule now + delay to turn back on.
	// No need to change any schedules
	if nextOffTime.Before(*nextOnTime) {
		logger.Debug("next OFF time is before next ON time; setting schedule to turn light back on", "duration", input.ForDuration.Duration)
		now := clock.Now().UTC()

		// Don't allow a delayDuration that will occur after nextOffTime
		if nextOffTime.Before(now.Add(input.ForDuration.Duration)) {
			return errors.New("unable to schedule delay that extends past the light turning back on")
		}

		adhocTime = now.Add(input.ForDuration.Duration)
	} else {
		// If nextOffTime is after nextOnTime, then light was not ON yet and we need to reschedule the regular ON time
		// and schedule nextOnTime + delay
		logger.Debug("next OFF time is after next ON time; delaying next ON time", "duration", input.ForDuration.Duration)

		nextOnJob, err := w.getNextLightJob(g, pkg.LightStateOn, false)
		if err != nil {
			return err
		}
		logger.Debug("found next ON Job and rescheduling in 24 hours")

		_, err = w.scheduler.Job(nextOnJob).StartAt(nextOnJob.NextRun().Add(lightInterval)).Update()
		if err != nil {
			return fmt.Errorf("error rescheduling next on job: %w", err)
		}

		// Add new ON schedule with action.Light.ForDuration that executes once
		adhocTime = nextOnTime.Add(input.ForDuration.Duration)
	}
	logger.Debug("saving adhoc on time to Garden: %v", "adhoc_on_time", adhocTime)

	// Add new lightSchedule with AdhocTime and Save Garden
	g.LightSchedule.AdhocOnTime = &adhocTime
	err := w.scheduleAdhocLightAction(g)
	if err != nil {
		return fmt.Errorf("error scheduling ad-hoc light action: %w", err)
	}

	return w.storageClient.Gardens.Set(context.Background(), g)
}

// RemoveJobsByID will remove Jobs tagged with the specific xid
func (w *Worker) RemoveJobsByID(id string) error {
	jobs, err := w.scheduler.FindJobsByTag(id)
	if err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	// Remove Jobs from metric
	for _, j := range jobs {
		scheduleJobsGauge.WithLabelValues(j.Tags()[0:2]...).Dec()
	}
	if err := w.scheduler.RemoveByTags(id); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	return nil
}

// getNextLightJob returns the next Job tagged with the gardenID and state. If allowAdhoc is true, return whichever job is soonest,
// otherwise return the first non-adhoc Job
func (w *Worker) getNextLightJob(g *pkg.Garden, state pkg.LightState, allowAdhoc bool) (*gocron.Job, error) {
	logger := w.contextLogger(g, nil, nil)
	logger.Debug("getting next light Job for state", "state", state, "allow_ad_hoc", allowAdhoc)

	jobs, err := w.scheduler.FindJobsByTag(g.ID.String(), state.String())
	if err != nil {
		return nil, err
	}
	sort.Sort(sortableJobs(jobs))

	if allowAdhoc {
		logger.Debug("found light jobs, returning the first one", "count", len(jobs))
		return jobs[0], nil
	}

	logger.Debug("found light jobs and now checking to remove any adhoc jobs", "count", len(jobs))
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
	logger.Info("creating one-time scheduled Job for lighting Garden")

	if g.LightSchedule.AdhocOnTime == nil {
		return errors.New("unable to schedule adhoc light schedule without LightSchedule.AdhocOnTime")
	}

	// Remove existing adhoc Jobs for this Garden
	if err := w.scheduler.RemoveByTags(g.ID.String(), adhocTag); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	logger.Debug("removed existing adhoc light Jobs")

	executeLightAction := func(a *action.LightAction, actionLogger *slog.Logger) {
		scheduleJobsGauge.WithLabelValues(gardenLabels(g)...).Dec()

		actionLogger = actionLogger.With(
			"state", a.State.String(),
			"adhoc", "true",
		)
		actionLogger.Info("executing adhoc LightAction with state")
		err := w.ExecuteLightAction(g, a)
		if err != nil {
			actionLogger.Error("error executing scheduled adhoc LightAction", "error", err)
		}

		w.sendLightActionNotification(g, a.State, actionLogger)

		actionLogger.Debug("removing AdhocOnTime")
		// Now set AdhocOnTime to nil and save
		g.LightSchedule.AdhocOnTime = nil
		err = w.storageClient.Gardens.Set(context.Background(), g)
		if err != nil {
			actionLogger.Error("error saving Garden after removing AdhocOnTime", "error", err)
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
		Do(executeLightAction, onAction, logger.With("source", "scheduled_job"))

	return err
}

func (w *Worker) contextLogger(g *pkg.Garden, z *pkg.Zone, ws *pkg.WaterSchedule) *slog.Logger {
	logger := w.logger.With()
	if g != nil {
		logger = logger.With("garden_id", g.ID.String())
	}
	if z != nil {
		logger = logger.With("zone_id", z.ID.String())
	}
	if ws != nil {
		logger = logger.With("water_schedule_id", ws.ID.String())
	}
	return logger
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

func (w *Worker) executeLightActionInScheduledJob(g *pkg.Garden, input *action.LightAction, actionLogger *slog.Logger) {
	actionLogger = actionLogger.With("state", input.State.String())
	actionLogger.Info("executing LightAction")

	if g.GetNotificationClientID() != "" {
		w.sendDownNotification(g, g.GetNotificationClientID(), "Light")
	}

	err := w.ExecuteLightAction(g, input)
	if err != nil {
		actionLogger.Error("error executing scheduled LightAction", "error", err)
		schedulerErrors.WithLabelValues(gardenLabels(g)...).Inc()

		if g.GetNotificationClientID() != "" {
			w.sendNotification(g.GetNotificationClientID(), fmt.Sprintf("%s: Light Action Error", g.Name), err.Error(), actionLogger)
		}
		return
	}

	w.sendLightActionNotification(g, input.State, actionLogger)
}

func timeAtDate(date *time.Time, startTime time.Time) time.Time {
	actualDate := clock.Now()
	if date != nil {
		actualDate = *date
	}
	actualDate = actualDate.In(startTime.Location())
	return time.Date(
		actualDate.Year(),
		actualDate.Month(),
		actualDate.Day(),
		startTime.Hour(),
		startTime.Minute(),
		startTime.Second(),
		0,
		startTime.Location(),
	)
}
