package analyzer

import (
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
