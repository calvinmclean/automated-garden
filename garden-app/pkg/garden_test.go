package pkg

import (
	"errors"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/stretchr/testify/mock"
)

func TestHealth(t *testing.T) {
	tests := []struct {
		name            string
		lastContactTime time.Time
		err             error
		expectedStatus  string
	}{
		{
			"GardenIsUp",
			time.Now(),
			nil,
			"UP",
		},
		{
			"GardenIsDown",
			time.Now().Add(-5 * time.Minute),
			nil,
			"DOWN",
		},
		{
			"InfluxDBError",
			time.Now(),
			errors.New("influxdb error"),
			"N/A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "garden").Return(tt.lastContactTime, tt.err)

			g := Garden{Name: "garden"}

			gardenHealth := g.Health(influxdbClient)
			if gardenHealth.Status != tt.expectedStatus {
				t.Errorf("Unexpected GardenHealth.Status: expected = %s, actual = %s", tt.expectedStatus, gardenHealth.Status)
			}
		})
	}
}
