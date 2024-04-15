package templates

import (
	"fmt"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

func formatDuration(d *pkg.Duration) string {
	days := d.Duration / (24 * time.Hour)
	remaining := d.Duration % (24 * time.Hour)

	remainingString := ""
	hours := int(remaining.Hours())
	remaining -= time.Duration(hours) * time.Hour

	minutes := int(remaining.Minutes()) % 60
	remaining -= time.Duration(minutes) * time.Minute

	seconds := int(remaining.Seconds()) % 3600

	if hours > 0 {
		remainingString += fmt.Sprintf("%dh", hours)
	}
	if minutes > 0 {
		remainingString += fmt.Sprintf("%dm", minutes)
	}
	if seconds > 0 {
		remainingString += fmt.Sprintf("%ds", seconds)
	}

	if days == 0 {
		return remainingString
	}

	if remainingString == "" {
		return fmt.Sprintf("%d days", days)
	}

	return fmt.Sprintf("%d days and %s", days, remainingString)
}
