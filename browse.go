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

type Items struct {
	NextToken string `json:"next_token"`
	Count     int64  `json:"count"`
	Items     []Item `json:"items"`
}

type Item struct {
	Name         string `json:"name"`
	IsDir        bool   `json:"is_dir"`
	Key          string `json:"key"`
	Url          string `json:"url"`
	Size         string `json:"size"`
	LastModified string `json:"last_modified"`
}

func (i Items) GenerateJson(w http.ResponseWriter) error {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	err := json.NewEncoder(buf).Encode(i)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	buf.WriteTo(w)
	return nil
}

func (i Items) GenerateHtml(w http.ResponseWriter, template *template.Template) error {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	err := template.Execute(buf, i)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
	return nil
}

// This is a lame ass default template - needs to get better
const defaultBrowseTemplate = `<!DOCTYPE html>
<html>
        <body>
                <ul>
                {{- range .Items }}
                <li>
                {{- if .IsDir}}
                <a href="{{html .Url}}">{{html .Name}}/</a>
                {{- else}}
                <a href="{{html .Url}}">{{html .Name}}</a> Size: {{html .Size}} Last Modified: {{html .LastModified}}
                {{- end}}
                </li>
                {{- end }}
                </ul>
        </body>
</html>`
