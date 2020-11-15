package caddys3proxy

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dustin/go-humanize"
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

func (p S3Proxy) ConstructListObjInput(r *http.Request, key string) s3.ListObjectsV2Input {
	// We should only get here if the path ends in a /, however, when we make the
	//call to ListObjects no / should be there
	prefix := strings.TrimSuffix(key, "/")

	input := s3.ListObjectsV2Input{
		Bucket:    aws.String(p.Bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	nextToken := r.URL.Query().Get("next")
	if nextToken != "" {
		input.ContinuationToken = aws.String(nextToken)
	}

	maxPerPage := r.URL.Query().Get("max")
	if maxPerPage != "" {
		maxKeys, err := strconv.ParseInt(maxPerPage, 10, 64)
		if err == nil && maxKeys > 0 && maxKeys <= 1000 {
			input.MaxKeys = aws.Int64(maxKeys)
		}
	}

	return input
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

func (p S3Proxy) MakePageObj(result *s3.ListObjectsV2Output) PageObj {
	po := PageObj{}
	po.Count = *result.KeyCount
	if result.NextContinuationToken != nil {
		var nextUrl url.URL
		queryItems := nextUrl.Query()

		queryItems.Add("next", *result.NextContinuationToken)
		if result.MaxKeys != nil {
			queryItems.Add("max", strconv.FormatInt(*result.MaxKeys, 10))
		}
		nextUrl.RawQuery = queryItems.Encode()
		po.MoreLink = nextUrl.String()
	}

	for _, dir := range result.CommonPrefixes {
		name := path.Base(*dir.Prefix)
		dirPath := "./" + name + "/"
		po.Items = append(po.Items, Item{
			Url:   dirPath,
			Name:  name,
			IsDir: true,
		})
	}
	for _, obj := range result.Contents {
		name := path.Base(*obj.Key)
		itemPath := "./" + name
		size := humanize.Bytes(uint64(*obj.Size))
		timeAgo := humanize.Time(*obj.LastModified)
		po.Items = append(po.Items, Item{
			Name:         name,
			Key:          *obj.Key,
			Url:          itemPath,
			Size:         size,
			LastModified: timeAgo,
			IsDir:        false,
		})
	}

	return po
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
