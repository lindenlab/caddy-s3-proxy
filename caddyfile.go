package caddys3proxy

import (
	"strconv"
	"time"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

// UnmarshalCaddyfile parses the caddfile block for the s3proxy configs
func (b *S3Proxy) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.NextArg() // skip block beginning: "s3proxy"

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
			err = d.Errf("%s not a valid s3proxy option", d.Val())
		}
		if err != nil {
			return d.Errf("Error parsing %s: %s", d.Val(), err)
		}
	}

	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var b S3Proxy
	err := b.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}
	return b, err
}

func parseBoolArg(d *caddyfile.Dispenser, out *bool) error {
	var strVal string
	err := parseStringArg(d, &strVal)
	if err == nil {
		*out, err = strconv.ParseBool(strVal)
	}
	return err
}

func parseDurationArg(d *caddyfile.Dispenser, out *time.Duration) error {
	var strVal string
	err := parseStringArg(d, &strVal)
	if err == nil {
		*out, err = time.ParseDuration(strVal)
	}
	return err
}

func parseStringArg(d *caddyfile.Dispenser, out *string) error {
	if !d.Args(out) {
		return d.ArgErr()
	}
	return nil
}
