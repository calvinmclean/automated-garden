package pkg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeLocationFromOffset(t *testing.T) {
	tests := []struct {
		name        string
		offset      string
		expectedLoc string
	}{
		{
			"MST",
			"420",
			"MST",
		},
		{
			"UTC",
			"0",
			"UTC",
		},
		{
			"GMT",
			"0",
			"GMT",
		},
	}

	now := time.Now()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedLoc, _ := time.LoadLocation(tt.expectedLoc)

			loc, err := TimeLocationFromOffset(tt.offset)
			assert.NoError(t, err)

			assert.Equal(t, now.In(expectedLoc).UnixNano(), now.In(loc).UnixNano())
		})
	}

	t.Run("InvalidInput", func(t *testing.T) {
		loc, err := TimeLocationFromOffset("f")
		assert.Error(t, err)
		assert.Nil(t, loc)
	})
}
