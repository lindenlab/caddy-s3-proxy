package caddys3proxy

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var defaultIndexNames = []string{"index.html", "index.txt"}

var awsErrorCodesMapping = map[string]int{
	"NotModified":        http.StatusNotModified,
	"PreconditionFailed": http.StatusPreconditionFailed,
	"InvalidRange":       http.StatusRequestedRangeNotSatisfiable,
}

func init() {
	caddy.RegisterModule(S3Proxy{})
}

// S3Proxy implements a proxy to return objects from S3
type S3Proxy struct {
	// The path to the root of the site. Default is `{http.vars.root}` if set,
	// Or if not set the value is "" - meaning use the whole path as a key.
	Root string `json:"root,omitempty"`

	// The AWS region the bucket is hosted in
	Region string `json:"region,omitempty"`

	// The name of the S3 bucket
	Bucket string `json:"bucket,omitempty"`

	// Use non-standard endpoint for S3
	Endpoint string `json:"endpoint,omitempty"`

	// The names of files to try as index files if a folder is requested.
	IndexNames []string `json:"index_names,omitempty"`

	// A glob pattern used to hide matching key paths (returning a 404)
	Hide []string

	// Flag to determine if PUT operations are allowed (default false)
	EnablePut bool

	// Flag to determine if DELETE operations are allowed (default false)
	EnableDelete bool

	// Mapping of HTTP error status to S3 keys.
	ErrorPages map[int]string `json:"error_pages,omitempty"`

	// S3 key to a default error page.
	DefaultErrorPage string `json:"default_error_page,omitempty"`

	client *s3.S3
	log    *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (S3Proxy) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.s3proxy",
		New: func() caddy.Module { return new(S3Proxy) },
	}
}

func (p *S3Proxy) Provision(ctx caddy.Context) (err error) {
	p.log = ctx.Logger(p)

	if p.Root == "" {
		p.Root = "{http.vars.root}"
	}

	if p.IndexNames == nil {
		p.IndexNames = defaultIndexNames
	}

	if p.ErrorPages == nil {
		p.ErrorPages = make(map[int]string)
	}

	var config aws.Config

	// This is usually required for localstack and other
	// S3 alternatives, and I don't think there is any downside
	// when using it on AWS.  So we will always set it.
	config.S3ForcePathStyle = aws.Bool(true)

	// If Region is not specified NewSession will look for it from an env value AWS_REGION
	if p.Region != "" {
		config.Region = aws.String(p.Region)
	}

	if p.Endpoint != "" {
		config.Endpoint = aws.String(p.Endpoint)
	}

	sess, err := session.NewSession(&config)
	if err != nil {
		p.log.Error("could not create AWS session",
			zap.String("error", err.Error()),
		)
		return err
	}

	// Create S3 service client
	p.client = s3.New(sess)
	p.log.Info("S3 proxy initialized for bucket: " + p.Bucket)
	p.log.Debug("config values",
		zap.String("endpoint", p.Endpoint),
		zap.String("region", p.Region),
		zap.Bool("enable_put", p.EnablePut),
		zap.Bool("enable_delete", p.EnableDelete),
	)

	return nil
}

func (p S3Proxy) getS3Object(bucket string, path string, headers http.Header) (*s3.GetObjectOutput, error) {
	oi := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	}

	if rg := headers.Get("Range"); rg != "" {
		oi = oi.SetRange(rg)
	}
	if ifMatch := headers.Get("If-Match"); ifMatch != "" {
		oi = oi.SetIfMatch(ifMatch)
	}
	if ifNoneMatch := headers.Get("If-None-Match"); ifNoneMatch != "" {
		oi = oi.SetIfNoneMatch(ifNoneMatch)
	}
	if ifModifiedSince := headers.Get("If-Modified-Since"); ifModifiedSince != "" {
		t, err := time.Parse(http.TimeFormat, ifModifiedSince)
		if err == nil {
			oi = oi.SetIfModifiedSince(t)
		}
	}
	if ifUnmodifiedSince := headers.Get("If-Unmodified-Since"); ifUnmodifiedSince != "" {
		t, err := time.Parse(http.TimeFormat, ifUnmodifiedSince)
		if err == nil {
			oi = oi.SetIfUnmodifiedSince(t)
		}
	}

	p.log.Info("get from S3",
		zap.String("bucket", bucket),
		zap.String("key", path),
	)

	return p.client.GetObject(oi)
}

func joinPath(root string, uriPath string) string {
	isDir := uriPath[len(uriPath)-1:] == "/"
	newPath := path.Join(root, uriPath)
	if isDir && newPath != "/" {
		// Join will strip the ending /
		// add it back if it was there as it implies a dir view
		return newPath + "/"
	}
	return newPath
}

func makeAwsString(str string) *string {
	if str == "" {
		return nil
	}
	return aws.String(str)
}

func (p S3Proxy) HandlePut(w http.ResponseWriter, r *http.Request, key string) error {
	isDir := strings.HasSuffix(key, "/")
	if isDir || !p.EnablePut {
		err := errors.New("method not allowed")
		return caddyhttp.Error(http.StatusMethodNotAllowed, err)
	}

	// The request gives us r.Body a ReadCloser.  However, Put need a ReadSeeker.
	// So we need to read the entire object in memory and create the ReadSeeker.
	// TODO: this will not work well for very large files - will run out of memory
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	oi := s3.PutObjectInput{
		Bucket:             aws.String(p.Bucket),
		Key:                aws.String(key),
		CacheControl:       makeAwsString(r.Header.Get("Cache-Control")),
		ContentDisposition: makeAwsString(r.Header.Get("Content-Disposition")),
		ContentEncoding:    makeAwsString(r.Header.Get("Content-Encoding")),
		ContentLanguage:    makeAwsString(r.Header.Get("Content-Language")),
		ContentType:        makeAwsString(r.Header.Get("Content-Type")),
		Body:               bytes.NewReader(buf),
	}
	po, err := p.client.PutObject(&oi)
	if err != nil {
		return err
	}

	setStrHeader(w, "ETag", po.ETag)

	return nil
}

func (p S3Proxy) HandleDelete(w http.ResponseWriter, r *http.Request, key string) error {
	isDir := strings.HasSuffix(key, "/")
	if isDir || !p.EnableDelete {
		err := errors.New("method not allowed")
		return caddyhttp.Error(http.StatusMethodNotAllowed, err)
	}

	di := s3.DeleteObjectInput{
		Bucket: aws.String(p.Bucket),
		Key:    aws.String(key),
	}
	_, err := p.client.DeleteObject(&di)
	if err != nil {
		return err
	}

	return nil
}

func (p S3Proxy) writeResponseFromGetObject(w http.ResponseWriter, obj *s3.GetObjectOutput) error {
	// Copy headers from AWS response to our response
	setStrHeader(w, "Content-Disposition", obj.ContentDisposition)
	setStrHeader(w, "Content-Encoding", obj.ContentEncoding)
	setStrHeader(w, "Content-Language", obj.ContentLanguage)
	setStrHeader(w, "Content-Range", obj.ContentRange)
	setStrHeader(w, "Content-Type", obj.ContentType)
	setStrHeader(w, "ETag", obj.ETag)
	setTimeHeader(w, "Last-Modified", obj.LastModified)

	var err error
	if obj.Body != nil {
		// io.Copy will set Content-Length
		w.Header().Del("Content-Length")
		_, err = io.Copy(w, obj.Body)
	}

	return err
}

func (p S3Proxy) serveErrorPage(w http.ResponseWriter, s3Key string) error {
	obj, err := p.getS3Object(p.Bucket, s3Key, nil)
	if err != nil {
		return err
	}

	if err := p.writeResponseFromGetObject(w, obj); err != nil {
		return err
	}

	return nil
}

func (p S3Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	return p.wrapHTTPErrors(w, p.doServeHTTP(w, r, next))
}

func (p S3Proxy) doServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	fullPath := joinPath(repl.ReplaceAll(p.Root, ""), r.URL.Path)

	// If file is hidden - return 404
	if fileHidden(fullPath, p.Hide) {
		return caddyhttp.Error(http.StatusNotFound, nil)
	}

	switch r.Method {
	case http.MethodGet:
		break
	case http.MethodPut:
		return p.HandlePut(w, r, fullPath)
	case http.MethodDelete:
		return p.HandleDelete(w, r, fullPath)
	default:
		err := errors.New("method not allowed")
		return caddyhttp.Error(http.StatusMethodNotAllowed, err)
	}

	// TODO: mayebe implement filtering out files (HiddenFiles)

	isDir := strings.HasSuffix(fullPath, "/")
	var obj *s3.GetObjectOutput
	var err error

	if isDir && len(p.IndexNames) > 0 {
		for _, indexPage := range p.IndexNames {
			indexPath := path.Join(fullPath, indexPage)
			obj, err = p.getS3Object(p.Bucket, indexPath, r.Header)
			if obj != nil {
				// We found an index!
				isDir = false
				break
			}
		}
	}

	// If this is still a dir then browse or throw an error
	if isDir {
		// TODO: implement browse
		err := errors.New("can not view a directory")
		return caddyhttp.Error(http.StatusForbidden, err)
	}

	// Get the obj from S3 (skip if we already did when looking for an index)
	if obj == nil {
		obj, err = p.getS3Object(p.Bucket, fullPath, r.Header)
	}
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket,
				s3.ErrCodeNoSuchKey,
				s3.ErrCodeObjectNotInActiveTierError:
				// 404
				p.log.Debug("not found",
					zap.String("bucket", p.Bucket),
					zap.String("key", fullPath),
					zap.String("err", aerr.Error()),
				)
				return caddyhttp.Error(http.StatusNotFound, aerr)
			default:
				if code, ok := awsErrorCodesMapping[aerr.Code()]; ok {
					return caddyhttp.Error(code, nil)
				}

				// return 403 maybe?  Why else would it fail?
				p.log.Error("failed to get object",
					zap.String("bucket", p.Bucket),
					zap.String("key", fullPath),
					zap.String("err", aerr.Error()),
				)

				return caddyhttp.Error(http.StatusForbidden, err)
			}
		} else {
			p.log.Error("failed to get object",
				zap.String("bucket", p.Bucket),
				zap.String("key", fullPath),
				zap.String("err", err.Error()),
			)

			return err
		}
	}

	return p.writeResponseFromGetObject(w, obj)
}

func (p S3Proxy) wrapHTTPErrors(w http.ResponseWriter, parentError error) error {
	if parentError == nil {
		return nil
	}

	caddyErr, isCaddyErr := parentError.(caddyhttp.HandlerError)

	if !isCaddyErr {
		caddyErr = caddyhttp.Error(http.StatusInternalServerError, parentError)
	}

	if caddyErr.StatusCode != 0 {
		w.WriteHeader(caddyErr.StatusCode)
	}

	var s3Key string
	if errorPageS3Key, hasErrorPageForCode := p.ErrorPages[caddyErr.StatusCode]; hasErrorPageForCode {
		s3Key = errorPageS3Key
	} else if p.DefaultErrorPage != "" {
		s3Key = p.DefaultErrorPage
	}

	if s3Key != "" {
		if err := p.serveErrorPage(w, s3Key); err != nil {
			// Just log the error as we don't want to swallow the parent error.
			p.log.Error("error serving error page",
				zap.String("bucket", p.Bucket),
				zap.String("key", s3Key),
				zap.String("err", err.Error()),
			)
		}
	}

	return caddyErr
}

func setStrHeader(w http.ResponseWriter, key string, value *string) {
	if value != nil && len(*value) > 0 {
		w.Header().Add(key, *value)
	}
}

func setTimeHeader(w http.ResponseWriter, key string, value *time.Time) {
	if value != nil && !reflect.DeepEqual(*value, time.Time{}) {
		w.Header().Add(key, value.UTC().Format(http.TimeFormat))
	}
}

// fileHidden returns true if filename is hidden
// according to the hide list.
func fileHidden(filename string, hide []string) bool {
	sep := string(filepath.Separator)
	var components []string

	for _, h := range hide {
		if !strings.Contains(h, sep) {
			// if there is no separator in h, then we assume the user
			// wants to hide any files or folders that match that
			// name; thus we have to compare against each component
			// of the filename, e.g. hiding "bar" would hide "/bar"
			// as well as "/foo/bar/baz" but not "/barstool".
			if len(components) == 0 {
				components = strings.Split(filename, sep)
			}
			for _, c := range components {
				if c == h {
					return true
				}
			}
		} else if strings.HasPrefix(filename, h) {
			// otherwise, if there is a separator in h, and
			// filename is exactly prefixed with h, then we
			// can do a prefix match so that "/foo" matches
			// "/foo/bar" but not "/foobar".
			withoutPrefix := strings.TrimPrefix(filename, h)
			if strings.HasPrefix(withoutPrefix, sep) {
				return true
			}
		}

		// in the general case, a glob match will suffice
		if hidden, _ := filepath.Match(h, filename); hidden {
			return true
		}
	}

	return false
}
