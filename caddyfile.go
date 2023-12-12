package caddys3proxy

import (
	"strconv"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("s3proxy", parseCaddyfile)
}

// parseCaddyfile parses the s3proxy directive. It enables the proxying
// requests to S3 and configures it with this syntax:
//
//	s3proxy [<matcher>] {
//	    root   <path to prefix S3 key with>
//	    region <aws region>
//	    profile <aws profile>
//	    bucket <s3 bucket name>
//	    index  <files...>
//	    hide   <file patterns...>
//	    endpoint <alternative endpoint>
//	    enable_put
//	    enable_delete
//	    force_path_style
//	    use_accelerate
//	    errors [<http code>] [<s3 key to error page>|pass_through]
//	    browse [<template file>]
//	}
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	return parseCaddyfileWithDispenser(h.Dispenser)
}

func parseCaddyfileWithDispenser(h *caddyfile.Dispenser) (*S3Proxy, error) {
	var b S3Proxy

	h.NextArg() // skip block beginning: "s3proxy"
parseLoop:
	for h.NextBlock(0) {
		switch h.Val() {
		case "endpoint":
			if !h.AllArgs(&b.Endpoint) {
				return nil, h.ArgErr()
			}
		case "region":
			if !h.AllArgs(&b.Region) {
				return nil, h.ArgErr()
			}
		case "profile":
			if !h.AllArgs(&b.Profile) {
				return nil, h.ArgErr()
			}
		case "root":
			if !h.AllArgs(&b.Root) {
				return nil, h.ArgErr()
			}
		case "hide":
			b.Hide = h.RemainingArgs()
			if len(b.Hide) == 0 {
				return nil, h.ArgErr()
			}
		case "bucket":
			if !h.AllArgs(&b.Bucket) {
				return nil, h.ArgErr()
			}
			if b.Bucket == "" {
				break parseLoop
			}
		case "index":
			b.IndexNames = h.RemainingArgs()
			if len(b.IndexNames) == 0 {
				return nil, h.ArgErr()
			}
		case "enable_put":
			b.EnablePut = true
		case "enable_delete":
			b.EnableDelete = true
		case "force_path_style":
			b.S3ForcePathStyle = true
		case "use_accelerate":
			b.S3UseAccelerate = true
		case "browse":
			b.EnableBrowse = true
			args := h.RemainingArgs()
			if len(args) == 1 {
				b.BrowseTemplate = args[0]
			}
			if len(args) > 1 {
				return nil, h.ArgErr()
			}
		case "error_page", "errors":
			if b.ErrorPages == nil {
				b.ErrorPages = make(map[int]string)
			}

			args := h.RemainingArgs()
			if len(args) == 1 {
				b.DefaultErrorPage = args[0]
			} else if len(args) == 2 {
				httpStatusStr := args[0]
				s3KeyOrPassThrough := args[1]

				httpStatus, err := strconv.Atoi(httpStatusStr)
				if err != nil {
					return nil, h.Errf("'%s' is not a valid HTTP status code", httpStatusStr)
				}

				b.ErrorPages[httpStatus] = s3KeyOrPassThrough
			} else {
				return nil, h.ArgErr()
			}
		default:
			return nil, h.Errf("%s not a valid s3proxy option", h.Val())
		}
	}
	if b.Bucket == "" {
		return nil, h.Err("bucket must be set and not empty")
	}

	return &b, nil
}
