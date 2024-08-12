package server

import (
	"github.com/calvinmclean/automated-garden/garden-app/clock"
	"github.com/calvinmclean/babyapi"

	"github.com/rs/xid"
)

var (
	ids = []string{
		"cqsnecmiuvoqlhrmf2jg",
		"cqsnecmiuvoqlhrmf2k0",
		"cqsnecmiuvoqlhrmf2kg",
		"cqsnecmiuvoqlhrmf2l0",
		"cqsnecmiuvoqlhrmf2lg",
		"cqsnecmiuvoqlhrmf2m0",
		"cqsnecmiuvoqlhrmf2mg",
		"cqsnecmiuvoqlhrmf2n0",
		"cqsnecmiuvoqlhrmf2ng",
		"cqsnecmiuvoqlhrmf2o0",
	}
	mockIDIndex   = 0
	enableMockIDs = false
)

// NewID creates a new unique xid by default, but mocking can be enabled to choose
// from a consistent list of IDs so results are repeatable
func NewID() babyapi.ID {
	if !enableMockIDs {
		return babyapi.ID{ID: xid.NewWithTime(clock.Now())}
	}

	id, err := xid.FromString(ids[mockIDIndex])
	if err != nil {
		panic(err)
	}
	mockIDIndex++

	return babyapi.ID{ID: id}
}
