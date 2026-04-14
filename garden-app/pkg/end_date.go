// Package pkg provides domain models and utilities for the garden application
package pkg

import "time"

type EndDateable interface {
	EndDated() bool
	SetEndDate(time.Time)
}
