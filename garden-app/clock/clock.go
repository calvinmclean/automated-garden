package clock

import (
	"time"

	"github.com/benbjohnson/clock"
	"github.com/go-co-op/gocron"
)

type Timer interface {
	Reset(d time.Duration) bool
}

// Clock allows mocking time
type Clock struct {
	clock.Clock
}

// DefaultClock is the underlying Clock used and can be overridden to mock
var DefaultClock = Clock{clock.New()}

var _ gocron.TimeWrapper = Clock{}

func (c Clock) Now(loc *time.Location) time.Time {
	return c.Clock.Now().In(loc)
}

func (c Clock) Unix(sec int64, nsec int64) time.Time {
	return time.Unix(sec, nsec).In(DefaultClock.Clock.Now().Location())
}

func Now() time.Time {
	return DefaultClock.Clock.Now()
}

func Since(t time.Time) time.Duration {
	return DefaultClock.Clock.Since(t)
}

// MockTime sets up the DefaultClock with a consistent time so it can be used across tests
func MockTime() *clock.Mock {
	mock := clock.NewMock()
	mock.Set(time.Date(2023, time.August, 23, 10, 0, 0, 0, time.UTC))
	DefaultClock = Clock{Clock: mock}
	return mock
}

// Reset returns the DefaultClock to real time
func Reset() {
	DefaultClock = Clock{clock.New()}
}

func AfterFunc(d time.Duration, f func()) Timer {
	return DefaultClock.AfterFunc(d, f)
}
