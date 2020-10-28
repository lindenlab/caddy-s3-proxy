package caddys3proxy

import (
	"encoding/json"
)

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

func (i Items) GenerateJson() string {
	bytes, _ := json.Marshal(i)
	return string(bytes)
}

func (i Items) GenerateHtml() string {
	// This is a total hack to show something.  This needs to change to use a template - and allow an
	// the template to be overridden by a user.  Doing this just for now to work out the data structures
	// and other stuff.
	html := "<!DOCTYPE html><html><body><ul>"
	for _, item := range i.Items {
		html += "<li><a href=\"" + item.Url + "\">" + item.Name + "</a>"
		if !item.IsDir {
			html += " Size: " + item.Size
			html += " Last Modified: " + item.LastModified
		}
		html += "</li>"
	}
	return html + "</ul></body></html>"
}