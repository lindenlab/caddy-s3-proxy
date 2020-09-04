package caddys3proxy

import (
	"strconv"
	"time"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

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
