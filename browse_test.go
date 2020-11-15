package caddys3proxy

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func TestConstructListObjInput(t *testing.T) {

	type testCase struct {
		name        string
		key         string
		bucket      string
		queryString string
		expected    s3.ListObjectsV2Input
	}

	testCases := []testCase{
		testCase{
			name:   "no query options",
			bucket: "myBucket",
			key:    "/mypath/",
			expected: s3.ListObjectsV2Input{
				Bucket:    aws.String("myBucket"),
				Delimiter: aws.String("/"),
				Prefix:    aws.String("/mypath"),
			},
		},
		testCase{
			name:        "max option",
			bucket:      "myBucket",
			key:         "/mypath/",
			queryString: "?max=20",
			expected: s3.ListObjectsV2Input{
				Bucket:    aws.String("myBucket"),
				Delimiter: aws.String("/"),
				Prefix:    aws.String("/mypath"),
				MaxKeys:   aws.Int64(20),
			},
		},
		testCase{
			name:        "max with next",
			bucket:      "myBucket",
			key:         "/mypath/",
			queryString: "?max=20&next=FOO",
			expected: s3.ListObjectsV2Input{
				Bucket:            aws.String("myBucket"),
				Delimiter:         aws.String("/"),
				Prefix:            aws.String("/mypath"),
				MaxKeys:           aws.Int64(20),
				ContinuationToken: aws.String("FOO"),
			},
		},
	}
	for _, tc := range testCases {
		r := http.Request{}
		u, _ := url.Parse(tc.queryString)
		r.URL = u
		p := S3Proxy{
			Bucket: tc.bucket,
		}
		result := p.ConstructListObjInput(&r, tc.key)
		if !reflect.DeepEqual(tc.expected, result) {
			t.Errorf("Expected obj %v, got %v.", tc.expected, result)
		}
	}
}
