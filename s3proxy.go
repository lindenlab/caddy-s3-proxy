package caddys3proxy

import (
	"errors"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

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

// S3Proxy implements a static file server responder for Caddy.
type S3Proxy struct {
	// The prefix to prepend to paths when looking for objects in S3
	Prefix string `json:"prefix,omitempty"`

	// The AWS region the bucket is hosted in
	Region string `json:"region,omitempty"`

	// The name of the S3 bucket
	Bucket string `json:"bucket,omitempty"`

	// In insecure is true - disable TLS in the client
	Insecure string `json:"insecure,omitempty"`

	// Use non-standard endpoint for S3
	Endpoint string `json:"endpoint,omitempty"`

	// A list of files or folders to hide; the file server will pretend as if
	// they don't exist. Accepts globular patterns like "*.hidden" or "/foo/*/bar".
	Hide []string `json:"hide,omitempty"`

	// The names of files to try as index files if a folder is requested.
	IndexNames []string `json:"index_names,omitempty"`

	// Use redirects to enforce trailing slashes for directories, or to
	// remove trailing slash from URIs for files. Default is true.
	CanonicalURIs *bool `json:"canonical_uris,omitempty"`

	// If pass-thru mode is enabled and a requested file is not found,
	// it will invoke the next handler in the chain instead of returning
	// a 404 error. By default, this is false (disabled).
	PassThru bool `json:"pass_thru,omitempty"`

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

	if b.IndexNames == nil {
		b.IndexNames = defaultIndexNames
	}

	var config aws.Config

	// If Region is not specified NewSession will look for it from an env value AWS_REGION
	if b.Region != "" {
		config.Region = aws.String(b.Region)
	}

	if b.Endpoint != "" {
		config.Endpoint = aws.String(b.Endpoint)
	}

	if b.Insecure != "" {
		// insecure, err := strconv.ParseBool(b.Insecure)
		// if err == nil && insecure == true {
		//config.DisableSLL = awws.Bool(true)
		//}
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

func (b S3Proxy) getS3Object(bucket string, path string) (*s3.GetObjectOutput, error) {
	oi := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	}
	b.log.Info("attempting to get",
		zap.String("bucket", bucket),
		zap.String("key", path),
	)
	obj, err := b.client.GetObject(&oi)
	return obj, err
}

func (b S3Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// TODO: Handle path manipulation (Root, Prefix, HiddenFiles, etc.)
	fullPath := r.URL.Path
	if fullPath == "" {
		fullPath = "/"
	}

	b.log.Info("In ServeHTTP for s3proxy")

	isDir := strings.HasSuffix(fullPath, "/")
	var obj *s3.GetObjectOutput
	var err error

	if isDir && len(b.IndexNames) > 0 {
		b.log.Info("isDir and looking for index")
		for _, indexPage := range b.IndexNames {
			indexPath := path.Join(fullPath, indexPage)
			obj, err = b.getS3Object(b.Bucket, indexPath)
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
		err := errors.New("browse not configured")
		return caddyhttp.Error(http.StatusForbidden, err)
	}

	// TODO: what to do amount weird method types
	// TODO: How to determine if a "dir"
	// TODO: render a "dir" with a template and get a list of objects with stats

	// Get the obj from S3 (skip if we already did when looking for an index)
	if obj == nil {
		obj, err = b.getS3Object(b.Bucket, fullPath)
	}
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				// 404
				b.log.Error("bucket not found",
					zap.String("bucket", b.Bucket),
				)
				return caddyhttp.Error(http.StatusNotFound, nil)
			case s3.ErrCodeNoSuchKey:
			case s3.ErrCodeObjectNotInActiveTierError:
				// 404
				b.log.Error("key not found",
					zap.String("key", fullPath),
				)
				return caddyhttp.Error(http.StatusNotFound, nil)
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

	b.log.Info("content type",
		zap.String("value", obj.String()),
	)
	w.Header().Set("Content-Type", aws.StringValue(obj.ContentType))
	w.Header().Set("Content-Length", strconv.FormatInt(aws.Int64Value(obj.ContentLength), 10))
	if obj.Body != nil {
		if _, err := io.Copy(w, obj.Body); err != nil {
			return err
		}
	}

	return nil
}
