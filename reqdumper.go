package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/shu-go/gli"
)

// Version is app version
var Version string

type globalCmd struct {
	Port int `cli:"port,p" default:"12345"`

	LogFormat  string `cli:"logformat,lf" default:"json" help:"json|jsonindent|text|markdown"`
	FileFormat string `cli:"fileformat,ff" default:"file/{uri_asdir}/{filename}_{year}{month}{day}_{hour}{minute}{second}{nano}.{ext}" help:"see usage"`

	MDUseCodeQuote bool `cli:"md-use-codequote" default:"true"`
}

const (
	lfJSON       = "json"
	lfJSONIndent = "jsonindent"
	lfText       = "text"
	lfMarkdown   = "markdown"
)

func (c globalCmd) fileName(paramName, fileName, uri string) string {
	fn := c.FileFormat

	now := time.Now()
	fn = strings.ReplaceAll(fn, "{year}", fmt.Sprintf("%04d", now.Year()))
	fn = strings.ReplaceAll(fn, "{month}", fmt.Sprintf("%02d", now.Month()))
	fn = strings.ReplaceAll(fn, "{day}", fmt.Sprintf("%02d", now.Day()))
	fn = strings.ReplaceAll(fn, "{hour}", fmt.Sprintf("%02d", now.Hour()))
	fn = strings.ReplaceAll(fn, "{minute}", fmt.Sprintf("%02d", now.Minute()))
	fn = strings.ReplaceAll(fn, "{second}", fmt.Sprintf("%02d", now.Second()))
	fn = strings.ReplaceAll(fn, "{nanosecond}", fmt.Sprintf("%02d", now.Nanosecond()))
	fn = strings.ReplaceAll(fn, "{nano}", fmt.Sprintf("%06d", now.Nanosecond()))

	qpos := strings.Index(uri, "?")
	if qpos != -1 {
		uri = uri[:qpos]
	}
	uri = strings.TrimLeft(uri, "/")
	uriAsDir := filepath.FromSlash(uri)
	if !strings.HasSuffix(uriAsDir, string(filepath.Separator)) {
		uriAsDir += string(filepath.Separator)
	}
	fn = strings.ReplaceAll(fn, "{uri_asdir}", uriAsDir)
	fn = strings.ReplaceAll(fn, "{url_asdir}", uriAsDir)
	uri = strings.ReplaceAll(uri, "/", "_")
	fn = strings.ReplaceAll(fn, "{uri}", uri)
	fn = strings.ReplaceAll(fn, "{url}", uri)

	ext := filepath.Ext(fileName)
	if len(ext) > 0 {
		fileName = fileName[:len(fileName)-len(ext)]
	}
	fn = strings.ReplaceAll(fn, "{paramname}", paramName)
	fn = strings.ReplaceAll(fn, "{filename}", fileName)
	fn = strings.ReplaceAll(fn, "{ext}", ext)

	return fn
}

func (c *globalCmd) Before() error {
	c.LogFormat = strings.ToLower(c.LogFormat)
	if c.LogFormat != lfJSON && c.LogFormat != lfJSONIndent && c.LogFormat != lfText && c.LogFormat != lfMarkdown {
		return errors.New("logformat must be [" + strings.Join([]string{lfJSON, lfJSONIndent, lfText, lfMarkdown}, ", ") + "]")
	}
	return nil
}

func (c globalCmd) Run(args []string) error {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		var err error
		entry := newLogEntry()

		entry.URI = req.RequestURI
		entry.Protocol = req.Proto
		entry.Method = req.Method
		entry.RemoteAddr = req.RemoteAddr

		for k, v := range req.Header {
			entry.Header[k] = v
		}

		for k, v := range req.URL.Query() {
			entry.QueryParam[k] = v
		}

		for k, v := range req.Trailer {
			entry.Trailer[k] = v
		}

		if err = req.ParseForm(); err == nil {
			for k, v := range req.PostForm {
				entry.FormParam[k] = v
			}
		}

		if err = req.ParseMultipartForm(32 << 20); err == nil {
			for k, v := range req.MultipartForm.Value {
				entry.FormParam[k] = v
			}
			for paramName, v := range req.MultipartForm.File {
				wd, err := os.Getwd()
				if err != nil {
					return
				}

				names := []string{}
				for _, fv := range v {
					fileName := fv.Filename
					names = append(names, fileName)

					destName := c.fileName(paramName, fileName, req.RequestURI)
					if !filepath.IsAbs(destName) {
						destName = filepath.Join(wd, destName)
					}

					dir := filepath.Dir(destName)
					err := os.MkdirAll(dir, os.ModePerm)
					if err != nil {
						fmt.Fprintf(os.Stderr, "error: %v\n", err)
						return
					}

					f, err := os.Create(destName)
					if err != nil {
						fmt.Fprintf(os.Stderr, "error: %v\n", err)
						f.Close()
						return
					}
					fsrc, err := fv.Open()
					if err != nil {
						fmt.Fprintf(os.Stderr, "error: %v\n", err)
						fsrc.Close()
						f.Close()
						return
					}
					io.Copy(f, fsrc)
					fsrc.Close()
					f.Close()
				}
				entry.File[paramName] = names
			}
		}

		var body bytes.Buffer
		_, err = io.Copy(&body, req.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		entry.Body = body.String()

		out := os.Stdout
		switch c.LogFormat {
		case lfJSON:
			err = entry.writeAsJSON(out)
		case lfJSONIndent:
			err = entry.writeAsJSONIndent(out)
		case lfText:
			err = entry.writeAsText(out)
		case lfMarkdown:
			err = entry.writeAsMarkdown(out, c.MDUseCodeQuote)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
	})

	return http.ListenAndServe(":"+strconv.Itoa(c.Port), nil)
}

func main() {
	app := gli.NewWith(&globalCmd{})
	app.Name = "reqdumper"
	app.Desc = "Dumps HTTP requests"
	app.Version = Version
	app.Usage = `--fileformat:
    {year}{month}{day}: date
    {hour}{minute}{second}{nanosecond}{nano}: time
    {uri}{url}: URI joined by _ instead of /
    {uri_asdir}{url_asdir}: URI joined by a path delimiter instead of /
    {paramname}: param name
    {filename}: file name without ext
    {ext}: the suffix beginning at the final dot in the filename`
	app.Copyright = "(C) 2022 Shuhei Kubota"
	app.SuppressErrorOutput = true
	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}
