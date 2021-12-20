package action

import (
	"errors"
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
	lightingInterval = 24 * time.Hour
)

// Scheduler exposes scheduling functionality that allows executing actions at predetermined times and intervals
type Scheduler interface {
	ScheduleWateringAction(*pkg.Garden, *pkg.Plant) error
	ResetWateringSchedule(*pkg.Garden, *pkg.Plant) error
	GetNextWateringTime(*pkg.Plant) *time.Time

	ScheduleLightActions(*pkg.Garden) error
	ResetLightingSchedule(*pkg.Garden) error
	GetNextLightTime(*pkg.Garden, pkg.LightState) *time.Time
	ScheduleLightDelay(*pkg.Garden, *LightAction) error

	RemoveJobsByID(xid.ID) error

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
	logger         *logrus.Logger
}

// NewScheduler creates a Scheduler from the
func NewScheduler(storageClient storage.Client, influxdbClient influxdb.Client, mqttClient mqtt.Client, logger *logrus.Logger) Scheduler {
	return &scheduler{
		Scheduler:      gocron.NewScheduler(time.Local),
		storageClient:  storageClient,
		influxdbClient: influxdbClient,
		mqttClient:     mqttClient,
		logger:         logger,
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

// ScheduleWateringAction will schedule watering actions for the Plant based off the CreatedAt date,
// WaterSchedule time, and Interval. The scheduled Job is tagged with the Plant's ID so it can
// easily be removed
func (s *scheduler) ScheduleWateringAction(g *pkg.Garden, p *pkg.Plant) error {
	s.logger.Infof("Creating scheduled Job for watering Plant %s", p.ID.String())

	// Read Plant's Interval string into a Duration
	duration, err := time.ParseDuration(p.WaterSchedule.Interval)
	if err != nil {
		return err
	}

	// Schedule the WaterAction execution
	action := WaterAction{Duration: duration.Milliseconds()}
	_, err = s.Scheduler.
		Every(duration).
		StartAt(*p.WaterSchedule.StartTime).
		Tag(p.ID.String()).
		Do(func() {
			defer s.influxdbClient.Close()

			s.logger.Infof("Executing WateringAction to water Plant %s for %d ms", p.ID.String(), action.Duration)
			err = action.Execute(g, p, s.mqttClient, s.influxdbClient)
			if err != nil {
				s.logger.Error("Error executing scheduled plant watering action: ", err)
			}
		})
	return err
}

// ResetWateringSchedule will simply remove the existing Job and create a new one
func (s *scheduler) ResetWateringSchedule(g *pkg.Garden, p *pkg.Plant) error {
	if err := s.RemoveJobsByID(g.ID); err != nil {
		return err
	}
	return s.ScheduleWateringAction(g, p)
}

// GetNextWateringTime determines the next scheduled watering time for a given Plant using tags
func (s *scheduler) GetNextWateringTime(p *pkg.Plant) *time.Time {
	for _, job := range s.Scheduler.Jobs() {
		for _, tag := range job.Tags() {
			if tag == p.ID.String() {
				result := job.NextRun()
				return &result
			}
		}
	}
	return nil
}

// ScheduleLightActions will schedule LightActions to turn the light on and off based off the CreatedAt date,
// LightingSchedule time, and Interval. The scheduled Jobs are tagged with the Garden's ID so they can
// easily be removed
func (s *scheduler) ScheduleLightActions(g *pkg.Garden) error {
	s.logger.Infof("Creating scheduled Jobs for lighting Garden %s", g.ID.String())

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

	executeLightAction := func(action *LightAction) {
		s.logger.Infof("Executing LightAction for Garden %s with state %s", g.ID.String(), action.State)
		err = action.Execute(g, s)
		if err != nil {
			s.logger.Error("Error executing scheduled LightAction: ", err)
		}
	}

	// Schedule the LightAction execution for ON and OFF
	onAction := &LightAction{State: pkg.LightStateOn}
	offAction := &LightAction{State: pkg.LightStateOff}
	_, err = s.Scheduler.
		Every(lightingInterval).
		StartAt(startDate).
		Tag(g.ID.String()).
		Tag(pkg.LightStateOn.String()).
		Do(executeLightAction, onAction)
	if err != nil {
		return err
	}
	_, err = s.Scheduler.
		Every(lightingInterval).
		StartAt(startDate.Add(duration)).
		Tag(g.ID.String()).
		Tag(pkg.LightStateOff.String()).
		Do(executeLightAction, offAction)
	if err != nil {
		return err
	}

	// If AdhocOnTime is defined (and is in the future), schedule it
	if g.LightSchedule.AdhocOnTime != nil {
		// If AdhocOnTime is in the past, reset it and return
		if g.LightSchedule.AdhocOnTime.Before(time.Now()) {
			g.LightSchedule.AdhocOnTime = nil
			return s.storageClient.SaveGarden(g)
		}

		// If nextOnTime is before AdhocOnTime, remove it
		nextOnTime := s.GetNextLightTime(g, pkg.LightStateOn)
		if nextOnTime.Before(*g.LightSchedule.AdhocOnTime) {
			if err := s.removeLightScheduleByState(g, pkg.LightStateOn.String()); err != nil {
				return err
			}
		}

		// Schedule one-time watering
		if err = s.scheduleAdhocLightAction(g); err != nil {
			return err
		}
	}
	return nil
}

// ResetLightingSchedule will simply remove the existing Job and create a new one
func (s *scheduler) ResetLightingSchedule(g *pkg.Garden) error {
	if err := s.RemoveJobsByID(g.ID); err != nil {
		return err
	}
	return s.ScheduleLightActions(g)
}

// GetNextLightTime returns the next time that the Garden's light will be turned to the specified state
func (s *scheduler) GetNextLightTime(g *pkg.Garden, state pkg.LightState) *time.Time {
	sort.Sort(s.Scheduler)
	for _, job := range s.Scheduler.Jobs() {
		matchedID := false
		matchedState := false
		for _, tag := range job.Tags() {
			if tag == g.ID.String() {
				matchedID = true
			}
			if tag == state.String() {
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

// ScheduleLightDelay handles a LightAction that requests delaying turning a light on
func (s *scheduler) ScheduleLightDelay(g *pkg.Garden, action *LightAction) error {
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

	nextOnTime := s.GetNextLightTime(g, pkg.LightStateOn)
	nextOffTime := s.GetNextLightTime(g, pkg.LightStateOff)

	var adhocTime time.Time

	// If nextOffTime is before nextOnTime, then the light was probably ON and we need to schedule now + delay to turn back on. No need to delete any schedules
	if nextOffTime.Before(*nextOnTime) {
		now := time.Now()

		// Don't allow a delayDuration that will occur after nextOffTime
		if nextOffTime.Before(now.Add(delayDuration)) {
			return errors.New("unable to schedule delay that extends past the light turning back on")
		}

		adhocTime = now.Add(delayDuration)
	} else {
		// If nextOffTime is after nextOnTime, then light was not ON yet and we need to delete nextOnTime and schedule nextOnTime + delay. Then we need to reschedule the regular ON time
		// Delete existing ON schedule
		if err := s.removeLightScheduleByState(g, pkg.LightStateOn.String()); err != nil {
			return err
		}

		// Add new ON schedule with action.Light.ForDuration that executes once
		adhocTime = nextOnTime.Add(delayDuration)

		// Add new regular ON schedule starting 24 hours from today's Date + g.LightSchedule.StartTime
		err = s.rescheduleLightOnAction(g)
		if err != nil {
			return err
		}
	}

	// Add new lightSchedule with AdhocTime and Save Garden
	g.LightSchedule.AdhocOnTime = &adhocTime
	err = s.scheduleAdhocLightAction(g)
	if err != nil {
		return err
	}
	return s.storageClient.SaveGarden(g)
}

// RemoveJobsByID will remove Jobs tagged with the specific xid
func (s *scheduler) RemoveJobsByID(id xid.ID) error {
	if err := s.Scheduler.RemoveByTags(id.String()); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	return nil
}

// scheduleAdhocLightAction schedules a one-time action to turn a light on based on the LightSchedule.AdhocOnTime
func (s *scheduler) scheduleAdhocLightAction(g *pkg.Garden) error {
	if g.LightSchedule.AdhocOnTime == nil {
		return errors.New("unable to schedule adhoc light schedule without LightSchedule.AdhocOnTime")
	}
	s.logger.Infof("Creating one-time scheduled Job for lighting Garden %s", g.ID.String())

	executeLightAction := func(action *LightAction) {
		s.logger.Infof("Executing LightAction for Garden %s with state %s", g.ID.String(), action.State)
		err := action.Execute(g, s)
		if err != nil {
			s.logger.Error("Error executing scheduled LightAction: ", err)
		}
		// Now set AdhocOnTime to nil and save
		g.LightSchedule.AdhocOnTime = nil
		err = s.storageClient.SaveGarden(g)
		if err != nil {
			s.logger.Error("Error saving Garden after removing AdhocOnTime: ", err)
		}
	}

	// Schedule the LightAction execution for ON and OFF
	onAction := &LightAction{State: pkg.LightStateOn}
	_, err := s.Scheduler.
		Every("1m"). // Every is required even though it's not needed for this Job
		LimitRunsTo(1).
		StartAt(*g.LightSchedule.AdhocOnTime).
		WaitForSchedule().
		Tag(g.ID.String()).
		Tag(pkg.LightStateOn.String()).
		Tag("ADHOC").
		Do(executeLightAction, onAction)

	return err
}

// rescheduleLightOnAction is used to reschedule only the ON action after it is removed as part of adhoc light delay
func (s *scheduler) rescheduleLightOnAction(g *pkg.Garden) error {
	s.logger.Infof("Creating new scheduled Jobs for turning ON Garden %s", g.ID.String())

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

	executeLightAction := func(action *LightAction) {
		s.logger.Infof("Executing LightAction for Garden %s with state %s", g.ID.String(), action.State)
		err = action.Execute(g, s)
		if err != nil {
			s.logger.Error("Error executing scheduled LightAction: ", err)
		}
	}

	// Schedule the LightAction execution for ON and OFF
	onAction := &LightAction{State: pkg.LightStateOn}
	_, err = s.Scheduler.
		Every(lightingInterval).
		StartAt(startDate).
		Tag(g.ID.String()).
		Tag(pkg.LightStateOn.String()).
		Do(executeLightAction, onAction)
	return err
}

func (s *scheduler) removeLightScheduleByState(g *pkg.Garden, state string) error {
	if err := s.Scheduler.RemoveByTags(g.ID.String(), state); err != nil && !errors.Is(err, gocron.ErrJobNotFoundWithTag) {
		return err
	}
	return nil
}
