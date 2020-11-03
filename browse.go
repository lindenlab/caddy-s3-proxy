package caddys3proxy

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"sync"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

type PageObj struct {
	NextToken string `json:"next_token"`
	Count     int64  `json:"count"`
	Items     []Item `json:"items"`
	MoreLink  string `json:"more"`
}

type Item struct {
	Name         string `json:"name"`
	IsDir        bool   `json:"is_dir"`
	Key          string `json:"key"`
	Url          string `json:"url"`
	Size         string `json:"size"`
	LastModified string `json:"last_modified"`
}

// GenerateJson generates JSON output for the PageObj
func (po PageObj) GenerateJson(w http.ResponseWriter) error {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	err := json.NewEncoder(buf).Encode(po)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err = buf.WriteTo(w)
	return err
}

// GenerateHtml generates html output for the PageObj
func (po PageObj) GenerateHtml(w http.ResponseWriter, template *template.Template) error {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	err := template.Execute(buf, po)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = buf.WriteTo(w)
	return err
}

// This is a lame ass default template - needs to get better
const defaultBrowseTemplate = `<!DOCTYPE html>
<html>
        <body>
                <ul>
                {{- range .PageObj }}
                <li>
                {{- if .IsDir}}
                <a href="{{html .Url}}">{{html .Name}}</a>
                {{- else}}
                <a href="{{html .Url}}">{{html .Name}}</a> Size: {{html .Size}} Last Modified: {{html .LastModified}}
                {{- end}}
                </li>
                {{- end }}
                </ul>
		<p>number of items: {{ .Count }}</p>
		{{- if .MoreLink }}
		<a href="{{ html .MoreLink }}">more...</a>
		{{- end }}
        </body>
</html>`
