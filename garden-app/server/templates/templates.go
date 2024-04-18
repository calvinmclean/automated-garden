package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/go-chi/render"
)

type Template int

const (
	Gardens Template = iota
	EditGardenModal
	Zones
	ZoneDetails
	WaterSchedules
	WaterScheduleEditModal
	WaterScheduleModal
)

const (
	baseTemplate = `<!doctype html>
<html>
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Garden App</title>
	<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/uikit@3.17.11/dist/css/uikit.min.css" />
	<script src="https://cdn.jsdelivr.net/npm/uikit@3.19.2/dist/js/uikit.min.js"></script>
	<script src="https://cdn.jsdelivr.net/npm/uikit@3.19.2/dist/js/uikit-icons.min.js"></script>
	<script src="https://unpkg.com/htmx.org@1.9.8"></script>
	<script src="https://unpkg.com/hyperscript.org@0.9.12"></script>
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
</html>`
)

var (
	//go:embed gardens.html
	gardensHTML []byte

	//go:embed edit_garden_modal.html
	editGardenModalHTML []byte

	//go:embed zones.html
	zonesHTML []byte

	//go:embed zone_details.html
	zoneDetailsHTML []byte

	//go:embed water_schedules.html
	waterSchedulesHTML []byte

	//go:embed water_schedule_edit_modal.html
	waterScheduleEditModal []byte

	//go:embed water_schedule_modal.html
	waterScheduleModal []byte

	templateFilenames = map[Template]string{
		Gardens:                "server/templates/gardens.html",
		EditGardenModal:        "server/templates/edit_garden_modal.html",
		Zones:                  "server/templates/zones.html",
		ZoneDetails:            "server/templates/zone_details.html",
		WaterSchedules:         "server/templates/water_schedules.html",
		WaterScheduleEditModal: "server/templates/water_schedule_edit_modal.html",
		WaterScheduleModal:     "server/templates/water_schedule_modal.html",
	}

	templates = map[Template][]byte{
		Gardens:                gardensHTML,
		EditGardenModal:        editGardenModalHTML,
		Zones:                  zonesHTML,
		ZoneDetails:            zoneDetailsHTML,
		WaterSchedules:         waterSchedulesHTML,
		WaterScheduleEditModal: waterScheduleEditModal,
		WaterScheduleModal:     waterScheduleModal,
	}
)

func (t Template) templateString() string {
	tmpl, ok := templates[t]
	if !ok {
		panic("template not found")
	}

	if os.Getenv("DEV_TEMPLATE") == "true" {
		templateFilename, ok := templateFilenames[t]
		if !ok {
			panic("template not found")
		}

		var err error
		tmpl, err = os.ReadFile(templateFilename)
		if err != nil {
			panic(err)
		}
	}

	return string(tmpl)
}

func (t Template) Render(r *http.Request, data any, fullPage bool) string {
	tmpl := t.templateString()
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
	})

	templates = template.Must(templates.New("innerHTML").Parse(tmpl))
	if fullPage {
		templates = template.Must(templates.New("GardenApp").Parse(baseTemplate))
	}

	var renderedOutput bytes.Buffer
	err := templates.Execute(&renderedOutput, data)
	if err != nil {
		panic(err)
	}

	return renderedOutput.String()
}

func Renderer(t Template, data any) render.Renderer {
	return htmlRenderer{t: t, data: data}
}

type htmlRenderer struct {
	t    Template
	data any
}

func (h htmlRenderer) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func (h htmlRenderer) HTML(r *http.Request) string {
	return h.t.Render(r, h.data, false)
}
