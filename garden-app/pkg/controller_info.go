package pkg

import "time"

// ControllerInfo represents runtime information about a physical garden controller
type ControllerInfo struct {
	GardenID        string     `json:"-"`
	MACAddress      string     `json:"mac_address"`
	IPAddress       string     `json:"ip_address"`
	FirmwareVersion string     `json:"firmware_version"`
	UpdatedAt       *time.Time `json:"updated_at"`
}
