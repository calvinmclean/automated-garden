package action

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/mqtt"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/storage"
	"github.com/go-co-op/gocron"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

const (
	lightInterval = 24 * time.Hour
	adhocTag      = "ADHOC"
)

// Scheduler exposes scheduling functionality that allows executing actions at predetermined times and intervals
type Scheduler interface {
	ScheduleWaterAction(*logrus.Entry, *pkg.Garden, *pkg.Zone) error
	ResetWaterSchedule(*logrus.Entry, *pkg.Garden, *pkg.Zone) error
	GetNextWaterTime(*logrus.Entry, *pkg.Zone) *time.Time

	ScheduleLightActions(*logrus.Entry, *pkg.Garden) error
	ResetLightSchedule(*logrus.Entry, *pkg.Garden) error
	GetNextLightTime(*logrus.Entry, *pkg.Garden, pkg.LightState) *time.Time
	ScheduleLightDelay(*logrus.Entry, *pkg.Garden, *LightAction) error

	RemoveJobsByID(*logrus.Entry, xid.ID) error

	// Client accessors
	StorageClient() storage.Client
	InfluxDBClient() influxdb.Client
	MQTTClient() mqtt.Client

	// Functions from the gocron Scheduler
	StartAsync()
	Stop()
}

// scheduler is capable of executing actions at predetermined times and intervals
type scheduler struct {
	*gocron.Scheduler
	storageClient  storage.Client
	influxdbClient influxdb.Client
	mqttClient     mqtt.Client
}

// NewScheduler creates a Scheduler from the
func NewScheduler(storageClient storage.Client, influxdbClient influxdb.Client, mqttClient mqtt.Client) Scheduler {
	return &scheduler{
		Scheduler:      gocron.NewScheduler(time.Local),
		storageClient:  storageClient,
		influxdbClient: influxdbClient,
		mqttClient:     mqttClient,
	}
}

func (s *scheduler) StorageClient() storage.Client {
	return s.storageClient
}
func (s *scheduler) InfluxDBClient() influxdb.Client {
	return s.influxdbClient
}
func (s *scheduler) MQTTClient() mqtt.Client {
	return s.mqttClient
}

// ScheduleWaterAction will schedule water actions for the Zone based off the CreatedAt date,
// WaterSchedule time, and Interval. The scheduled Job is tagged with the Zone's ID so it can
// easily be removed
func (s *scheduler) ScheduleWaterAction(logger *logrus.Entry, g *pkg.Garden, z *pkg.Zone) error {
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
	action := WaterAction{Duration: duration.Milliseconds()}
	_, err = s.Scheduler.
		Every(interval).
		StartAt(*z.WaterSchedule.StartTime).
		Tag(z.ID.String()).
		Do(func(jobLogger *logrus.Entry) {
			defer s.influxdbClient.Close()

			jobLogger.Infof("executing WaterAction for %d ms", action.Duration)
			err = action.Execute(g, z, s)
			if err != nil {
				jobLogger.Errorf("error executing scheduled zone water action: %v", err)
			}
		}, logrus.New().WithFields(logrus.Fields{
			"garden_id": g.ID,
			"zone_id":   z.ID,
		}))
	return err
}

// ResetWaterSchedule will simply remove the existing Job and create a new one
func (s *scheduler) ResetWaterSchedule(logger *logrus.Entry, g *pkg.Garden, z *pkg.Zone) error {
	logger.Debugf("resetting WaterSchedule")

	if err := s.RemoveJobsByID(logger, z.ID); err != nil {
		return err
	}
	return s.ScheduleWaterAction(logger, g, z)
}

// GetNextWaterTime determines the next scheduled watering time for a given Zone using tags
func (s *scheduler) GetNextWaterTime(logger *logrus.Entry, z *pkg.Zone) *time.Time {
	logger.Debugf("getting next water time")

	for _, job := range s.Scheduler.Jobs() {
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
func (s *scheduler) ScheduleLightActions(logger *logrus.Entry, g *pkg.Garden) error {
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

	executeLightAction := func(action *LightAction, actionLogger *logrus.Entry) {
		actionLogger = actionLogger.WithField("state", action.State.String())
		actionLogger.Infof("executing LightAction with state %s", action.State)
		err = action.Execute(g, nil, s)
		if err != nil {
			actionLogger.Errorf("error executing scheduled LightAction: %v", err)
		}
	}

	actionLogger := logrus.New().WithFields(logrus.Fields{
		"garden_id": g.ID,
	})
	// Schedule the LightAction execution for ON and OFF
	onAction := &LightAction{State: pkg.LightStateOn}
	offAction := &LightAction{State: pkg.LightStateOff}
	_, err = s.Scheduler.
		Every(lightInterval).
		StartAt(startDate).
		Tag(g.ID.String()).
		Tag(pkg.LightStateOn.String()).
		Do(executeLightAction, onAction, actionLogger)
	if err != nil {
		return err
	}
	_, err = s.Scheduler.
		Every(lightInterval).
		StartAt(startDate.Add(duration)).
		Tag(g.ID.String()).
		Tag(pkg.LightStateOff.String()).
		Do(executeLightAction, offAction, actionLogger)
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
			return s.storageClient.SaveGarden(g)
		}

		nextOnJob, err := s.getNextLightJob(logger, g, pkg.LightStateOn, false)
		if err != nil {
			return err
		}

		// If nextOnTime is before AdhocOnTime, delay it by 24 hours
		nextOnTime := nextOnJob.NextRun()
		logger.Debugf("garden's next ON time is %v", nextOnTime)
		if nextOnTime.Before(*g.LightSchedule.AdhocOnTime) {
			logger.Debug("next ON time is before the adhoc time, so delaying it by 24 hours")
			_, err = s.Job(nextOnJob).StartAt(nextOnTime.Add(24 * time.Hour)).Update()
			if err != nil {
				return err
			}
		}

		// Schedule one-time watering
		if err = s.scheduleAdhocLightAction(logger, g); err != nil {
			return err
		}
		logger.Debug("successfully scheduled adhoc ON time")
	}
	return nil
}

// ResetLightSchedule will simply remove the existing Job and create a new one
func (s *scheduler) ResetLightSchedule(logger *logrus.Entry, g *pkg.Garden) error {
	logger.Debug("resetting LightSchedule")

	if err := s.RemoveJobsByID(logger, g.ID); err != nil {
		return err
	}
	return s.ScheduleLightActions(logger, g)
}

// GetNextLightTime returns the next time that the Garden's light will be turned to the specified state
func (s *scheduler) GetNextLightTime(logger *logrus.Entry, g *pkg.Garden, state pkg.LightState) *time.Time {
	logger.Debugf("getting next light time for state %s", state.String())

	nextJob, err := s.getNextLightJob(logger, g, state, true)
	if err != nil {
		return nil
	}
	nextRun := nextJob.NextRun()
	return &nextRun
}

// ScheduleLightDelay handles a LightAction that requests delaying turning a light on
func (s *scheduler) ScheduleLightDelay(logger *logrus.Entry, g *pkg.Garden, action *LightAction) error {
	if logger == nil {
		logger = s.contextLogger(g, nil)
	}
	logger.Infof("scheduling light delay: %+v", *action)

	// Only allow when action state is OFF
	if action.State != pkg.LightStateOff {
		return errors.New("unable to use delay when state is not OFF")
	}

	// Read delay Duration string into a time.Duration
	delayDuration, err := time.ParseDuration(action.ForDuration)
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

	nextOnTime := s.GetNextLightTime(logger, g, pkg.LightStateOn)
	if nextOnTime == nil {
		return errors.New("unable to get next light-on time")
	}
	logger.Debugf("found next ON time %v", *nextOnTime)

	nextOffTime := s.GetNextLightTime(logger, g, pkg.LightStateOff)
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

		nextOnJob, err := s.getNextLightJob(logger, g, pkg.LightStateOn, false)
		if err != nil {
			return err
		}
		logger.Debug("found next ON Job and rescheduling in 24 hours")

		// Delay the original ON Job for 24 hours
		_, err = s.Job(nextOnJob).StartAt(nextOnJob.NextRun().Add(24 * time.Hour)).Update()
		if err != nil {
			return err
		}

		// Add new ON schedule with action.Light.ForDuration that executes once
		adhocTime = nextOnTime.Add(delayDuration)
	}
	logger.Debugf("saving adhoc on time to Garden: %v", adhocTime)

	// Add new lightSchedule with AdhocTime and Save Garden
	g.LightSchedule.AdhocOnTime = &adhocTime
	err = s.scheduleAdhocLightAction(logger, g)
	if err != nil {
		return err
	}

	return s.storageClient.SaveGarden(g)
}

// RemoveJobsByID will remove Jobs tagged with the specific xid
func (s *scheduler) RemoveJobsByID(logger *logrus.Entry, id xid.ID) error {
	if err := s.Scheduler.RemoveByTags(id.String()); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	return nil
}

// getNextLightJob returns the next Job tagged with the gardenID and state. If allowAdhoc is true, return whichever job is soonest,
// otherwise return the first non-adhoc Job
func (s *scheduler) getNextLightJob(logger *logrus.Entry, g *pkg.Garden, state pkg.LightState, allowAdhoc bool) (*gocron.Job, error) {
	logger.Debugf("getting next light Job for state %s, allowAdhoc=%t", state, allowAdhoc)

	sort.Sort(s.Scheduler)
	jobs, err := s.FindJobsByTag(g.ID.String(), state.String())
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
func (s *scheduler) scheduleAdhocLightAction(logger *logrus.Entry, g *pkg.Garden) error {
	logger.Infof("creating one-time scheduled Job for lighting Garden")

	if g.LightSchedule.AdhocOnTime == nil {
		return errors.New("unable to schedule adhoc light schedule without LightSchedule.AdhocOnTime")
	}

	// Remove existing adhoc Jobs for this Garden
	if err := s.Scheduler.RemoveByTags(g.ID.String(), adhocTag); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	logger.Debug("removed existing adhoc light Jobs")

	executeLightAction := func(action *LightAction, actionLogger *logrus.Entry) {
		actionLogger = actionLogger.WithFields(logrus.Fields{
			"state": action.State.String(),
			"adhoc": "true",
		})
		actionLogger.Infof("executing adhoc LightAction with state %s", action.State)
		err := action.Execute(g, nil, s)
		if err != nil {
			actionLogger.Errorf("error executing scheduled adhoc LightAction: %v", err)
		}
		actionLogger.Debug("removing AdhocOnTime")
		// Now set AdhocOnTime to nil and save
		g.LightSchedule.AdhocOnTime = nil
		err = s.storageClient.SaveGarden(g)
		if err != nil {
			actionLogger.Errorf("error saving Garden after removing AdhocOnTime: %v", err)
		}
	}

	actionLogger := logrus.New().WithFields(logrus.Fields{
		"garden_id": g.ID,
	})

	// Schedule the LightAction execution for ON and OFF
	onAction := &LightAction{State: pkg.LightStateOn}
	_, err := s.Scheduler.
		Every("1s"). // Every is required even though it's not needed for this Job
		LimitRunsTo(1).
		StartAt(*g.LightSchedule.AdhocOnTime).
		WaitForSchedule().
		Tag(g.ID.String()).
		Tag(pkg.LightStateOn.String()).
		Tag(adhocTag).
		Do(executeLightAction, onAction, actionLogger)

	return err
}

func (s *scheduler) contextLogger(g *pkg.Garden, z *pkg.Zone) *logrus.Entry {
	fields := logrus.Fields{}
	if g != nil {
		fields["garden_id"] = g.ID.String()
	}
	if z != nil {
		fields["zone_id"] = z.ID.String()
	}
	return logrus.New().WithFields(fields)
}
