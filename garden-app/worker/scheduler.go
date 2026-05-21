package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/go-co-op/gocron"
)

const (
	lightInterval = 24 * time.Hour
)

// ScheduleWaterAction will schedule water actions for the Zone based off the CreatedAt date,
// WaterSchedule time, and Interval. The scheduled Job is tagged with the Zone's ID so it can
// easily be removed
func (w *Worker) ScheduleWaterAction(waterSchedule *pkg.WaterSchedule) error {
	logger := w.contextLogger(nil, nil, waterSchedule)
	logger.Info("creating scheduled Job for WaterSchedule")

	startDate := clock.Now()
	if waterSchedule.StartDate != nil {
		startDate = waterSchedule.StartDate.ToTimeInLocation(waterSchedule.StartTime.Time.Location())
	}

	// Schedule the WaterAction execution
	scheduleJobsGauge.WithLabelValues(waterScheduleLabels(waterSchedule)...).Inc()
	_, err := waterSchedule.Interval.SchedulerFunc(w.scheduler).
		StartAt(waterSchedule.StartTime.OnDate(startDate).UTC()).
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

				// Calculate duration for weather control (for notifications and zone watering)
				duration := ws.Duration.Duration
				if ws.HasWeatherControl() {
					duration, _ = w.ScaleWateringDuration(ws)
				}

				// Get zones using this WaterSchedule (for notifications and watering)
				zonesAndGardens, err := w.storageClient.GetZonesUsingWaterSchedule(ws.ID.String())
				if err != nil {
					return fmt.Errorf("error getting Zones for WaterSchedule when executing scheduled Job: %w", err)
				}

				// Send watering notification if enabled
				w.sendWateringReminder(ws, duration, len(zonesAndGardens), jobLogger)

				// If duration is 0 (weather says skip), don't water any zones
				if duration == 0 {
					jobLogger.Info("skipping watering all zones due to weather control")
					return nil
				}

				for _, zg := range zonesAndGardens {
					err = w.ExecuteScheduledWaterAction(zg.Garden, zg.Zone, ws, duration)
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
			if tag != ws.ID.String() {
				continue
			}

			result := job.NextRun()
			for !ws.IsActive(result) {
				result = result.Add(ws.Interval.Duration)
			}
			return &result
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

	now := clock.Now()
	onStartDate := g.LightSchedule.StartTime.OnDate(now).UTC()
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
		Tag("light").
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
		Tag("light").
		Tag(pkg.LightStateOff.String()).
		Do(w.executeLightActionInScheduledJob, g, offAction, logger.With("source", "scheduled_job"))
	if err != nil {
		return err
	}

	return nil
}

// ResetLightSchedule will simply remove the existing Job and create a new one
func (w *Worker) ResetLightSchedule(g *pkg.Garden) error {
	logger := w.contextLogger(g, nil, nil)
	logger.Debug("resetting LightSchedule")

	if err := w.RemoveJobsByTag(g.ID.String(), "light"); err != nil {
		return err
	}
	return w.ScheduleLightActions(g)
}

// RemoveJobsByID will remove Jobs tagged with the specific xid
func (w *Worker) RemoveJobsByID(id string) error {
	return w.RemoveJobsByTag(id)
}

// RemoveJobsByTag will remove Jobs tagged with the specific tags. All tags must match for a job to be removed.
func (w *Worker) RemoveJobsByTag(tag string, extraTags ...string) error {
	jobs, err := w.scheduler.FindJobsByTag(tag)
	if err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	// Remove Jobs from metric
	for _, j := range jobs {
		hasAll := true
		for _, et := range extraTags {
			if !slices.Contains(j.Tags(), et) {
				hasAll = false
				break
			}
		}
		if hasAll {
			scheduleJobsGauge.WithLabelValues(j.Tags()[0:2]...).Dec()
		}
	}
	tags := append([]string{tag}, extraTags...)
	if err := w.scheduler.RemoveByTags(tags...); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	return nil
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

// ScheduleFanActions will schedule FanActions to turn the fan on based off the FanSchedule's
// active time, off time, and interval. The scheduled Jobs are tagged with the Garden's ID so they can
// easily be removed
func (w *Worker) ScheduleFanActions(g *pkg.Garden) error {
	logger := w.contextLogger(g, nil, nil)
	logger.Info("creating scheduled Job for fan Garden", "fan_schedule", *g.FanSchedule)

	now := clock.Now()
	startDate := now.UTC()
	if g.FanSchedule.StartTime != nil {
		startDate = g.FanSchedule.StartTime.OnDate(now).UTC()
	}

	scheduleJobsGauge.WithLabelValues(gardenLabels(g)...).Inc()
	_, err := w.scheduler.
		Every(g.FanSchedule.Interval.Duration).
		StartAt(startDate).
		Tag("garden").
		Tag(g.ID.String()).
		Tag("fan").
		Do(w.executeFanActionInScheduledJob, g, logger.With("source", "scheduled_job"))
	return err
}

// ResetFanSchedule will simply remove the existing Job and create a new one
func (w *Worker) ResetFanSchedule(g *pkg.Garden) error {
	logger := w.contextLogger(g, nil, nil)
	logger.Debug("resetting FanSchedule")

	if err := w.RemoveJobsByTag(g.ID.String(), "fan"); err != nil {
		return err
	}
	return w.ScheduleFanActions(g)
}

func (w *Worker) executeFanActionInScheduledJob(g *pkg.Garden, actionLogger *slog.Logger) {
	actionLogger.Info("executing FanAction")

	if g.FanSchedule.OnlyWithLight && g.LightSchedule != nil {
		if g.LightSchedule.ExpectedStateAtTime(clock.Now()) != pkg.LightStateOn {
			actionLogger.Info("skipping FanAction because LightSchedule is not active")
			return
		}
	}

	if g.GetNotificationClientID() != "" {
		w.sendDownNotification(g, g.GetNotificationClientID(), "Fan")
	}

	input := &action.FanAction{
		Duration: g.FanSchedule.ActiveTime.Duration.Milliseconds(),
		Power:    g.FanSchedule.PowerToPWM(),
	}

	err := w.ExecuteFanAction(g, input)
	if err != nil {
		actionLogger.Error("error executing scheduled FanAction", "error", err)
		schedulerErrors.WithLabelValues(gardenLabels(g)...).Inc()

		if g.GetNotificationClientID() != "" {
			w.sendNotification(g.GetNotificationClientID(), fmt.Sprintf("%s: Fan Action Error", g.Name), err.Error(), actionLogger)
		}
		return
	}

	w.sendFanActionNotification(g, actionLogger)
}
