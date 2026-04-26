package weather

import (
	"fmt"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

// Test interpolation modes with table-driven tests
func TestInterpolators(t *testing.T) {
	tests := []struct {
		name         string
		interpolator Interpolator
		t            float64
		expected     float64
		tolerance    float64
	}{
		// Linear interpolation tests
		{"linear at 0", LinearInterpolator{}, 0.0, 0.0, 0.0001},
		{"linear at 0.25", LinearInterpolator{}, 0.25, 0.25, 0.0001},
		{"linear at 0.5", LinearInterpolator{}, 0.5, 0.5, 0.0001},
		{"linear at 0.75", LinearInterpolator{}, 0.75, 0.75, 0.0001},
		{"linear at 1", LinearInterpolator{}, 1.0, 1.0, 0.0001},

		// Ease-in interpolation tests (t²)
		{"ease_in at 0", EaseInInterpolator{}, 0.0, 0.0, 0.0001},
		{"ease_in at 0.25", EaseInInterpolator{}, 0.25, 0.0625, 0.0001},
		{"ease_in at 0.5", EaseInInterpolator{}, 0.5, 0.25, 0.0001},
		{"ease_in at 0.75", EaseInInterpolator{}, 0.75, 0.5625, 0.0001},
		{"ease_in at 1", EaseInInterpolator{}, 1.0, 1.0, 0.0001},

		// Ease-out interpolation tests (1 - (1-t)²)
		{"ease_out at 0", EaseOutInterpolator{}, 0.0, 0.0, 0.0001},
		{"ease_out at 0.25", EaseOutInterpolator{}, 0.25, 0.4375, 0.0001},
		{"ease_out at 0.5", EaseOutInterpolator{}, 0.5, 0.75, 0.0001},
		{"ease_out at 0.75", EaseOutInterpolator{}, 0.75, 0.9375, 0.0001},
		{"ease_out at 1", EaseOutInterpolator{}, 1.0, 1.0, 0.0001},

		// Ease-in-out interpolation tests
		{"ease_in_out at 0", EaseInOutInterpolator{}, 0.0, 0.0, 0.0001},
		{"ease_in_out at 0.25", EaseInOutInterpolator{}, 0.25, 0.125, 0.0001},
		{"ease_in_out at 0.5", EaseInOutInterpolator{}, 0.5, 0.5, 0.0001},
		{"ease_in_out at 0.75", EaseInOutInterpolator{}, 0.75, 0.875, 0.0001},
		{"ease_in_out at 1", EaseInOutInterpolator{}, 1.0, 1.0, 0.0001},

		// Step interpolation tests
		{"step at 0", StepInterpolator{}, 0.0, 0.0, 0.0001},
		{"step at 0.5", StepInterpolator{}, 0.5, 0.0, 0.0001},
		{"step at 0.99", StepInterpolator{}, 0.99, 0.0, 0.0001},
		{"step at 1", StepInterpolator{}, 1.0, 1.0, 0.0001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.interpolator.Interpolate(tt.t)
			assert.InDelta(t, tt.expected, result, tt.tolerance,
				"Interpolate(%v) expected %v but got %v", tt.t, tt.expected, result)
		})
	}
}

// Test ease_in_out symmetry property
func TestEaseInOutSymmetry(t *testing.T) {
	interpolator := EaseInOutInterpolator{}

	// Test symmetry: f(t) + f(1-t) should equal 1
	for val := 0.0; val <= 1.0; val += 0.1 {
		f1 := interpolator.Interpolate(val)
		f2 := interpolator.Interpolate(1 - val)
		assert.InDelta(t, 1.0, f1+f2, 0.0001,
			"Symmetry check failed at t=%v: f(t)=%v, f(1-t)=%v", val, f1, f2)
	}
}

// Test GetInterpolator factory function
func TestGetInterpolator(t *testing.T) {
	tests := []struct {
		mode     InterpolationMode
		expected Interpolator
	}{
		{Linear, LinearInterpolator{}},
		{EaseIn, EaseInInterpolator{}},
		{EaseOut, EaseOutInterpolator{}},
		{EaseInOut, EaseInOutInterpolator{}},
		{Step, StepInterpolator{}},
		{"invalid", LinearInterpolator{}}, // defaults to linear
		{"", LinearInterpolator{}},        // empty defaults to linear
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			result := GetInterpolator(tt.mode)
			assert.IsType(t, tt.expected, result)
		})
	}
}

// Test InterpolationMode.IsValid
func TestInterpolationModeIsValid(t *testing.T) {
	tests := []struct {
		mode     InterpolationMode
		expected bool
	}{
		{Linear, true},
		{EaseIn, true},
		{EaseOut, true},
		{EaseInOut, true},
		{Step, true},
		{"invalid", false},
		{"", false},
		{"LINEAR", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.mode.IsValid())
		})
	}
}

// Test WeatherScaler Scale method with clamping
func TestWeatherScalerClamping(t *testing.T) {
	scaler := &WeatherScaler{
		Interpolation: Linear,
		InputMin:      float64Ptr(10.0),
		InputMax:      float64Ptr(50.0),
		FactorMin:     float64Ptr(0.5),
		FactorMax:     float64Ptr(1.5),
	}

	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"below min", 5.0, 0.5},
		{"at min", 10.0, 0.5},
		{"above max", 60.0, 1.5},
		{"at max", 50.0, 1.5},
		{"midpoint", 30.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scaler.Scale(tt.input)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

// Scenario 1: Rain Scaler - Input: 0-30mm, Output: 1.0-0.0 (linear)
func TestRainScenario(t *testing.T) {
	rainScaler := &WeatherScaler{
		Interpolation: Linear,
		InputMin:      float64Ptr(0.0),
		InputMax:      float64Ptr(30.0),
		FactorMin:     float64Ptr(1.0),
		FactorMax:     float64Ptr(0.0),
	}

	tests := []struct {
		input    float64
		expected float64
	}{
		{0.0, 1.0},
		{15.0, 0.5},
		{30.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("rain_%vmm", tt.input), func(t *testing.T) {
			result := rainScaler.Scale(tt.input)
			assert.InDelta(t, tt.expected, result, 0.0001)
		})
	}
}

// Scenario 2: Temperature Scaler - Input: 20-46°C, Output: 1.0-1.5 (ease_in)
func TestTemperatureScenario(t *testing.T) {
	tempScaler := &WeatherScaler{
		Interpolation: EaseIn,
		InputMin:      float64Ptr(20.0),
		InputMax:      float64Ptr(46.0),
		FactorMin:     float64Ptr(1.0),
		FactorMax:     float64Ptr(1.5),
	}

	// Manual calculation verification
	t.Run("20C", func(t *testing.T) {
		assert.InDelta(t, 1.0, tempScaler.Scale(20.0), 0.0001)
	})
	t.Run("33C", func(t *testing.T) {
		// t = (33-20)/(46-20) = 0.5
		// ease_in(0.5) = 0.25
		// result = 1.0 + 0.5*0.25 = 1.125
		assert.InDelta(t, 1.125, tempScaler.Scale(33.0), 0.0001)
	})
	t.Run("46C", func(t *testing.T) {
		assert.InDelta(t, 1.5, tempScaler.Scale(46.0), 0.0001)
	})
}

// Test multi-scaler multiplication
func TestScaleMulti(t *testing.T) {
	rainScaler := &WeatherScaler{
		Interpolation: Linear,
		InputMin:      float64Ptr(0.0),
		InputMax:      float64Ptr(30.0),
		FactorMin:     float64Ptr(1.0),
		FactorMax:     float64Ptr(0.0),
	}

	tempScaler := &WeatherScaler{
		Interpolation: Linear,
		InputMin:      float64Ptr(20.0),
		InputMax:      float64Ptr(46.0),
		FactorMin:     float64Ptr(1.0),
		FactorMax:     float64Ptr(1.5),
	}

	// rain: 6mm -> t=0.2 -> 0.8, temp: 30C -> t=(30-20)/26=10/26 -> 1.0+0.5*(10/26)=1.1923...
	// expected: 0.8 * 1.192307... = 0.953846...
	scalers := []*WeatherScaler{rainScaler, tempScaler}
	inputs := []float64{6.0, 30.0}

	result := ScaleMulti(scalers, inputs)
	assert.InDelta(t, 0.953846, result, 0.0001)
}

// Test ScaleMulti with mismatched lengths
func TestScaleMultiMismatchedLengths(t *testing.T) {
	scaler := &WeatherScaler{
		Interpolation: Linear,
		InputMin:      float64Ptr(0.0),
		InputMax:      float64Ptr(10.0),
		FactorMin:     float64Ptr(0.5),
		FactorMax:     float64Ptr(1.0),
	}

	// Mismatched lengths should return 1.0
	result := ScaleMulti([]*WeatherScaler{scaler}, []float64{5.0, 10.0})
	assert.InDelta(t, 1.0, result, 0.0001)
}

// Test Validation
func TestWeatherScalerValidate(t *testing.T) {
	tests := []struct {
		name      string
		scaler    WeatherScaler
		wantError bool
		errMsg    string
	}{
		{
			name: "valid config",
			scaler: WeatherScaler{
				ClientID:      xid.New(),
				Interpolation: Linear,
				InputMin:      float64Ptr(0.0),
				InputMax:      float64Ptr(10.0),
				FactorMin:     float64Ptr(0.5),
				FactorMax:     float64Ptr(1.5),
			},
			wantError: false,
		},
		{
			name: "input_max <= input_min",
			scaler: WeatherScaler{
				Interpolation: Linear,
				InputMin:      float64Ptr(10.0),
				InputMax:      float64Ptr(10.0),
				FactorMin:     float64Ptr(0.5),
				FactorMax:     float64Ptr(1.5),
			},
			wantError: true,
			errMsg:    "input_max must be greater than input_min",
		},
		{
			name: "negative FactorMin",
			scaler: WeatherScaler{
				Interpolation: Linear,
				InputMin:      float64Ptr(0.0),
				InputMax:      float64Ptr(10.0),
				FactorMin:     float64Ptr(-0.5),
				FactorMax:     float64Ptr(1.5),
			},
			wantError: true,
			errMsg:    "factors must be non-negative",
		},
		{
			name: "negative FactorMax",
			scaler: WeatherScaler{
				Interpolation: Linear,
				InputMin:      float64Ptr(0.0),
				InputMax:      float64Ptr(10.0),
				FactorMin:     float64Ptr(0.5),
				FactorMax:     float64Ptr(-1.5),
			},
			wantError: true,
			errMsg:    "factors must be non-negative",
		},
		{
			name: "invalid interpolation mode",
			scaler: WeatherScaler{
				Interpolation: "invalid_mode",
				InputMin:      float64Ptr(0.0),
				InputMax:      float64Ptr(10.0),
				FactorMin:     float64Ptr(0.5),
				FactorMax:     float64Ptr(1.5),
			},
			wantError: true,
			errMsg:    "invalid interpolation mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.scaler.Validate()
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test edge cases
func TestWeatherScalerEdgeCases(t *testing.T) {
	t.Run("flat line - FactorMin equals FactorMax", func(t *testing.T) {
		scaler := &WeatherScaler{
			Interpolation: Linear,
			InputMin:      float64Ptr(0.0),
			InputMax:      float64Ptr(10.0),
			FactorMin:     float64Ptr(0.5),
			FactorMax:     float64Ptr(0.5),
		}

		// Should always return 0.5 regardless of input
		assert.InDelta(t, 0.5, scaler.Scale(0.0), 0.0001)
		assert.InDelta(t, 0.5, scaler.Scale(5.0), 0.0001)
		assert.InDelta(t, 0.5, scaler.Scale(10.0), 0.0001)
	})

	t.Run("very large input range", func(t *testing.T) {
		scaler := &WeatherScaler{
			Interpolation: Linear,
			InputMin:      float64Ptr(0.0),
			InputMax:      float64Ptr(1e9),
			FactorMin:     float64Ptr(0.0),
			FactorMax:     float64Ptr(1.0),
		}

		assert.InDelta(t, 0.5, scaler.Scale(5e8), 0.0001)
	})

	t.Run("very small input range", func(t *testing.T) {
		scaler := &WeatherScaler{
			Interpolation: Linear,
			InputMin:      float64Ptr(0.0),
			InputMax:      float64Ptr(0.001),
			FactorMin:     float64Ptr(0.0),
			FactorMax:     float64Ptr(1.0),
		}

		assert.InDelta(t, 0.5, scaler.Scale(0.0005), 0.0001)
	})
}

// Test floating point precision
func TestFloatingPointPrecision(t *testing.T) {
	scaler := &WeatherScaler{
		Interpolation: Linear,
		InputMin:      float64Ptr(0.0),
		InputMax:      float64Ptr(1.0),
		FactorMin:     float64Ptr(0.0),
		FactorMax:     float64Ptr(1.0),
	}

	// Test that results are precise
	for i := 0; i <= 100; i++ {
		input := float64(i) / 100.0
		result := scaler.Scale(input)
		expected := input
		assert.InDelta(t, expected, result, 0.0001,
			"Precision test failed at input %v", input)
	}
}

// Test with all interpolation modes
func TestAllInterpolationModes(t *testing.T) {
	modes := []InterpolationMode{Linear, EaseIn, EaseOut, EaseInOut, Step}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			scaler := &WeatherScaler{
				Interpolation: mode,
				InputMin:      float64Ptr(0.0),
				InputMax:      float64Ptr(10.0),
				FactorMin:     float64Ptr(0.0),
				FactorMax:     float64Ptr(1.0),
			}

			// All interpolators should return FactorMin at InputMin
			assert.InDelta(t, 0.0, scaler.Scale(0.0), 0.0001)

			// All interpolators should return FactorMax at InputMax
			assert.InDelta(t, 1.0, scaler.Scale(10.0), 0.0001)

			// All interpolators should return values in valid range
			for input := 0.0; input <= 10.0; input += 1.0 {
				result := scaler.Scale(input)
				assert.True(t, result >= 0.0 && result <= 1.0,
					"Result %v out of bounds for input %v", result, input)
			}
		})
	}
}

// Benchmark interpolation functions
func BenchmarkLinearInterpolation(b *testing.B) {
	interpolator := LinearInterpolator{}
	for i := 0; i < b.N; i++ {
		interpolator.Interpolate(0.5)
	}
}

func BenchmarkEaseInInterpolation(b *testing.B) {
	interpolator := EaseInInterpolator{}
	for i := 0; i < b.N; i++ {
		interpolator.Interpolate(0.5)
	}
}

func BenchmarkWeatherScalerScale(b *testing.B) {
	scaler := &WeatherScaler{
		Interpolation: EaseInOut,
		InputMin:      float64Ptr(0.0),
		InputMax:      float64Ptr(100.0),
		FactorMin:     float64Ptr(0.5),
		FactorMax:     float64Ptr(1.5),
	}
	for i := 0; i < b.N; i++ {
		scaler.Scale(50.0)
	}
}
