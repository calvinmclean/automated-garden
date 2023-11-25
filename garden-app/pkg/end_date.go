package pkg

import "time"

type EndDateable interface {
	EndDated() bool
	SetEndDate(time.Time)
}
