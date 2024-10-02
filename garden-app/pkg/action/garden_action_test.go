package action

import (
	"net/http/httptest"
	"testing"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

func TestGardenActionBind(t *testing.T) {
	tests := []struct {
		name   string
		action *GardenAction
		err    string
	}{
		{
			"EmptyRequestError",
			nil,
			"missing required action fields",
		},
		{
			"EmptyActionError",
			&GardenAction{},
			"missing required action fields",
		},
		{
			"EmptyGardenActionError",
			&GardenAction{},
			"missing required action fields",
		},
		{
			"ErrorMissingUpdateConfig",
			&GardenAction{Update: &UpdateAction{}},
			"update action must have config=true",
		},
	}

	t.Run("SuccessfulLightAction", func(t *testing.T) {
		ar := &GardenAction{
			Light: &LightAction{
				State: pkg.LightStateOn,
			},
		}
		r := httptest.NewRequest("", "/", nil)
		err := ar.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading GardenAction JSON: %v", err)
		}
	})
	t.Run("SuccessfulStopAction", func(t *testing.T) {
		ar := &GardenAction{
			Stop: &StopAction{},
		}
		r := httptest.NewRequest("", "/", nil)
		err := ar.Bind(r)
		if err != nil {
			t.Errorf("Unexpected error reading GardenAction JSON: %v", err)
		}
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("", "/", nil)
			err := tt.action.Bind(r)
			if err == nil {
				t.Error("Expected error reading GardenAction JSON, but none occurred")
				return
			}
			if err.Error() != tt.err {
				t.Errorf("Unexpected error string: %v", err)
			}
		})
	}
}
