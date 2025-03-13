package pkg

import (
	"testing"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/stretchr/testify/assert"
)

func TestHistoryProgress(t *testing.T) {
	c := clock.MockTime()
	defer clock.Reset()
	now := c.Now()

	tests := []struct {
		name     string
		history  []WaterHistory
		expected WaterHistoryProgress
	}{
		{
			"EmptyHistory", nil, WaterHistoryProgress{},
		},
		{
			"OneEvent_Halfway",
			[]WaterHistory{{
				Duration:    Duration{Duration: 30 * time.Minute},
				EventID:     "EventID",
				Status:      WaterStatusStarted,
				SentAt:      now.Add(-15 * time.Minute),
				StartedAt:   now.Add(-15 * time.Minute),
				CompletedAt: time.Time{},
			}},
			WaterHistoryProgress{
				Duration: Duration{Duration: 30 * time.Minute},
				Elapsed:  Duration{Duration: 15 * time.Minute},
				Progress: 0.50,
				Queue:    0,
			},
		},
		{
			"OneEvent_Complete_ShowsNone",
			[]WaterHistory{{
				Duration:    Duration{Duration: 30 * time.Minute},
				EventID:     "EventID",
				Status:      WaterStatusCompleted,
				SentAt:      now.Add(-31 * time.Minute),
				StartedAt:   now.Add(-31 * time.Minute),
				CompletedAt: now.Add(-1 * time.Minute),
			}},
			WaterHistoryProgress{},
		},
		{
			"OneEvent_ElapsedLongerThanDuration_Error",
			[]WaterHistory{{
				Duration:    Duration{Duration: 30 * time.Minute},
				EventID:     "EventID",
				Status:      WaterStatusStarted,
				SentAt:      now.Add(-45 * time.Minute),
				StartedAt:   now.Add(-45 * time.Minute),
				CompletedAt: time.Time{},
			}},
			WaterHistoryProgress{
				Duration: Duration{Duration: 30 * time.Minute},
				Elapsed:  Duration{Duration: 45 * time.Minute},
				Progress: 0,
				Queue:    0,
				Error:    ErrElapsedExceedsDuration,
			},
		},
		{
			"OneEvent_Completed_DoNotShowOverHourOld",
			[]WaterHistory{{
				Duration:    Duration{Duration: 30 * time.Minute},
				EventID:     "EventID",
				Status:      WaterStatusCompleted,
				SentAt:      now.Add(-91 * time.Minute),
				StartedAt:   now.Add(-91 * time.Minute),
				CompletedAt: now.Add(-61 * time.Minute),
			}},
			WaterHistoryProgress{},
		},
		{
			"OneEvent_Sent",
			[]WaterHistory{{
				Duration:    Duration{Duration: 30 * time.Minute},
				EventID:     "EventID",
				Status:      WaterStatusSent,
				SentAt:      now.Add(-2 * time.Minute),
				StartedAt:   time.Time{},
				CompletedAt: time.Time{},
			}},
			WaterHistoryProgress{
				Duration: Duration{},
				Elapsed:  Duration{},
				Progress: 0.00,
				Queue:    1,
			},
		},
		{
			"TwoEvents_Sent_Started",
			[]WaterHistory{
				{
					Duration:    Duration{Duration: 15 * time.Minute},
					EventID:     "EventID_Sent",
					Status:      WaterStatusSent,
					SentAt:      now.Add(-2 * time.Minute),
					StartedAt:   time.Time{},
					CompletedAt: time.Time{},
				},
				{
					Duration:    Duration{Duration: 30 * time.Minute},
					EventID:     "EventID_Started",
					Status:      WaterStatusStarted,
					SentAt:      now.Add(-15 * time.Minute),
					StartedAt:   now.Add(-15 * time.Minute),
					CompletedAt: time.Time{},
				},
			},
			WaterHistoryProgress{
				Duration: Duration{Duration: 30 * time.Minute},
				Elapsed:  Duration{Duration: 15 * time.Minute},
				Progress: 0.50,
				Queue:    1,
			},
		},
		{
			"TwoEvents_Sent_Completed_Err",
			[]WaterHistory{
				{
					Duration:    Duration{Duration: 15 * time.Minute},
					EventID:     "EventID_Sent",
					Status:      WaterStatusSent,
					SentAt:      now.Add(-2 * time.Minute),
					StartedAt:   time.Time{},
					CompletedAt: time.Time{},
				},
				{
					Duration:    Duration{Duration: 30 * time.Minute},
					EventID:     "EventID_Completed",
					Status:      WaterStatusCompleted,
					SentAt:      now.Add(-31 * time.Minute),
					StartedAt:   now.Add(-31 * time.Minute),
					CompletedAt: now.Add(-1 * time.Minute),
				},
			},
			WaterHistoryProgress{
				Duration: Duration{},
				Elapsed:  Duration{},
				Progress: 0,
				Queue:    1,
				Error:    ErrSentButNotStarted,
			},
		},
		{
			"ThreeEvents_Sent_Completed_Err",
			[]WaterHistory{
				{
					Duration:    Duration{Duration: 15 * time.Minute},
					EventID:     "EventID_Sent2",
					Status:      WaterStatusSent,
					SentAt:      now.Add(-2 * time.Minute),
					StartedAt:   time.Time{},
					CompletedAt: time.Time{},
				},
				{
					Duration:    Duration{Duration: 15 * time.Minute},
					EventID:     "EventID_Sent1",
					Status:      WaterStatusSent,
					SentAt:      now.Add(-3 * time.Minute),
					StartedAt:   time.Time{},
					CompletedAt: time.Time{},
				},
				{
					Duration:    Duration{Duration: 30 * time.Minute},
					EventID:     "EventID_Completed",
					Status:      WaterStatusCompleted,
					SentAt:      now.Add(-31 * time.Minute),
					StartedAt:   now.Add(-31 * time.Minute),
					CompletedAt: now.Add(-1 * time.Minute),
				},
			},
			WaterHistoryProgress{
				Duration: Duration{},
				Elapsed:  Duration{},
				Progress: 0,
				Queue:    2,
				Error:    ErrSentButNotStarted,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progress := CalculateWaterProgress(tt.history)
			assert.Equal(t, tt.expected, progress)
		})
	}
}
