package server

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
)

func renderTemplate(r *http.Request, tmpl string, data any) string {
	templates := template.New("base").Funcs(map[string]any{
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
	})

	templates = template.Must(templates.New("innerHTML").Parse(tmpl))
	// TODO: use current URL from request to set class="uk-active" in navbar
	templates = template.Must(templates.New("GardenApp").Parse(`<!doctype html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Garden App</title>
	<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/uikit@3.17.11/dist/css/uikit.min.css" />
	<script src="https://cdn.jsdelivr.net/npm/uikit@3.19.2/dist/js/uikit.min.js"></script>
	<script src="https://cdn.jsdelivr.net/npm/uikit@3.19.2/dist/js/uikit-icons.min.js"></script>
	<script src="https://unpkg.com/htmx.org@1.9.8"></script>
</head>

<style>
	div.htmx-swapping {
		opacity: 0;
		transition: opacity 1s ease-out;
	}
</style>

<body>
<nav class="uk-navbar-container">
    <div class="uk-container">
        <div uk-navbar>
            <div class="uk-navbar-left">
                <ul class="uk-navbar-nav">
                    <li {{ if URLContains "/gardens" }}class="uk-active"{{ end }}><a href="/gardens">Gardens</a></li>
                    <li {{ if URLContains "/water_schedules" }}class="uk-active"{{ end }}><a href="/water_schedules">Water Schedules</a></li>
					<li {{ if URLContains "/weather_clients" }}class="uk-active"{{ end }}><a href="/weather_clients">Weather Clients</a></li>
                </ul>
            </div>
        </div>
    </div>
</nav>

{{template "innerHTML" .}}

</body>
</html>`))

	var renderedOutput bytes.Buffer
	err := templates.Execute(&renderedOutput, data)
	if err != nil {
		panic(err)
	}

	return renderedOutput.String()
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
