package pkg

import (
	"time"

	"github.com/rs/xid"
)

type Garden struct {
	Name      string            `json:"name" yaml:"name,omitempty"`
	ID        xid.ID            `json:"id" yaml:"id,omitempty"`
	Plants    map[xid.ID]*Plant `json:"plants" yaml:"plants,omitempty"`
	CreatedAt *time.Time        `json:"created_at" yaml:"created_at,omitempty"`
	EndDate   *time.Time        `json:"end_date,omitempty" yaml:"end_date,omitempty"`
}
