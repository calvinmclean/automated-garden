package controller

import (
	"sync"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/action"
	"github.com/stretchr/testify/assert"
)

type assertionData struct {
	sync.Mutex

	waterActions   []action.WaterMessage
	stopActions    int
	stopAllActions int
	lightActions   []action.LightAction
}

// AssertWaterActions is used to check that all expected WaterMessages were received, then reset recorded info
func (c *Controller) AssertWaterActions(t *testing.T, expected ...action.WaterMessage) {
	t.Helper()

	c.assertionData.Lock()
	assert.Equal(t, expected, c.assertionData.waterActions)
	c.assertionData.waterActions = []action.WaterMessage{}
	c.assertionData.Unlock()
}

// AssertStopActions is used to check that the expected number of StopActions were received, then reset recorded info
func (c *Controller) AssertStopActions(t *testing.T, expected int) {
	t.Helper()

	c.assertionData.Lock()
	assert.Equal(t, expected, c.assertionData.stopActions)
	c.assertionData.stopActions = 0
	c.assertionData.Unlock()
}

// AssertStopAllActions is used to check that the expected number of StopAllActions were received, then reset recorded info
func (c *Controller) AssertStopAllActions(t *testing.T, expected int) {
	t.Helper()

	c.assertionData.Lock()
	assert.Equal(t, expected, c.assertionData.stopAllActions)
	c.assertionData.stopAllActions = 0
	c.assertionData.Unlock()
}

// AssertLightActions is used to check that all expected LightActions were received, then reset recorded info
func (c *Controller) AssertLightActions(t *testing.T, expected ...action.LightAction) {
	t.Helper()

	c.assertionData.Lock()
	assert.Equal(t, expected, c.assertionData.lightActions)
	c.assertionData.lightActions = []action.LightAction{}
	c.assertionData.Unlock()
}
