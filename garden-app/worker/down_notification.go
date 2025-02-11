package worker

import (
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func (w *Worker) healthMessageHandler(_ mqtt.Client, msg mqtt.Message) {
	w.handleHealthMessage(msg.Topic(), string(msg.Payload()))
}

func (w *Worker) handleHealthMessage(topic, payload string) {
	logger := w.logger.With("topic", topic)

	if !checkHealthMessage(topic, payload) {
		logger.Warn("unexpected message from controller", "message", payload)
		return
	}
	logger.Info("received message", "message", payload)

	garden, err := w.getGardenForTopic(topic)
	if err != nil {
		logger.Error("error getting Garden for health topic", "error", err)
		return
	}

	downtime := garden.GetNotificationSettings().Downtime
	if downtime == nil || downtime.Duration <= 0 {
		return
	}

	timer, ok := w.downTimers[topic]
	if !ok {
		timer := w.newDownTimer(downtime.Duration, topic)
		w.downTimers[topic] = timer
		logger.Info("created new timer")
	} else {
		timer.Reset(downtime.Duration)
		logger.Info("reset timer")
	}
}

func (w *Worker) newDownTimer(d time.Duration, topic string) clock.Timer {
	return clock.AfterFunc(d, func() {
		// make Worker wait for this before shutdown
		w.downtimeWG.Add(1)
		defer w.downtimeWG.Done()

		logger := w.logger.With("topic", topic)

		err := w.handleDowntimeNotification(topic)
		if err != nil {
			logger.Error("failed to handle downtime timer", "error", err)
			return
		}

		logger.Info("successfully sent down notification")
	})
}

// handleDowntimeNotification re-reads the Garden from storage in case it's configuration changed
// after this goroutine was started
func (w *Worker) handleDowntimeNotification(topic string) error {
	garden, err := w.getGardenForTopic(topic)
	if err != nil {
		return err
	}

	downtime := garden.GetNotificationSettings().Downtime
	if downtime == nil || downtime.Duration <= 0 {
		return nil
	}

	title := fmt.Sprintf("%s is down", garden.Name)
	msg := fmt.Sprintf("Garden has been down for > %s", downtime.String())

	return w.sendNotificationForGarden(garden, title, msg)
}

// message format: 'health garden="{{ TopicPrefix }}"'
func checkHealthMessage(topic, msg string) bool {
	topicPrefix, err := getTopicPrefix(topic)
	if err != nil {
		return false
	}
	return msg == fmt.Sprintf(`health garden="%s"`, topicPrefix)
}
