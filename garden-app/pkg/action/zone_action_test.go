package action

import (
	"net/http/httptest"
	"testing"
)

func TestZoneAction(t *testing.T) {
	tests := []struct {
		name   string
		action *ZoneAction
		err    string
	}{
		{
			"EmptyRequestError",
			nil,
			"missing required action fields",
		},
		{
			"EmptyZoneActionError",
			&ZoneAction{},
			"missing required action fields",
		},
	}

	t.Run("Successful", func(t *testing.T) {
		ar := &ZoneAction{
			Water: &WaterAction{},
		}
		r := httptest.NewRequest("", "/", nil)
		err := ar.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading ZoneActionRequest JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.action.Bind(r)
			if err == nil {
				t.Error("Expected error reading ZoneActionRequest JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}
