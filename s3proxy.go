package caddys3proxy

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func init() {
	caddy.RegisterModule(S3Proxy{})
	httpcaddyfile.RegisterHandlerDirective("s3proxy", parseCaddyfile)
}

// FileServer implements a static file server responder for Caddy.
type S3Proxy struct {
	// The prefix to prepend to paths when looking for objects in S3
	Prefix string `json:"prefix,omitempty"`

	// The AWS region the bucket is hosted in
	Region string `json:"region,omitempty"`

	// The name of the S3 bucket
	Bucket string `json:"bucket,omitempty"`

	// Use non-standard endpoint for S3
	Endpoint string `json:"endpoint,omitempty"`

	// A list of files or folders to hide; the file server will pretend as if
	// they don't exist. Accepts globular patterns like "*.hidden" or "/foo/*/bar".
	Hide []string `json:"hide,omitempty"`

	// The names of files to try as index files if a folder is requested.
	IndexNames []string `json:"index_names,omitempty"`

	// Enables file listings if a directory was requested and no index
	// file is present.
	// Browse *Browse `json:"browse,omitempty"`

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

func (b *S3Proxy) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.NextArg() // skip block beginning: "s3browser"

	for d.NextBlock(0) {
		var err error
		switch d.Val() {
		//case "site_name":
		//err = parseStringArg(d, &b.SiteName)
		case "endpoint":
			err = parseStringArg(d, &b.Endpoint)
		case "region":
			err = parseStringArg(d, &b.Region)
		//case "key":
		//err = parseStringArg(d, &b.Key)
		//case "secret":
		//err = parseStringArg(d, &b.Secret)
		case "bucket":
			err = parseStringArg(d, &b.Bucket)
		//case "secure":
		//err = parseBoolArg(d, &b.Secure)
		//case "refresh_interval":
		//err = parseDurationArg(d, &b.RefreshInterval)
		//case "refresh_api_secret":
		//err = parseStringArg(d, &b.RefreshAPISecret)
		//case "debug":
		//err = parseBoolArg(d, &b.Debug)
		//case "signed_url_redirect":
		//err = parseBoolArg(d, &b.SignedURLRedirect)
		default:
			err = d.Errf("not a valid s3browser option")
		}
		if err != nil {
			return d.Errf("Error parsing %s: %s", d.Val(), err)
		}
	}

	return nil
}

func (b *S3Proxy) Provision(ctx caddy.Context) (err error) {
	b.log = ctx.Logger(b)

	b.log.Debug("Initializing S3 Proxy")

	var config aws.Config
	if b.Region == "" {
		return errors.New("Region is required to be set")
	}
	config.Region = aws.String(b.Region)
	if b.Endpoint != "" {
		config.Endpoint = aws.String(b.Endpoint)
	}

	sess, err := session.NewSession(&config)
	if err != nil {
		return err
	}

	// Create S3 service client
	b.client = s3.New(sess)
	return nil
}

func (b S3Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// TODO: Handle path manipulation (Root, Prefix, HiddenFiles, etc.)
	fullPath := r.URL.Path
	if fullPath == "" {
		fullPath = "/"
	}

	// TODO: what to do amount weird method types
	// TODO: How to determine if a "dir"
	// TODO: render a "dir" with a template and get a list of objects with stats

	// Hey for now - just serve a freakin path
	oi := s3.GetObjectInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(fullPath),
	}
	obj, err := b.client.GetObject(&oi)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
			case s3.ErrCodeNoSuchKey:
				// 404
				return caddyhttp.Error(http.StatusNotFound, nil)
			default:
				// return 403 maybe?  Why else would it fail?
				return caddyhttp.Error(http.StatusForbidden, err)
			}
		} else {
			return caddyhttp.Error(http.StatusInternalServerError, err)
		}
	}

	w.Header().Set("Content-Type", aws.StringValue(obj.ContentType))
	w.Header().Set("Content-Length", strconv.FormatInt(aws.Int64Value(obj.ContentLength), 10))
	if _, err := io.Copy(w, obj.Body); err != nil {
		return err
	}

	return nil
}
