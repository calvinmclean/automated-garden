package server

import _ "embed"

//go:embed templates/water_schedules.html
var waterSchedulesHTML []byte

//go:embed templates/zones.html
var zonesHTML []byte

//go:embed templates/zone_details.html
var zoneDetailsHTML []byte

//go:embed templates/gardens.html
var gardensHTML []byte
