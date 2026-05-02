// Package units provides unit conversion utilities for temperature and length
package units

import "golang.org/x/exp/constraints"

// UnitSystem represents the unit system (metric or imperial)
type UnitSystem string

const (
	Metric   UnitSystem = "metric"
	Imperial UnitSystem = "imperial"
)

// IsMetric returns true if the unit system is metric
func (u UnitSystem) IsMetric() bool {
	return u == Metric
}

// IsImperial returns true if the unit system is imperial
func (u UnitSystem) IsImperial() bool {
	return u == Imperial
}

// Float is a constraint that matches both float32 and float64
type Float interface {
	constraints.Float
}

// CelsiusToFahrenheit converts Celsius to Fahrenheit
func CelsiusToFahrenheit[T Float](c T) T {
	return c*1.8 + 32
}

// FahrenheitToCelsius converts Fahrenheit to Celsius
func FahrenheitToCelsius[T Float](f T) T {
	return (f - 32) / 1.8
}

// MmToInches converts millimeters to inches using exact conversion (25.4 mm = 1 inch)
func MmToInches[T Float](mm T) T {
	return mm / 25.4
}

// InchesToMm converts inches to millimeters
func InchesToMm[T Float](inches T) T {
	return inches * 25.4
}
