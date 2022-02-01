package caddys3proxy

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

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
				Prefix:    aws.String("mypath/"),
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
				Prefix:    aws.String("mypath/"),
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
				Prefix:            aws.String("mypath/"),
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

func TestMakePageObj(t *testing.T) {
	p := S3Proxy{}
	listOutput := s3.ListObjectsV2Output{
		KeyCount:              aws.Int64(20),
		NextContinuationToken: aws.String("next_token"),
		MaxKeys:               aws.Int64(20),
		CommonPrefixes: []*s3.CommonPrefix{
			&s3.CommonPrefix{
				Prefix: aws.String("/mydir"),
			},
			&s3.CommonPrefix{
				Prefix: aws.String("/otherdir"),
			},
		},
		Contents: []*s3.Object{
			&s3.Object{
				Key:          aws.String("/path/to/myobj"),
				Size:         aws.Int64(1024),
				LastModified: aws.Time(time.Date(1845, time.November, 10, 23, 0, 0, 0, time.UTC)),
			},
		},
	}

	result := p.MakePageObj(&listOutput)
	expected := PageObj{
		Count:    20,
		MoreLink: "?max=20&next=next_token",
		Items: []Item{
			Item{
				Url:   "./mydir/",
				IsDir: true,
				Name:  "mydir",
			},
			Item{
				Url:   "./otherdir/",
				IsDir: true,
				Name:  "otherdir",
			},
			Item{
				Url:          "./myobj",
				Key:          "/path/to/myobj",
				IsDir:        false,
				Name:         "myobj",
				Size:         "1.0 kB",
				LastModified: "a long while ago",
			},
		},
	}

	if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected obj %v, got %v.", expected, result)
	}
}
