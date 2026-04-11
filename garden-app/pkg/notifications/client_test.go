package notifications

import (
	"testing"

	"github.com/containrrr/shoutrrr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigPatch(t *testing.T) {
	tests := []struct {
		name      string
		newConfig *Client
	}{
		{
			"PatchURL",
			&Client{URL: "pushover://shoutrrr:test@user/"},
		},
		{
			"PatchName",
			&Client{Name: "NewName"},
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

func TestNewClientInvalidURL(t *testing.T) {
	_, err := shoutrrr.CreateSender("invalid://url")
	assert.Error(t, err)
}

func TestEndDated(t *testing.T) {
	assert.False(t, (&Client{}).EndDated())
}

func TestTestCreateEmptyURL(t *testing.T) {
	err := (&Client{}).TestCreate()
	assert.Error(t, err)
	assert.Equal(t, "missing required url field", err.Error())
}

func TestTestCreateFakeURL(t *testing.T) {
	err := (&Client{URL: "fake://"}).TestCreate()
	assert.NoError(t, err)
}

func TestTestCreateFakeURLError(t *testing.T) {
	err := (&Client{URL: "fake://?create_error=fail"}).TestCreate()
	assert.Error(t, err)
}

func TestSendMessageFakeURL(t *testing.T) {
	client := &Client{URL: "fake://"}
	err := client.SendMessage("title", "message")
	assert.NoError(t, err)
}

func TestSendMessageFakeURLError(t *testing.T) {
	client := &Client{URL: "fake://?send_message_error=fail"}
	err := client.SendMessage("title", "message")
	assert.Error(t, err)
}
