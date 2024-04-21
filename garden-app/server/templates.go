package server

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/calvinmclean/babyapi"
	"github.com/calvinmclean/babyapi/html"
	"github.com/rs/xid"
)

//go:embed templates/*
var templates embed.FS

const (
	gardensPageTemplate              html.Template = "GardensPage"
	gardensTemplate                  html.Template = "Gardens"
	gardenModalTemplate              html.Template = "GardenModal"
	zonesPageTemplate                html.Template = "ZonesPage"
	zonesTemplate                    html.Template = "Zones"
	zoneDetailsTemplate              html.Template = "ZoneDetails"
	waterSchedulesPageTemplate       html.Template = "WaterSchedulesPage"
	waterSchedulesTemplate           html.Template = "WaterSchedules"
	waterScheduleModalTemplate       html.Template = "WaterScheduleModal"
	waterScheduleDetailModalTemplate html.Template = "WaterScheduleDetailModal"
	zoneModalTemplate                html.Template = "ZoneModal"
	zoneActionModalTemplate          html.Template = "ZoneActionModal"
)

func templateFuncs(r *http.Request) map[string]any {
	return map[string]any{
		// args is used to create input maps when including sub-templates. It converts a slice to a map
		// by using N as the key and N+1 as a value
		"args": func(input ...string) map[string]any {
			result := map[string]any{}
			if len(input) < 2 {
				return result
			}

			for i := 0; i+1 < len(input); i++ {
				result[input[i]] = input[i+1]
			}

			return result
		},
		"ToLower": strings.ToLower,
		"FormatUpcomingDate": func(date *time.Time) string {
			now := time.Now()
			if date.YearDay() == now.YearDay() && date.Year() == now.Year() {
				return date.Format("at 3:04PM")
			}
			return date.Format("on Monday, 02 Jan at 3:04PM")
		},
		"FormatDateTime": func(date *time.Time) string {
			return date.Local().Format(time.DateTime)
		},
		"FormatTimeOnly": func(date *time.Time) string {
			return date.Local().Format(time.Kitchen)
		},
		"Sprintf": fmt.Sprintf,
		"CelsiusToFahrenheit": func(c float64) float64 {
			return c*1.8 + 32
		},
		"timeNow": func() time.Time {
			return time.Now()
		},
		"URLContains": func(input string) bool {
			return strings.Contains(r.URL.Path, input)
		},
		"FormatDuration": formatDuration,
		"ShortMonth": func(month string) string {
			return month[0:3]
		},
		"RFC3339Nano": func(t *time.Time) string {
			if t == nil {
				return ""
			}
			return t.Format(time.RFC3339Nano)
		},
		"MonthRows": func(ap *pkg.ActivePeriod, startMonth bool) template.HTML {
			var sb strings.Builder

			start := "Start being active in..."
			selected := ""
			if !startMonth {
				start = "Stop being active after..."
			}
			if ap == nil {
				selected = "selected"
			}

			sb.WriteString(fmt.Sprintf("<option value=\"\" disabled %s>%s</option>\n", selected, start))

			for month := time.January; month <= time.December; month++ {
				format := `<option value="%s">%s</option>`

				if ap != nil {
					selected := ap.StartMonth == month.String()
					if !startMonth {
						selected = ap.EndMonth == month.String()
					}

					if selected {
						format = `<option value="%s" selected>%s</option>`
					}
				}

				sb.WriteString(fmt.Sprintf(format, month.String(), month.String()))
				sb.WriteString("\n")
			}

			//nolint:gosec
			return template.HTML(sb.String())
		},
		"URLPath": func() string {
			return r.URL.Path
		},
		"QueryParam": func(key string) string {
			return r.URL.Query().Get(key)
		},
		"ContainsID": func(items []xid.ID, target babyapi.ID) bool {
			return slices.ContainsFunc(items, func(item xid.ID) bool {
				return item.String() == target.String()
			})
		},
	}
}

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
