package html

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/calvinmclean/automated-garden/garden-app/pkg"
	"github.com/go-chi/render"
)

type Template string

const (
	Gardens                Template = "Gardens"
	EditGardenModal        Template = "EditGardenModal"
	Zones                  Template = "Zones"
	ZoneDetails            Template = "ZoneDetails"
	WaterSchedules         Template = "WaterSchedules"
	WaterScheduleEditModal Template = "WaterScheduleEditModal"
	WaterScheduleModal     Template = "WaterScheduleModal"
)

var (
	//go:embed templates/*
	all embed.FS

	templateNames = map[Template]string{
		Gardens:                "Gardens",
		EditGardenModal:        "EditGardenModal",
		Zones:                  "Zones",
		ZoneDetails:            "ZoneDetails",
		WaterSchedules:         "WaterSchedules",
		WaterScheduleEditModal: "WaterScheduleEditModal",
		WaterScheduleModal:     "WaterScheduleModal",
	}
)

func (t Template) Name() string {
	return templateNames[t]
}

func (t Template) Render(r *http.Request, data any) string {
	// tmpl := t.templateString()
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

	if dir := os.Getenv("DEV_TEMPLATE"); dir != "" {
		templates = template.Must(templates.ParseGlob(dir + "/*"))
	} else {
		templates = template.Must(templates.ParseFS(all, "./*"))
	}

	var renderedOutput bytes.Buffer
	err := templates.ExecuteTemplate(&renderedOutput, t.Name(), data)
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
	return h.t.Render(r, h.data)
}
