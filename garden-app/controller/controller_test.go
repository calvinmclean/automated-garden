package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddNoise(t *testing.T) {
	base := 100.0
	percentRange := 5.0
	for i := 0; i < 100; i++ {
		r := addNoise(base, percentRange)
		assert.LessOrEqual(t, r, base+float64(percentRange))
		assert.GreaterOrEqual(t, r, base-float64(percentRange))
	}
}
