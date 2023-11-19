package server

import (
	"bytes"
	"html/template"
	"strings"
)

const autoDarkModeJS = `
<script>
(() => {
	'use strict'

	const getPreferredTheme = () => {
		return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
	}

	const setTheme = theme => {
		document.documentElement.setAttribute('data-bs-theme', theme)
	}

	setTheme(getPreferredTheme())

	window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
		setTheme(getPreferredTheme())
	})
})()
</script>`

type HTMLer interface {
	HTML() string
}

func renderHTML(htmler HTMLer, data any) string {
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
	})

	templates = template.Must(templates.New("autoDarkModeJS").Parse(autoDarkModeJS))
	templates = template.Must(templates.New("innerHTML").Parse(htmler.HTML()))
	templates = template.Must(templates.New("GardenApp").Parse(`<!doctype html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Garden App</title>
	<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css">
	<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.10.5/font/bootstrap-icons.css">
	<script src="https://cdn.jsdelivr.net/npm/@popperjs/core@2.11.8/dist/umd/popper.min.js"></script>
	<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.min.js"></script>
	<script src="https://unpkg.com/htmx.org@1.9.8"></script>
	<script src="https://unpkg.com/htmx.org/dist/ext/json-enc.js"></script>
	
	{{ template "autoDarkModeJS" }}
</head>

<body>
<nav class="navbar navbar-expand-md bg-success" data-bs-theme="light">
<div class="container-fluid"><a class="navbar-brand" href="#/gardens">Garden App</a> <button
		class="navbar-toggler"><span class="navbar-toggler-icon"></span></button>
	<div class="navbar-collapse collapse show" style="">
		<ul class="ms-auto navbar-nav">
			<li class="nav-item"><a href="#/gardens" class="nav-link">Gardens</a></li>
			<li class="nav-item"><a href="#/water_schedules" class="nav-link">Water Schedules</a></li>
			<li class="nav-item"><a href="#/weather_clients" class="nav-link">Weather Clients</a></li>
		</ul>
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
