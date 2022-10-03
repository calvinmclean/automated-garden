package controller

import (
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/stretchr/testify/assert"
)

type assertionData struct {
	waterActions   []action.WaterMessage
	stopActions    int
	stopAllActions int
	lightActions   []action.LightAction
}

// AssertWaterActions is used to check that all expected WaterMessages were received, then reset recorded info
func (c *Controller) AssertWaterActions(t *testing.T, expected ...action.WaterMessage) {
	assert.Equal(t, expected, c.assertionData.waterActions)
	c.assertionData.waterActions = []action.WaterMessage{}
}

// AssertStopActions is used to check that the expected number of StopActions were received, then reset recorded info
func (c *Controller) AssertStopActions(t *testing.T, expected int) {
	assert.Equal(t, expected, c.assertionData.stopActions)
	c.assertionData.stopActions = 0
}

// AssertStopAllActions is used to check that the expected number of StopAllActions were received, then reset recorded info
func (c *Controller) AssertStopAllActions(t *testing.T, expected int) {
	assert.Equal(t, expected, c.assertionData.stopAllActions)
	c.assertionData.stopAllActions = 0
}

// AssertLightActions is used to check that all expected LightActions were received, then reset recorded info
func (c *Controller) AssertLightActions(t *testing.T, expected ...action.LightAction) {
	assert.Equal(t, expected, c.assertionData.lightActions)
	c.assertionData.lightActions = []action.LightAction{}
}
