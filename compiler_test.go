package main

import "testing"

func TestCompiledOutputPath(t *testing.T) {
	tests := []struct {
		pfile    projectFile
		want     string
		strategy upFileType
	}{
		{
			projectFile{path: "app/pages/index.up", projectFilesSubdir: "app/pages"},
			"index.up.go",
			upFilePage,
		},
		{
			projectFile{path: "app/pages/about.up", projectFilesSubdir: "app/pages"},
			"about.up.go",
			upFilePage,
		},
		{
			projectFile{path: "app/pages/x/sub.up", projectFilesSubdir: "app/pages"},
			"x__sub.up.go",
			upFilePage,
		},
		{
			projectFile{path: "testdata/foo.up", projectFilesSubdir: ""},
			"testdata__foo.up.go",
			upFilePage,
		},
		{
			projectFile{path: "app/layouts/default.up", projectFilesSubdir: "app/layouts"},
			"default.layout.up.go",
			upFileLayout,
		},
		{
			projectFile{path: "app/pages/$foo.up", projectFilesSubdir: "app/pages"},
			"0x24foo.up.go",
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
