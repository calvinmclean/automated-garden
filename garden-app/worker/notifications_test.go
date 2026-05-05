package worker

import (
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/stretchr/testify/assert"
)

func TestGenerateWateringNotificationContent(t *testing.T) {
	tests := []struct {
		name      string
		ws        *pkg.WaterSchedule
		duration  time.Duration
		zoneCount int
		wantTitle string
		wantMsg   string
	}{
		{
			name: "Single zone with name",
			ws: &pkg.WaterSchedule{
				Name:     "My Schedule",
				Duration: &pkg.Duration{Duration: 30 * time.Minute},
			},
			duration:  30 * time.Minute,
			zoneCount: 1,
			wantTitle: "Watering 1 Zone: My Schedule",
			wantMsg:   "Duration: 30m",
		},
		{
			name: "Multiple zones with name",
			ws: &pkg.WaterSchedule{
				Name:     "My Schedule",
				Duration: &pkg.Duration{Duration: 30 * time.Minute},
			},
			duration:  30 * time.Minute,
			zoneCount: 3,
			wantTitle: "Watering 3 Zones: My Schedule",
			wantMsg:   "Duration: 30m",
		},
		{
			name: "Single zone without name",
			ws: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: 30 * time.Minute},
			},
			duration:  30 * time.Minute,
			zoneCount: 1,
			wantTitle: "Watering 1 Zone",
			wantMsg:   "Duration: 30m",
		},
		{
			name: "No zones with name - reminder mode",
			ws: &pkg.WaterSchedule{
				Name:     "Indoor Plants",
				Duration: &pkg.Duration{Duration: 30 * time.Minute},
			},
			duration:  30 * time.Minute,
			zoneCount: 0,
			wantTitle: "Watering Reminder: Indoor Plants",
			wantMsg:   "Duration: 30m",
		},
		{
			name: "No zones without name - reminder mode",
			ws: &pkg.WaterSchedule{
				Duration: &pkg.Duration{Duration: 30 * time.Minute},
			},
			duration:  30 * time.Minute,
			zoneCount: 0,
			wantTitle: "Watering Reminder",
			wantMsg:   "Duration: 30m",
		},
		{
			name: "Zero duration with zones - weather skip",
			ws: &pkg.WaterSchedule{
				Name:     "My Schedule",
				Duration: &pkg.Duration{Duration: 30 * time.Minute},
			},
			duration:  0,
			zoneCount: 2,
			wantTitle: "Watering 2 Zones: My Schedule",
			wantMsg:   "Weather conditions suggest skipping watering today",
		},
		{
			name: "Zero duration no zones - weather skip reminder",
			ws: &pkg.WaterSchedule{
				Name:     "Indoor Plants",
				Duration: &pkg.Duration{Duration: 30 * time.Minute},
			},
			duration:  0,
			zoneCount: 0,
			wantTitle: "Watering Reminder: Indoor Plants",
			wantMsg:   "Weather conditions suggest skipping watering today",
		},
		{
			name: "Scaled duration with zones",
			ws: &pkg.WaterSchedule{
				Name:     "My Schedule",
				Duration: &pkg.Duration{Duration: 30 * time.Minute},
			},
			duration:  15 * time.Minute,
			zoneCount: 1,
			wantTitle: "Watering 1 Zone: My Schedule",
			wantMsg:   "Duration: 15m (base: 30m, scaled 0.50x)",
		},
		{
			name: "Scaled duration no zones",
			ws: &pkg.WaterSchedule{
				Name:     "Indoor Plants",
				Duration: &pkg.Duration{Duration: 1 * time.Hour},
			},
			duration:  30 * time.Minute,
			zoneCount: 0,
			wantTitle: "Watering Reminder: Indoor Plants",
			wantMsg:   "Duration: 30m (base: 1h, scaled 0.50x)",
		},
		{
			name: "Scaled up duration",
			ws: &pkg.WaterSchedule{
				Name:     "My Schedule",
				Duration: &pkg.Duration{Duration: 30 * time.Minute},
			},
			duration:  45 * time.Minute,
			zoneCount: 1,
			wantTitle: "Watering 1 Zone: My Schedule",
			wantMsg:   "Duration: 45m (base: 30m, scaled 1.50x)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTitle, gotMsg := generateWateringNotificationContent(tt.ws, tt.duration, tt.zoneCount)
			assert.Equal(t, tt.wantTitle, gotTitle)
			assert.Equal(t, tt.wantMsg, gotMsg)
		})
	}
}
