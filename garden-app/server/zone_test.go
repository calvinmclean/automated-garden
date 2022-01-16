package server

import (
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/rs/xid"
)

func createExampleZone() *pkg.Zone {
	time, _ := time.Parse(time.RFC3339Nano, "2021-10-03T11:24:52.891386-07:00")
	id, _ := xid.FromString("c5cvhpcbcv45e8bp16dg")
	return &pkg.Zone{
		Name:      "test-zone",
		ID:        id,
		CreatedAt: &time,
	}
}
