package caddys3proxy

import (
	"testing"
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
