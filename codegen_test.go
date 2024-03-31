package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCodeGenFromNode(t *testing.T) {
	tests := []struct {
		node Node
		want string
	}{
		{
			node: &NodeElement{
				Tag: Tag{Name: "div", Attrs: []*Attr{{Name: StringPos{Text: "id"}, Value: StringPos{Text: "foo"}}}},
				StartTagNodes: []Node{
					&NodeLiteral{Text: "<div "},
					&NodeLiteral{Text: "id=\""},
					&NodeLiteral{Text: "foo"},
					&NodeLiteral{Text: "\">"},
				},
				Children: []Node{&NodeLiteral{Text: "bar"}},
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
			page, err := newPageFromTree(&SyntaxTree{Nodes: []Node{test.node}})
			if err != nil {
				t.Fatalf("new page from tree: %v", err)
			}
			cparams := &compileParams{}
			g := newPageCodeGen(page, "", cparams)
			g.lineDirectivesEnabled = false
			g.genNode(test.node)
			got := g.body.String()
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
			"pages/index.up",
			"/",
		},
		{
			"pages/about.up",
			"/about",
		},
		{
			"pages/x/sub.up",
			"/x/sub",
		},
		{
			"pages/x/name__param.up",
			"/x/:name",
		},
		{
			"pages/projectId__param/productId__param",
			"/:projectId/:productId",
		},
		{
			"pages/blah/index.up",
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
		{projectFile{path: "a/b/d.up"}, upFileComponent, "D"},
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
