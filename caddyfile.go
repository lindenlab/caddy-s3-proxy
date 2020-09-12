package caddys3proxy

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("s3proxy", parseCaddyfile)
}

// parseCaddyfile parses the s3proxy directive. It enables the proxying
// requests to S3 and configures it with this syntax:
//
//    s3proxy {
//	      region <aws region>
//	      bucket <s3 bucket name>
//	      index  <files...>
//	      endpoint: <alternative endpoint>
//    }
//
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var b S3Proxy

	h.NextArg() // skip block beginning: "s3proxy"
	for h.NextBlock(0) {
		var err error
		switch h.Val() {
		case "endpoint":
			h.Args(&b.Endpoint)
		case "region":
			h.Args(&b.Region)
		case "bucket":
			h.Args(&b.Bucket)
			if b.Bucket == "" {
				return nil, h.Err("bucket can not be empty")
			}
		case "index":
			b.IndexNames = h.RemainingArgs()
			if len(b.IndexNames) == 0 {
				return nil, h.ArgErr()
			}
		default:
			err = h.Errf("%s not a valid s3proxy option", h.Val())
		}
		if err != nil {
			return nil, h.Errf("Error parsing %s: %s", h.Val(), err)
		}
	}

	return b, nil
}
