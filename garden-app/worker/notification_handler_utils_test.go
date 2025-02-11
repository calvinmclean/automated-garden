package worker

import (
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/stretchr/testify/assert"
)

func TestSendNotificationForGarden(t *testing.T) {
	t.Run("GardenWithoutNotificationClientID", func(t *testing.T) {
		w := &Worker{}
		err := w.sendNotificationForGarden(&pkg.Garden{}, "title", "message")
		assert.Error(t, err)
		assert.Equal(t, "garden does not have notification client", err.Error())
	})
}
