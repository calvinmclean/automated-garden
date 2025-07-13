package pkg

import (
	"errors"
	"net/http"

	"github.com/calvinmclean/babyapi"
)

// WaterRoutineStep specifies a Zone and Duration to water
type WaterRoutineStep struct {
	ZoneID   babyapi.ID `json:"zone_id" yaml:"zone_id"`
	Duration *Duration  `json:"duration" yaml:"duration"`
}

// WaterRoutine allows watering multiple Zones sequentially with one request
type WaterRoutine struct {
	ID    babyapi.ID         `json:"id" yaml:"id"`
	Name  string             `json:"name" yaml:"name"`
	Steps []WaterRoutineStep `json:"steps" yaml:"steps"`
}

func (wr WaterRoutine) GetID() string {
	return wr.ID.String()
}

func (wr *WaterRoutine) Bind(r *http.Request) error {
	if wr == nil {
		return errors.New("missing required WaterRoutine fields")
	}
	err := wr.ID.Bind(r)
	if err != nil {
		return err
	}

	return nil
}

func (wr *WaterRoutine) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
