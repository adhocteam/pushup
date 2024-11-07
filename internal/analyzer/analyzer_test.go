package analyzer

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/adhocteam/pushup/internal/parser"
	"github.com/adhocteam/pushup/internal/up"
)

func TestAnalyzeParam(t *testing.T) {
	tests := []struct {
		file     up.File
		hasError bool
	}{
		{
			file: up.File{
				Kind:    up.Page,
				Content: []byte("^param name string"),
			},
			hasError: true,
		},
		{
			file: up.File{
				Kind:    up.Component,
				Content: []byte("^param name string"),
			},
			hasError: false,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			parser := parser.New()
			doc, err := parser.Parse(test.file.Content)
			if err != nil {
				t.Errorf("parsing content: %v", err)
			}

			unit := &up.CompileUnit{File: &test.file}
			err = analyze(doc, unit)
			if test.hasError && err == nil {
				t.Errorf("expected error, got nil")
			} else if !test.hasError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestExtractComponentName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			"<components.Greeting ",
			"components.Greeting",
		},
		{
			"<components.Greeting/>",
			"components.Greeting",
		},
		{
			"<components.Greeting />",
			"components.Greeting",
		},
		{
			"<components.Greeting>",
			"components.Greeting",
		},
		{
			"<components.Greeting name=\"world\"| />",
			"components.Greeting",
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			got, err := extractComponentName(test.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.want != got {
				t.Errorf("want: %q got: %q", test.want, got)
			}
		})
	}
}

func TestRouteForPage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{
			path:     filepath.Join("pages", "about.up"),
			expected: "/about",
		},
		{
			path:     filepath.Join("pages", "index.up"),
			expected: "/{$}",
		},
		{
			path:     filepath.Join("pages", "blog", "post.up"),
			expected: "/blog/post",
		},
		{
			path:     filepath.Join("pages", "blog", "index.up"),
			expected: "/blog/{$}",
		},
		{
			path:     filepath.Join("pages", "blog", "2024", "posts", "hello.up"),
			expected: "/blog/2024/posts/hello",
		},
		{
			path:     filepath.Join("pages", "users", "id__param", "profile.up"),
			expected: "/users/{id}/profile",
		},
		{
			path:     filepath.Join("pages", "org__param", "repo__param", "issues", "id__param.up"),
			expected: "/{org}/{repo}/issues/{id}",
		},
		{
			path:     filepath.Join("pages", "id__param.up"),
			expected: "/{id}",
		},
		{
			path:     filepath.Join("pages", "users", "id__param", "index.up"),
			expected: "/users/{id}/{$}",
		},
		{
			path:     filepath.Join("pages", "api", "v1", "users", "id__param", "posts", "post__param.up"),
			expected: "/api/v1/users/{id}/posts/{post}",
		},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			// Convert path separators for Windows compatibility
			if runtime.GOOS == "windows" {
				tt.path = filepath.FromSlash(tt.path)
			}

			got := routeForPage(tt.path)
			if got != tt.expected {
				t.Errorf("routeForPage(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}
