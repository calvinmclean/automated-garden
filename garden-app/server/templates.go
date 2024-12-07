package server

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/clock"
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
	weatherClientsPageTemplate       html.Template = "WeatherClientsPage"
	weatherClientsTemplate           html.Template = "WeatherClients"
	weatherClientModalTemplate       html.Template = "WeatherClientModal"
)

func templateFuncs(r *http.Request) map[string]any {
	return map[string]any{
		// args is used to create input maps when including sub-templates. It converts a slice to a map
		// by using N as the key and N+1 as a value
		"args": func(input ...any) map[string]any {
			result := map[string]any{}
			if len(input) < 2 {
				return result
			}

			for i := 0; i+1 < len(input); i++ {
				result[input[i].(string)] = input[i+1]
			}

			return result
		},
		"ToLower": strings.ToLower,
		"FormatUpcomingDate": func(date *time.Time) string {
			now := clock.Now()
			if date.YearDay() == now.YearDay() && date.Year() == now.Year() {
				return date.Format("at 3:04PM")
			}
			return date.Format("on Monday, 02 Jan at 3:04PM")
		},
		"FormatDateTime": func(date *time.Time) string {
			return date.Local().Format(time.DateTime)
		},
		"FormatStartTime": func(startTime *pkg.StartTime) string {
			return startTime.Time.Format(time.Kitchen)
		},
		"FormatTZOffset": func(startTime *pkg.StartTime) string {
			return startTime.Time.Format("Z07:00")
		},
		"FormatInt00": func(i int) string {
			return fmt.Sprintf("%02d", i)
		},
		"Sprintf": fmt.Sprintf,
		"CelsiusToFahrenheit": func(c float64) float64 {
			return c*1.8 + 32
		},
		"timeNow": func() time.Time {
			return clock.Now()
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
		"TZOffsetOptions": func(startTime *pkg.StartTime) []map[string]string {
			options := []map[string]string{
				{"Name": "UTC", "Value": "Z", "Selected": ""},
				{"Name": "UTC-07:00", "Value": "-07:00", "Selected": ""},
			}

			if startTime == nil {
				return options
			}

			for _, opt := range options {
				if opt["Value"] == startTime.Time.Format("Z07:00") {
					opt["Selected"] = "selected"
				}
			}

			return options
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
		"CompareNotificationClientID": func(ncID string, parent interface {
			GetNotificationClientID() string
		},
		) bool {
			if parent == nil {
				return false
			}
			return ncID == parent.GetNotificationClientID()
		},
		"ZoneQuickWater": func(z *ZoneResponse) []string {
			var waterDurations []string
			if z.NextWater.Duration == nil || z.NextWater.Duration.Duration == 0 {
				return append(waterDurations, "15m", "30m", "1h")
			}

			divideDuration := func(d time.Duration, f int) string {
				divided := time.Duration(int(d.Seconds())/f) * time.Second
				if divided > time.Minute {
					divided = (divided + 30*time.Second).Truncate(time.Minute)
				}
				return formatDuration(&pkg.Duration{Duration: divided})
			}

			baseDuration := z.NextWater.Duration.Duration
			waterDurations = append(waterDurations, divideDuration(baseDuration, 4))
			waterDurations = append(waterDurations, divideDuration(baseDuration, 2))
			waterDurations = append(waterDurations, divideDuration(baseDuration, 1))

			return waterDurations
		},
		"ExcludeWeatherData": func() bool {
			return excludeWeatherData(r)
		},
		"NotRefresh": func() bool {
			return r.URL.Query().Get("refresh") != "true"
		},
		"IncludePlusButton": func() bool {
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) == 0 {
				return false
			}
			_, err := xid.FromString(parts[len(parts)-1])
			return err != nil
		},
		"LightScheduleRange": func(ls *pkg.LightSchedule) map[int]string {
			result := map[int]string{}
			for i := 0; i < 24; i++ {
				selected := ""
				if ls != nil && ls.Duration.Hours() == float64(i) {
					selected = "selected"
				}
				result[i] = selected
			}
			return result
		},
		"UIntRange": func(n *uint) []uint {
			if n == nil {
				return []uint{}
			}
			result := make([]uint, *n)
			for i := range *n {
				result[i] = i
			}
			return result
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
