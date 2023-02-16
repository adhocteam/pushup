package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCodeGenFromNode(t *testing.T) {
	tests := []struct {
		node node
		want string
	}{
		{
			node: &nodeElement{
				tag: tag{name: "div", attrs: []*attr{{name: stringPos{string: "id"}, value: stringPos{string: "foo"}}}},
				startTagNodes: []node{
					&nodeLiteral{str: "<div "},
					&nodeLiteral{str: "id=\""},
					&nodeLiteral{str: "foo"},
					&nodeLiteral{str: "\">"},
				},
				children: []node{&nodeLiteral{str: "bar"}},
			},
			want: `io.WriteString(w, "<div ")
io.WriteString(w, "id=\"")
io.WriteString(w, "foo")
io.WriteString(w, "\">")
io.WriteString(w, "bar")
io.WriteString(w, "</div>")
`,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			page, err := newPageFromTree(&syntaxTree{nodes: []node{test.node}})
			if err != nil {
				t.Fatalf("new page from tree: %v", err)
			}
			g := newPageCodeGen(page, projectFile{}, "")
			g.lineDirectivesEnabled = false
			g.genNode(test.node)
			got := g.bodyb.String()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("expected code gen diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRouteForPage(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{
			"index.up",
			"/",
		},
		{
			"about.up",
			"/about",
		},
		{
			"x/sub.up",
			"/x/sub",
		},
		{
			"testdata/foo.up",
			"/testdata/foo",
		},
		{
			"x/name__param.up",
			"/x/:name",
		},
		{
			"projectId__param/productId__param",
			"/:projectId/:productId",
		},
		{
			"blah/index.up",
			"/blah/",
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if got := routeForPage(test.path); test.want != got {
				t.Errorf("want %q, got %q", test.want, got)
			}
		})
	}
}

func TestGeneratedTypename(t *testing.T) {
	tests := []struct {
		pfile    projectFile
		strategy upFileType
		want     string
	}{
		{projectFile{path: "index.up"}, upFilePage, "IndexPage"},
		{projectFile{path: "foo-bar.up"}, upFilePage, "FooBarPage"},
		{projectFile{path: "foo_bar.up"}, upFilePage, "FooBarPage"},
		{projectFile{path: "a/b/c.up"}, upFilePage, "CPage"},
	}

	for _, test := range tests {
		t.Run(test.pfile.path, func(t *testing.T) {
			got := generatedTypename(test.pfile, test.strategy)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("(-want, +got)\n%s", diff)
			}
		})
	}
}
