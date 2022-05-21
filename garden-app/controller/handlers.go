package controller

import (
	"encoding/json"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

func (c *Controller) waterHandler(topic string) paho.MessageHandler {
	return func(pc paho.Client, msg paho.Message) {
		waterLogger := subLogger.WithField("topic", topic)
		var waterMsg action.WaterMessage
		err := json.Unmarshal(msg.Payload(), &waterMsg)
		if err != nil {
			waterLogger.WithError(err).Error("unable to unmarshal WaterMessage JSON")
			return
		}
		waterLogger.WithFields(logrus.Fields{
			"zone_id":  waterMsg.ZoneID,
			"position": waterMsg.Position,
			"duration": waterMsg.Duration,
		}).Info("received WaterAction")
		c.publishWaterEvent(waterMsg, topic)
	}
}

func (c *Controller) stopHandler(topic string) paho.MessageHandler {
	return func(pc paho.Client, msg paho.Message) {
		subLogger.WithFields(logrus.Fields{
			"topic": msg.Topic(),
		}).Info("received StopAction")
	}
}

func (c *Controller) stopAllHandler(topic string) paho.MessageHandler {
	return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
		subLogger.WithFields(logrus.Fields{
			"topic": msg.Topic(),
		}).Info("received StopAllAction")
	})
}

func (c *Controller) lightHandler(topic string) paho.MessageHandler {
	return paho.MessageHandler(func(pc paho.Client, msg paho.Message) {
		lightLogger := subLogger.WithField("topic", topic)
		var action action.LightAction
		err := json.Unmarshal(msg.Payload(), &action)
		if err != nil {
			lightLogger.WithError(err).Error("unable to unmarshal LightAction JSON")
			return
		}
		lightLogger.WithFields(logrus.Fields{
			"state": action.State,
		}).Info("received LightAction")
	})
}