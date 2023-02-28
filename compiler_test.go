package main

import "testing"

func TestCompiledOutputPath(t *testing.T) {
	tests := []struct {
		pfile    projectFile
		want     string
		strategy upFileType
	}{
		{
			projectFile{path: "pages/index.up"},
			"pages/index.up.go",
			upFilePage,
		},
		{
			projectFile{path: "pages/about.up"},
			"pages/about.up.go",
			upFilePage,
		},
		{
			projectFile{path: "pages/x/sub.up"},
			"pages/x/sub.up.go",
			upFilePage,
		},
		{
			projectFile{path: "testdata/foo.up"},
			"testdata/foo.up.go",
			upFilePage,
		},
		{
			projectFile{path: "pages/foo__param.up"},
			"pages/foo__param.up.go",
			upFilePage,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if got := compiledOutputPath(test.pfile, test.strategy); test.want != got {
				t.Errorf("want %q, got %q", test.want, got)
			}
		})
	}
}
