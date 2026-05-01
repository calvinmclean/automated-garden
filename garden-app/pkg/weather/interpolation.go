package weather

// Interpolator defines the interface for interpolation algorithms
type Interpolator interface {
	Interpolate(t float64) float64 // t is normalized 0-1
}

// InterpolationMode represents the type of interpolation to use
type InterpolationMode string

const (
	Linear    InterpolationMode = "linear"
	EaseIn    InterpolationMode = "ease_in"
	EaseOut   InterpolationMode = "ease_out"
	EaseInOut InterpolationMode = "ease_in_out"
	Step      InterpolationMode = "step"
)

// IsValid checks if the interpolation mode is valid
func (m InterpolationMode) IsValid() bool {
	switch m {
	case Linear, EaseIn, EaseOut, EaseInOut, Step:
		return true
	}
	return false
}

// LinearInterpolator implements linear interpolation (t -> t)
type LinearInterpolator struct{}

// Interpolate performs linear interpolation: t
func (l LinearInterpolator) Interpolate(t float64) float64 {
	return t
}

// EaseInInterpolator implements ease-in interpolation (t -> t²)
type EaseInInterpolator struct{}

// Interpolate performs ease-in interpolation: t²
func (e EaseInInterpolator) Interpolate(t float64) float64 {
	return t * t
}

// EaseOutInterpolator implements ease-out interpolation (t -> 1 - (1-t)²)
type EaseOutInterpolator struct{}

// Interpolate performs ease-out interpolation: 1 - (1-t)²
func (e EaseOutInterpolator) Interpolate(t float64) float64 {
	return 1 - (1-t)*(1-t)
}

// EaseInOutInterpolator implements ease-in-out interpolation
type EaseInOutInterpolator struct{}

// Interpolate performs ease-in-out interpolation:
// t < 0.5 ? 2t² : 1 - 2(1-t)²
func (e EaseInOutInterpolator) Interpolate(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return 1 - 2*(1-t)*(1-t)
}

// StepInterpolator implements step interpolation
type StepInterpolator struct{}

// Interpolate performs step interpolation: t < 1 ? 0 : 1
func (s StepInterpolator) Interpolate(t float64) float64 {
	if t < 1.0 {
		return 0.0
	}
	return 1.0
}

// GetInterpolator returns the appropriate interpolator for the given mode
func GetInterpolator(mode InterpolationMode) Interpolator {
	switch mode {
	case EaseIn:
		return EaseInInterpolator{}
	case EaseOut:
		return EaseOutInterpolator{}
	case EaseInOut:
		return EaseInOutInterpolator{}
	case Step:
		return StepInterpolator{}
	case Linear:
	default:
		return LinearInterpolator{}
	}
	return LinearInterpolator{}
}
