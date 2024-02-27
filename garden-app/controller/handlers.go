package controller

import (
	"encoding/json"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	paho "github.com/eclipse/paho.mqtt.golang"
)

func (c *Controller) waterHandler(topic string) paho.MessageHandler {
	return func(_ paho.Client, msg paho.Message) {
		waterLogger := c.subLogger.With("topic", topic)
		var waterMsg action.WaterMessage
		err := json.Unmarshal(msg.Payload(), &waterMsg)
		if err != nil {
			waterLogger.Error("unable to unmarshal WaterMessage JSON", "error", err)
			return
		}

		c.assertionData.Lock()
		c.assertionData.waterActions = append(c.assertionData.waterActions, waterMsg)
		c.assertionData.Unlock()

		waterLogger.With(
			"zone_id", waterMsg.ZoneID,
			"position", waterMsg.Position,
			"duration", waterMsg.Duration,
		).Info("received WaterAction")
		c.publishWaterEvent(waterMsg, topic)
	}
}

func (c *Controller) stopHandler(_ string) paho.MessageHandler {
	return func(_ paho.Client, msg paho.Message) {
		c.assertionData.Lock()
		c.assertionData.stopActions++
		c.assertionData.Unlock()

		c.subLogger.Info("received StopAction", "topic", msg.Topic())
	}
}

func (c *Controller) stopAllHandler(_ string) paho.MessageHandler {
	return paho.MessageHandler(func(_ paho.Client, msg paho.Message) {
		c.assertionData.Lock()
		c.assertionData.stopAllActions++
		c.assertionData.Unlock()

		c.subLogger.Info("received StopAllAction", "topic", msg.Topic())
	})
}

func (c *Controller) lightHandler(topic string) paho.MessageHandler {
	return paho.MessageHandler(func(_ paho.Client, msg paho.Message) {
		lightLogger := c.subLogger.With("topic", topic)
		var action action.LightAction
		err := json.Unmarshal(msg.Payload(), &action)
		if err != nil {
			lightLogger.Error("unable to unmarshal LightAction JSON", "error", err)
			return
		}

		c.assertionData.Lock()
		c.assertionData.lightActions = append(c.assertionData.lightActions, action)
		c.assertionData.Unlock()

		lightLogger.Info("received LightAction", "state", action.State)
	})
}
