package caddys3proxy

import (
	"reflect"
	"testing"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

type testCase struct {
	desc      string
	input     string
	shouldErr bool
	errString string
	obj       S3Proxy
}

func TestParseCaddyfile(t *testing.T) {
	testCases := []testCase{
		{
			desc: "bad sub directive",
			input: `s3proxy {
				foo
			}`,
			shouldErr: true,
			errString: "foo not a valid s3proxy option, at Testfile:2",
		},
		{
			desc: "bucket bad # args",
			input: `s3proxy {
			bucket
			}`,
			shouldErr: true,
			errString: "wrong argument count or unexpected line ending after 'bucket', at Testfile:2",
		},
		{
			desc: "bucket empty string",
			input: `s3proxy {
				bucket ""
			}`,
			shouldErr: true,
			errString: "bucket must be set and not empty, at Testfile:2",
		},
		{
			desc: "bucket missing",
			input: `s3proxy {
				region foo
			}`,
			shouldErr: true,
			errString: "bucket must be set and not empty, at Testfile:3",
		},
		{
			desc: "endpoint bad # args",
			input: `s3proxy {
				endpoint
			}`,
			shouldErr: true,
			errString: "wrong argument count or unexpected line ending after 'endpoint', at Testfile:2",
		},
		{
			desc: "region bad # args",
			input: `s3proxy {
				region one two
			}`,
			shouldErr: true,
			errString: "wrong argument count or unexpected line ending after 'one', at Testfile:2",
		},
		{
			desc: "root bad # args",
			input: `s3proxy {
				root one two
			}`,
			shouldErr: true,
			errString: "wrong argument count or unexpected line ending after 'one', at Testfile:2",
		},
		{
			desc: "errors on invalid HTTP status for errors",
			input: `s3proxy {
				bucket mybucket
				errors invalid "path/to/404.html"
			}`,
			shouldErr: true,
			errString: "'invalid' is not a valid HTTP status code, at Testfile:3",
		},
		{
			desc: "errors on too many arguments for errors",
			input: `s3proxy {
				bucket mybucket
				errors 403 "path/to/404.html" "what's this?"
			}`,
			shouldErr: true,
			errString: "wrong argument count or unexpected line ending after 'what's this?', at Testfile:3",
		},
		{
			desc: "endpoint gets set",
			input: `s3proxy {
				bucket mybucket
				endpoint myvalue
				region myregion
			}`,
			shouldErr: false,
			obj: S3Proxy{
				Bucket:   "mybucket",
				Endpoint: "myvalue",
				Region:   "myregion",
			},
		},
		{
			desc: "enable pu",
			input: `s3proxy {
				bucket mybucket
				enable_put
			}`,
			shouldErr: false,
			obj: S3Proxy{
				Bucket:    "mybucket",
				EnablePut: true,
			},
		},
		{
			desc: "enable delete",
			input: `s3proxy {
				bucket mybucket
				enable_delete
			}`,
			shouldErr: false,
			obj: S3Proxy{
				Bucket:       "mybucket",
				EnableDelete: true,
			},
		},
		{
			desc: "enable error pages",
			input: `s3proxy {
				bucket mybucket
				errors 404 "path/to/404.html"
				errors 403 "path/to/403.html"
				errors "path/to/default_error.html"
			}`,
			shouldErr: false,
			obj: S3Proxy{
				Bucket: "mybucket",
				ErrorPages: map[int]string{
					403: "path/to/403.html",
					404: "path/to/404.html",
				},
				DefaultErrorPage: "path/to/default_error.html",
			},
		},
		{
			desc: "hide files",
			input: `s3proxy {
				bucket mybucket
				hide foo.txt _*
			}`,
			shouldErr: false,
			obj: S3Proxy{
				Bucket: "mybucket",
				Hide:   []string{"foo.txt", "_*"},
			},
		},
		{
			desc: "hide files - missing arg",
			input: `s3proxy {
				bucket mybucket
				hide
			}`,
			shouldErr: true,
			errString: "wrong argument count or unexpected line ending after 'hide', at Testfile:3",
		},
		{
			desc: "index test",
			input: `s3proxy {
				bucket mybucket
				index i.htm i.html
			}`,
			shouldErr: false,
			obj: S3Proxy{
				Bucket:     "mybucket",
				IndexNames: []string{"i.htm", "i.html"},
			},
		},
		{
			desc: "index - missing arg",
			input: `s3proxy {
				bucket mybucket
				index
			}`,
			shouldErr: true,
			errString: "wrong argument count or unexpected line ending after 'index', at Testfile:3",
		},
	}

	for _, tc := range testCases {
		d := caddyfile.NewTestDispenser(tc.input)
		prox, err := parseCaddyfileWithDispenser(d)

		if tc.shouldErr {
			if err == nil {
				t.Errorf("Test case '%s' expected an err and did not get one", tc.desc)
			}
			if prox != nil {
				t.Errorf("Test case '%s' expected an nil obj but it was not nil", tc.desc)
			}
			if err.Error() != tc.errString {
				t.Errorf("Test case '%s' expected err '%s' but got '%s'", tc.desc, tc.errString, err.Error())
			}
		} else {
			if err != nil {
				t.Errorf("Test case '%s' unexpected err '%s'", tc.desc, err.Error())
			}
			if prox == nil {
				t.Errorf("Test case '%s' return object was nil", tc.desc)
				continue
			}
			if !reflect.DeepEqual(*prox, tc.obj) {
				t.Errorf("Test case '%s' expected Endpoint of  '%#v' but got '%#v'", tc.desc, tc.obj, prox)
			}
		}
	}
}
