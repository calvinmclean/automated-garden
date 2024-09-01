package sql

import (
	"github.com/calvinmclean/babyapi"
	"github.com/rs/xid"
)

func parseID(in string) (babyapi.ID, error) {
	id, err := xid.FromString(in)
	if err != nil {
		return babyapi.ID{}, err
	}
	return babyapi.ID{ID: id}, nil
}
