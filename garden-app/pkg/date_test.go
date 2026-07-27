package pkg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDateUnmarshalText(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Date
		expectError bool
	}{
		{
			name:     "DateOnly format",
			input:    "2026-05-06",
			expected: Date{Year: 2026, Month: 5, Day: 6},
		},
		{
			name:     "empty string",
			input:    "",
			expected: Date{},
		},
		{
			name:        "invalid format",
			input:       "not-a-date",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Date
			err := d.UnmarshalText([]byte(tt.input))
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, d)
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Date
		expectError bool
	}{
		{
			name:     "DateOnly format",
			input:    "2026-05-06",
			expected: Date{Year: 2026, Month: 5, Day: 6},
		},
		{
			name:     "RFC3339 fallback",
			input:    "2026-05-06T00:00:00Z",
			expected: Date{Year: 2026, Month: 5, Day: 6},
		},
		{
			name:     "empty string",
			input:    "",
			expected: Date{},
		},
		{
			name:        "invalid format",
			input:       "not-a-date",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := ParseDate(tt.input)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, d)
		})
	}
}
