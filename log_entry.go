package main

import (
	"encoding/json"
	"io"
	"text/template"
	"time"
)

type logEntry struct {
	Timestamp time.Time

	URI        string
	Protocol   string
	Method     string
	RemoteAddr string

	Header     map[string]param
	QueryParam map[string]param
	FormParam  map[string]param
	File       map[string]param
	Body       string
	Trailer    map[string]param
}

func newLogEntry() *logEntry {
	return &logEntry{
		Timestamp: time.Now(),

		Header:     make(map[string]param),
		QueryParam: make(map[string]param),
		FormParam:  make(map[string]param),
		File:       make(map[string]param),
		Trailer:    make(map[string]param),
	}
}

func (l logEntry) writeAsJSON(out io.Writer) error {
	b, err := json.Marshal(l)
	if err != nil {
		return err
	}
	_, err = out.Write(b)
	if err != nil {
		return err
	}
	_, err = out.Write([]byte("\n"))
	return err
}

func (l logEntry) writeAsJSONIndent(out io.Writer) error {
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	_, err = out.Write(b)
	if err != nil {
		return err
	}
	_, err = out.Write([]byte("\n"))
	return err
}

func (l logEntry) writeAsText(out io.Writer) error {
	templ := template.Must(template.New("logEntry").Parse(`{{.Timestamp.Format "2006-01-02T15:04:05.999999999Z07:00"}}
    {{.Method}} {{.URI}} {{.Protocol}}
{{range $k,$v := .Header}}    {{$k}}: {{$v}}
{{end}}
    {{.Body}}

{{range $k,$v := .Trailer -}}    {{$k}}: {{$v}}
{{end}}`))

	return templ.Execute(out, l)
}

func (l logEntry) writeAsMarkdown(out io.Writer, useCodeQuote bool) error {
	funcMap := map[string]any{
		"bt":  func() any { return "`" },
		"bt3": func() any { return "```" },
	}
	if !useCodeQuote {
		funcMap["bt"] = func() any { return "" }
		funcMap["bt3"] = func() any { return "" }
	}

	templ := template.Must(template.New("logEntry").Funcs(funcMap).Parse(`## {{.Timestamp.Format "2006-01-02T15:04:05.999999999Z07:00"}}

### Request

{{.Method}} {{.URI}} {{.Protocol}}

{{if ne (len .QueryParam) 0 -}}
#### Query Parameters

| name | value |
|------|-------|
{{range $k,$v := .QueryParam}}| {{bt}}{{$k}}{{bt}} | {{bt}}{{$v}}{{bt}} |
{{end}}
{{end -}}

{{if ne (len .RemoteAddr) 0 -}}
#### Remote Address

{{bt}}{{.RemoteAddr}}{{bt}}

{{end -}}
{{if ne (len .Header) 0 -}}
### Headers

| name | value |
|------|-------|
{{range $k,$v := .Header}}| {{bt}}{{$k}}{{bt}} | {{bt}}{{$v}}{{bt}} |
{{end}}
{{end -}}
{{if ne (len .Body) 0 -}}
### Body

{{bt3}}
{{.Body}}
{{bt3}}
{{end -}}
{{if ne (len .FormParam) 0 -}}
### Form Parameters

| name | value |
|------|-------|
{{range $k,$v := .FormParam}}| {{bt}}{{$k}}{{bt}} | {{bt}}{{$v}}{{bt}} |
{{end}}
{{end -}}
{{if ne (len .File) 0 -}}
### Form Parameters (Files)

| name | value |
|------|-------|
{{range $k,$v := .File}}| {{bt}}{{$k}}{{bt}} | {{bt}}{{$v}}{{bt}} |
{{end}}
{{end -}}
{{if ne (len .Trailer) 0 -}}
### Trailers

| name | value |
|------|-------|
{{range $k,$v := .Trailer -}}| {{bt}}{{$k}}{{bt}} | {{bt}}{{$v}}{{bt}} |
{{end}}
{{- end -}}`))

	return templ.Execute(out, l)
}
