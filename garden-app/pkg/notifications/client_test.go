package notifications

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigPatch(t *testing.T) {
	tests := []struct {
		name      string
		newConfig *Client
	}{
		{
			"PatchType",
			&Client{Type: "other_type"},
		},
		{
			"PatchName",
			&Client{Name: "NewName"},
		},
		{
			"PatchOptions",
			&Client{Options: map[string]any{
				"key": "value",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{}
			err := c.Patch(tt.newConfig)
			require.Nil(t, err)
			assert.Equal(t, tt.newConfig, c)
		})
	}
}

func TestNewClientInvalidType(t *testing.T) {
	_, err := newClient(&Client{Type: "DNE"})
	assert.Error(t, err)
	assert.Equal(t, "invalid type 'DNE'", err.Error())
}

func TestEndDated(t *testing.T) {
	assert.False(t, (&Client{}).EndDated())
}
