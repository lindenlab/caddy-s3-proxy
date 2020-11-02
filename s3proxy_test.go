package caddys3proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/caddyserver/caddy/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type jTestCase struct {
	root     string
	path     string
	expected string
}

func TestJoinPath(t *testing.T) {
	testCases := []jTestCase{
		jTestCase{
			root:     "",
			path:     "/foo",
			expected: "/foo",
		},
		jTestCase{
			root:     "",
			path:     "/",
			expected: "/",
		},
		jTestCase{
			root:     "/",
			path:     "/",
			expected: "/",
		},
		jTestCase{
			root:     "/",
			path:     "/foo",
			expected: "/foo",
		},
		jTestCase{
			root:     "/cat",
			path:     "/dog",
			expected: "/cat/dog",
		},
		jTestCase{
			root:     "/cat/",
			path:     "/dog",
			expected: "/cat/dog",
		},
		jTestCase{
			root:     "/cat/",
			path:     "/dog/",
			expected: "/cat/dog/",
		},
		jTestCase{
			root:     "",
			path:     "/dog/",
			expected: "/dog/",
		},
	}
	for _, tc := range testCases {
		r := joinPath(tc.root, tc.path)
		if r != tc.expected {
			t.Errorf("When joining '%s' and '%s' we expected '%s' but got '%s'", tc.root, tc.path, tc.expected, r)
		}
	}
}

func TestFileHidden(t *testing.T) {
	for i, tc := range []struct {
		inputHide []string
		inputPath string
		expect    bool
	}{
		{
			inputHide: nil,
			inputPath: "",
			expect:    false,
		},
		{
			inputHide: []string{".gitignore"},
			inputPath: "/.gitignore",
			expect:    true,
		},
		{
			inputHide: []string{".git"},
			inputPath: "/.gitignore",
			expect:    false,
		},
		{
			inputHide: []string{"/.git"},
			inputPath: "/.gitignore",
			expect:    false,
		},
		{
			inputHide: []string{".git"},
			inputPath: "/.git",
			expect:    true,
		},
		{
			inputHide: []string{".git"},
			inputPath: "/.git/foo",
			expect:    true,
		},
		{
			inputHide: []string{".git"},
			inputPath: "/foo/.git/bar",
			expect:    true,
		},
		{
			inputHide: []string{"/prefix"},
			inputPath: "/prefix/foo",
			expect:    true,
		},
		{
			inputHide: []string{"/foo/*/bar"},
			inputPath: "/foo/asdf/bar",
			expect:    true,
		},
		{
			inputHide: []string{"/foo"},
			inputPath: "/foo",
			expect:    true,
		},
		{
			inputHide: []string{"/foo"},
			inputPath: "/foobar",
			expect:    false,
		},
	} {
		// for Windows' sake
		tc.inputPath = filepath.FromSlash(tc.inputPath)
		for i := range tc.inputHide {
			tc.inputHide[i] = filepath.FromSlash(tc.inputHide[i])
		}

		actual := fileHidden(tc.inputPath, tc.inputHide)
		if actual != tc.expect {
			t.Errorf("Test %d: Is %s hidden in %v? Got %t but expected %t",
				i, tc.inputPath, tc.inputHide, actual, tc.expect)
		}
	}
}

func newS3Client(t *testing.T) *s3.S3 {
	endpoint := os.Getenv("AWS_ENDPOINT")
	if endpoint == "" {
		t.Skip("Skipping test because AWS_ENDPOINT environment variable is not set.")
	}

	config := aws.Config{
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         aws.String(endpoint),
	}

	sess, err := session.NewSession(&config)
	if err != nil {
		t.Fatal(err)
	}

	return s3.New(sess)
}

func setupTestBucket(t *testing.T, client *s3.S3) string {
	bucketName := fmt.Sprintf(
		"caddy-s3-proxy-testdata-%d-%d",
		time.Now().UnixNano(),
		rand.Int(),
	)
	testDataDir := "testdata"

	_, err := client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if awsErr, isAwsErr := err.(awserr.Error); isAwsErr {
		if awsErr.Code() == s3.ErrCodeBucketAlreadyExists {
			err = nil
		}
	}
	if err != nil {
		t.Fatal(err)
	}

	if err := filepath.Walk(testDataDir, func(p string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if err != nil {
			return err
		}

		key := strings.TrimPrefix(p, testDataDir)
		contentType := mime.TypeByExtension(filepath.Ext(p))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		file, err := os.Open(p)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := client.PutObject(&s3.PutObjectInput{
			Bucket:      aws.String(bucketName),
			Key:         aws.String(key),
			ContentType: aws.String(contentType),
			Body:        file,
		}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	return bucketName
}

func TestProxy(t *testing.T) {
	client := newS3Client(t)
	bucketName := setupTestBucket(t, client)

	for _, tc := range []struct {
		name                 string
		proxy                S3Proxy
		method               string
		body                 []byte
		headers              http.Header
		path                 string
		expectedCode         int
		expectedHeaders      http.Header
		expectedResponseText string
		expectsEmptyResponse bool
	}{
		{
			name:                 "can get simple JSON object",
			proxy:                S3Proxy{Bucket: bucketName},
			method:               http.MethodGet,
			path:                 "/test.json",
			expectedCode:         http.StatusOK,
			expectedResponseText: `{"foo": "bar"}`,
			expectedHeaders: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			name:                 "hidden file are not served",
			proxy:                S3Proxy{Bucket: bucketName, Hide: []string{"test.json"}},
			method:               http.MethodGet,
			path:                 "/test.json",
			expectedCode:         http.StatusNotFound,
			expectsEmptyResponse: true,
		},
		{
			name:                 "can't post",
			proxy:                S3Proxy{Bucket: bucketName},
			method:               http.MethodPost,
			path:                 "/cannot-post",
			expectedCode:         http.StatusMethodNotAllowed,
			expectsEmptyResponse: true,
		},
		{
			name:                 "can't delete if not allowed",
			proxy:                S3Proxy{Bucket: bucketName},
			method:               http.MethodDelete,
			path:                 "/cannot-delete",
			expectedCode:         http.StatusMethodNotAllowed,
			expectsEmptyResponse: true,
		},
		{
			name:                 "can delete if allowed",
			proxy:                S3Proxy{Bucket: bucketName, EnableDelete: true},
			method:               http.MethodDelete,
			path:                 "/to-delete.json",
			expectedCode:         http.StatusOK,
			expectsEmptyResponse: true,
		},
		{
			name:                 "can't put if not allowed",
			proxy:                S3Proxy{Bucket: bucketName},
			method:               http.MethodPut,
			path:                 "/cannot-put",
			expectedCode:         http.StatusMethodNotAllowed,
			expectsEmptyResponse: true,
		},
		{
			name:                 "can put if allowed",
			proxy:                S3Proxy{Bucket: bucketName, EnablePut: true},
			method:               http.MethodPut,
			path:                 "/can-put",
			body:                 []byte("some content"),
			expectedCode:         http.StatusOK,
			expectsEmptyResponse: true,
		},
		{
			name:                 "serves index.html",
			proxy:                S3Proxy{Bucket: bucketName, IndexNames: []string{"index.html"}},
			method:               http.MethodGet,
			path:                 "/inner/",
			expectedCode:         http.StatusOK,
			expectedResponseText: "my index.html",
		},
		{
			name:                 "cannot browse",
			proxy:                S3Proxy{Bucket: bucketName},
			method:               http.MethodGet,
			path:                 "/inner/",
			expectedCode:         http.StatusForbidden,
			expectsEmptyResponse: true,
		},
		{
			name:         "returns 404 if not found",
			proxy:        S3Proxy{Bucket: bucketName},
			method:       http.MethodGet,
			path:         "/doesnt-exist",
			expectedCode: http.StatusNotFound,
		},
		{
			name: "returns 404 page if 404 error page is set",
			proxy: S3Proxy{
				Bucket:           bucketName,
				ErrorPages:       map[int]string{404: "_404.txt"},
				DefaultErrorPage: "default_error_page.txt",
			},
			method:               http.MethodGet,
			path:                 "/doesnt-exist",
			expectedCode:         http.StatusNotFound,
			expectedResponseText: `this is 404`,
		},
		{
			name: "returns default page if default error page is set",
			proxy: S3Proxy{
				Bucket:           bucketName,
				DefaultErrorPage: "default_error_page.txt",
			},
			method:               http.MethodGet,
			path:                 "/doesnt-exist",
			expectedCode:         http.StatusNotFound,
			expectedResponseText: `this is a default error page`,
		},
		{
			name:   "returns range",
			proxy:  S3Proxy{Bucket: bucketName},
			method: http.MethodGet,
			path:   "/test.json",
			headers: http.Header{
				"Range": []string{"bytes=0-4"},
			},
			expectedCode:         http.StatusOK,
			expectedResponseText: `{"foo`,
		},
		{
			name:   "returns 200 code If-Match",
			proxy:  S3Proxy{Bucket: bucketName},
			method: http.MethodGet,
			path:   "/test.json",
			headers: http.Header{
				"If-Match": []string{`"a38212e01d6f419c9bd303b304a99e9b"`},
			},
			expectedCode:         http.StatusOK,
			expectedResponseText: `{"foo": "bar"}`,
		},
		{
			name:   "returns 412 If-Match",
			proxy:  S3Proxy{Bucket: bucketName},
			method: http.MethodGet,
			path:   "/test.json",
			headers: http.Header{
				"If-Match": []string{`"no good etag"`},
			},
			expectedCode:         http.StatusPreconditionFailed,
			expectsEmptyResponse: true,
		},
		{
			name:   "returns 304 If-None-Match",
			proxy:  S3Proxy{Bucket: bucketName},
			method: http.MethodGet,
			path:   "/test.json",
			headers: http.Header{
				"If-None-Match": []string{`"a38212e01d6f419c9bd303b304a99e9b"`},
			},
			expectedCode:         http.StatusNotModified,
			expectsEmptyResponse: true,
		},
		{
			name:   "returns 200 If-None-Match",
			proxy:  S3Proxy{Bucket: bucketName},
			method: http.MethodGet,
			path:   "/test.json",
			headers: http.Header{
				"If-None-Match": []string{`"no good etag"`},
			},
			expectedCode:         http.StatusOK,
			expectedResponseText: `{"foo": "bar"}`,
		},
		{
			name:   "returns 200 If-Unmodified-Since",
			proxy:  S3Proxy{Bucket: bucketName},
			method: http.MethodGet,
			path:   "/test.json",
			headers: http.Header{
				"If-Unmodified-Since": []string{`Thu, 05 May 2568 07:28:00 GMT`},
			},
			expectedCode:         http.StatusOK,
			expectedResponseText: `{"foo": "bar"}`,
		},
		{
			name:   "returns 412 If-Unmodified-Since",
			proxy:  S3Proxy{Bucket: bucketName},
			method: http.MethodGet,
			path:   "/test.json",
			headers: http.Header{
				"If-Unmodified-Since": []string{`Wed, 21 Oct 2015 07:28:00 GMT`},
			},
			expectedCode:         http.StatusPreconditionFailed,
			expectsEmptyResponse: true,
		},
		{
			name:   "returns 200 If-Modified-Since",
			proxy:  S3Proxy{Bucket: bucketName},
			method: http.MethodGet,
			path:   "/test.json",
			headers: http.Header{
				"If-Modified-Since": []string{`Thu, 05 May 2568 07:28:00 GMT`},
			},
			expectedCode:         http.StatusNotModified,
			expectsEmptyResponse: true,
		},
		{
			name:   "returns 412 If-Modified-Since",
			proxy:  S3Proxy{Bucket: bucketName},
			method: http.MethodGet,
			path:   "/test.json",
			headers: http.Header{
				"If-Modified-Since": []string{`Wed, 21 Oct 2015 07:28:00 GMT`},
			},
			expectedCode:         http.StatusOK,
			expectedResponseText: `{"foo": "bar"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader
			if tc.body != nil {
				body = bytes.NewReader(tc.body)
			}

			req, err := http.NewRequest(tc.method, tc.path, body)
			if err != nil {
				t.Fatal(err)
			}
			repl := caddy.NewReplacer()
			ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
			req = req.WithContext(ctx)
			req.Header = tc.headers

			recorder := httptest.NewRecorder()

			tc.proxy.client = client
			tc.proxy.log = zap.NewExample()

			_ = tc.proxy.ServeHTTP(recorder, req, nil)

			// Check HTTP status code
			if tc.expectedCode != 0 && recorder.Code != tc.expectedCode {
				t.Errorf("Expected code %d, got %d.", tc.expectedCode, recorder.Code)
			}

			// Check response headers
			respHeaders := recorder.Header()
			for k, v := range tc.expectedHeaders {
				if !reflect.DeepEqual(respHeaders.Values(k), v) {
					t.Errorf("Expected headers %v, got %v.", tc.expectedHeaders, respHeaders.Values(k))
				}
			}

			// Check response body
			if tc.expectedResponseText != "" && tc.expectedResponseText != strings.TrimSpace(recorder.Body.String()) {
				t.Errorf(
					"Expected response text %s, got %s.",
					tc.expectedResponseText,
					recorder.Body.String(),
				)
			}

			// Check if response should be empty
			if tc.expectsEmptyResponse && recorder.Body.Len() != 0 {
				t.Errorf("Expected response body to be empty, got %s.", recorder.Body.String())
			}
		})
	}
}
