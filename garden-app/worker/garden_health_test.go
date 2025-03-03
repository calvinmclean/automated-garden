package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/automated-garden/garden-app/pkg/influxdb"
	"github.com/stretchr/testify/mock"
)

func TestGetGardenHealth(t *testing.T) {
	tests := []struct {
		name            string
		lastContactTime time.Time
		err             error
		expectedStatus  pkg.HealthStatus
	}{
		{
			"GardenIsUp",
			clock.Now(),
			nil,
			pkg.HealthStatusUp,
		},
		{
			"GardenIsDown",
			clock.Now().Add(-5 * time.Minute),
			nil,
			pkg.HealthStatusDown,
		},
		{
			"InfluxDBError",
			time.Time{},
			errors.New("influxdb error"),
			pkg.HealthStatusUnknown,
		},
		{
			"ZeroTime",
			time.Time{},
			nil,
			pkg.HealthStatusDown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			influxdbClient := new(influxdb.MockClient)
			influxdbClient.On("GetLastContact", mock.Anything, "garden").Return(tt.lastContactTime, tt.err)

			g := &pkg.Garden{TopicPrefix: "garden"}

			w := Worker{influxdbClient: influxdbClient}
			gardenHealth := w.GetGardenHealth(context.Background(), g)
			if gardenHealth.Status != tt.expectedStatus {
				t.Errorf("Unexpected GardenHealth.Status: expected = %s, actual = %s", tt.expectedStatus, gardenHealth.Status)
			}
		})
	}
}
