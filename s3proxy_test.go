package caddys3proxy

import (
	"path/filepath"
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
