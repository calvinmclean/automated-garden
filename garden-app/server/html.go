package server

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"
)

func renderTemplate(tmpl string, data any) string {
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
		"Sprintf": fmt.Sprintf,
		"CelsiusToFahrenheit": func(c float64) float64 {
			return c*1.8 + 32
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
	<script src="https://unpkg.com/htmx.org/dist/ext/json-enc.js"></script>
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
                    <li class="uk-active"><a href="/gardens">Gardens</a></li>
                    <li><a href="/water_schedules">Water Schedules</a></li>
					<li><a href="/weather_clients">Weather Clients</a></li>
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
