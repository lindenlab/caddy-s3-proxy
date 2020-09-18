package caddys3proxy

import (
	"errors"
	"io"
	"net/http"
	"path"
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

func (b *S3Proxy) Provision(ctx caddy.Context) (err error) {
	b.log = ctx.Logger(b)

	if b.Root == "" {
		b.Root = "{http.vars.root}"
	}

	if b.IndexNames == nil {
		b.IndexNames = defaultIndexNames
	}

	var config aws.Config

	// This is usually required for localstack and other
	// S3 alternatives, and I don't think there is any downside
	// when using it on AWS.  So we will always set it.
	config.S3ForcePathStyle = aws.Bool(true)

	// If Region is not specified NewSession will look for it from an env value AWS_REGION
	if b.Region != "" {
		config.Region = aws.String(b.Region)
	}

	if b.Endpoint != "" {
		config.Endpoint = aws.String(b.Endpoint)
	}

	sess, err := session.NewSession(&config)
	if err != nil {
		b.log.Error("could not create AWS session",
			zap.String("error", err.Error()),
		)
		return err
	}

	// Create S3 service client
	b.client = s3.New(sess)
	b.log.Info("S3 proxy initialized")

	return nil
}

func (b S3Proxy) getS3Object(bucket string, path string, rangeHeader *string) (*s3.GetObjectOutput, error) {
	oi := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
		Range:  rangeHeader,
	}
	b.log.Info("get from S3",
		zap.String("bucket", bucket),
		zap.String("key", path),
	)
	obj, err := b.client.GetObject(&oi)
	return obj, err
}

func joinPath(root string, uriPath string) string {
	isDir := uriPath[len(uriPath)-1:] == "/"
	newPath := path.Join(root, uriPath)
	if isDir && newPath != "/" {
		// Join will strip the ending / which we do not want
		return newPath + "/"
	}
	return newPath
}

func (b S3Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	// urlPath := r.URL.Path
	// root := repl.ReplaceAll(b.Root, "")
	// suffix := repl.ReplaceAll(urlPath, "")
	// fullPath := path.Join(root, filepath.FromSlash(path.Clean("/"+suffix)))
	// if fullPath == "" {
	//fullPath = "/"
	//}

	fullPath := joinPath(repl.ReplaceAll(b.Root, ""), r.URL.Path)

	b.log.Debug("path parts",
		// zap.String("root", root),
		zap.String("url path", r.URL.Path),
		zap.String("fullPath", fullPath),
	)

	// TODO: mayebe implement filtering out files (HiddenFiles)

	// Check for Range header
	var rangeHeader *string
	if rh := r.Header.Get("Range"); rh != "" {
		rangeHeader = aws.String(rh)
	}

	isDir := strings.HasSuffix(fullPath, "/")
	var obj *s3.GetObjectOutput
	var err error

	if isDir && len(b.IndexNames) > 0 {
		for _, indexPage := range b.IndexNames {
			indexPath := path.Join(fullPath, indexPage)
			obj, err = b.getS3Object(b.Bucket, indexPath, rangeHeader)
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
		obj, err = b.getS3Object(b.Bucket, fullPath, rangeHeader)
	}
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket,
				s3.ErrCodeNoSuchKey,
				s3.ErrCodeObjectNotInActiveTierError:
				// 404
				b.log.Error("not found",
					zap.String("bucket", b.Bucket),
					zap.String("key", fullPath),
					zap.String("err", aerr.Error()),
				)
				return caddyhttp.Error(http.StatusNotFound, aerr)
			default:
				// return 403 maybe?  Why else would it fail?
				b.log.Error("failed to get object",
					zap.String("bucket", b.Bucket),
					zap.String("key", fullPath),
					zap.String("err", aerr.Error()),
				)
				return caddyhttp.Error(http.StatusForbidden, err)
			}
		} else {
			b.log.Error("failed to get object",
				zap.String("bucket", b.Bucket),
				zap.String("key", fullPath),
				zap.String("err", err.Error()),
			)
			return caddyhttp.Error(http.StatusInternalServerError, err)
		}
	}

	// Copy heads from AWS response to our response
	setStrHeader(w, "Content-Disposition", obj.ContentDisposition)
	setStrHeader(w, "Content-Encoding", obj.ContentEncoding)
	setStrHeader(w, "Content-Language", obj.ContentLanguage)
	setStrHeader(w, "Content-Range", obj.ContentRange)
	setStrHeader(w, "Content-Type", obj.ContentType)
	setStrHeader(w, "ETag", obj.ETag)
	setTimeHeader(w, "Last-Modified", obj.LastModified)

	if obj.Body != nil {
		// io.Copy will set Content-Length
		w.Header().Del("Content-Length")
		if _, err := io.Copy(w, obj.Body); err != nil {
			return err
		}
	}

	return nil
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
