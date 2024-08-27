package worker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStartupMessage(t *testing.T) {
	input := "logs message=\"garden-controller setup complete\""
	msg := parseStartupMessage([]byte(input))
	require.Equal(t, "garden-controller setup complete", msg)
}
