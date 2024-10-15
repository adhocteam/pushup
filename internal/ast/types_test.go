package ast

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"strings"
	"testing"
	"text/template"
)

func TestCodeGenAST(t *testing.T) {
	type fieldspec struct {
		Name string
		Type string
	}

	type nodespec struct {
		Name     string
		Fields   []fieldspec
		SpanExpr string
	}

	nodespecs := []nodespec{
		{"NodeLiteral", []fieldspec{{"Text", "string"}, {"Span", "source.Span"}}, "n.Span"},
		{"NodeGoStrExpr", []fieldspec{{"Expr", "string"}, {"Span", "source.Span"}}, "n.Span"},
		{"NodeGoCode", []fieldspec{{"Context", "GoCodeContext"}, {"Code", "string"}, {"Span", "source.Span"}}, "n.Span"},
		{"NodeIf", []fieldspec{{"Cond", "*NodeGoStrExpr"}, {"Then", "*NodeBlock"}, {"Alt", "Node"}}, "n.Cond.Pos()"},
		{"NodeFor", []fieldspec{{"Clause", "*NodeGoCode"}, {"Block", "*NodeBlock"}}, "n.Clause.Pos()"},
		{"NodePartial", []fieldspec{{"Name", "string"}, {"Span", "source.Span"}, {"Block", "*NodeBlock"}}, "n.Span"},
		{"NodeBlock", []fieldspec{{"Nodes", "[]Node"}}, "n.Nodes[0].Pos()"},
		{"NodeElement", []fieldspec{{"Tag", "element.Tag"}, {"StartTagNodes", "[]Node"}, {"Children", "[]Node"}, {"Span", "source.Span"}}, "n.Span"},
		{"NodeImport", []fieldspec{{"Decl", "ImportDecl"}, {"Span", "source.Span"}}, "n.Span"},
		// TODO NodeList
	}

	beginMarker := `// BEGIN GENERATED CODE NODE DEFINITIONS -- DO NOT EDIT`
	endMarker := `// END GENERATED CODE NODE DEFINITIONS -- DO NOT EDIT`

	originalText, err := os.ReadFile("types.go")
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer

	before, after, found := strings.Cut(string(originalText), beginMarker)
	if !found {
		t.Fatalf("could not find %q in types.go", beginMarker)
	}

	_, after, found = strings.Cut(after, endMarker)
	if !found {
		t.Fatalf("could not find %q in types.go", endMarker)
	}

	buf.WriteString(before)
	buf.WriteString(beginMarker)

	for _, ns := range nodespecs {
		nodetmpl := `
type {{.Name}} struct {
{{range .Fields}}	{{.Name}} {{.Type}}
{{end}}
}

func (n {{.Name}}) Pos() source.Span {
	return {{.SpanExpr}}
}

func (n {{.Name}}) MarshalJSON() ([]byte, error) {
	type t {{.Name}}

	return json.Marshal(struct {
		Type string
		Node t
	}{
		Type: "{{.Name}}",
		Node: t{
{{range .Fields}}		{{.Name}}: n.{{.Name}},
{{end}}
		},
	})
}

func (n *{{.Name}}) UnmarshalJSON(data []byte) error {
	type raw struct {
{{range .Fields}}		{{.Name}} {{if isNodeType .Type}}json.RawMessage{{else if eq .Type "[]Node"}}[]json.RawMessage{{else}}{{.Type}}{{end}}
{{end}}
	}
	var t raw

	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}

{{range .Fields}}
{{if isNodeType .Type}}
	{
		var wrapped NodeWrapper
		if err := json.Unmarshal(t.{{.Name}}, &wrapped); err != nil {
			return err
		}
		n.{{.Name}} = wrapped.Node{{if eq .Type "Node"}}{{else}}.({{.Type}}){{end}}
	}
{{else if eq .Type "[]Node"}}
	for _, raw := range t.{{.Name}} {
		var wrapped NodeWrapper
		if err := json.Unmarshal(raw, &wrapped); err != nil {
			return err
		}
		n.{{.Name}} = append(n.{{.Name}}, wrapped.Node)
	}
{{else}}
	n.{{.Name}} = t.{{.Name}}
{{end}}
{{end}}

	return nil
}

var _ Node = (*{{.Name}})(nil)
`
		tmpl := template.Must(template.New("node").Funcs(map[string]any{
			"isNodeType": func(t string) bool {
				if t == "Node" {
					return true
				}
				for _, ns := range nodespecs {
					if ns.Name == t || (t[0] == '*' && ns.Name == t[1:]) {
						return true
					}
				}
				return false
			},
		}).Parse(nodetmpl))
		if err := tmpl.Execute(&buf, ns); err != nil {
			t.Fatal(err)
		}
	}

	unmarshaltmpl := `
func (nw *NodeWrapper) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var typeMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &typeMap); err != nil {
		return err
	}

	var typ string
	if err := json.Unmarshal(typeMap["Type"], &typ); err != nil {
		return err
	}

	var err error
	switch typ {
	{{range .}}
		case "{{.Name}}":
			var node {{.Name}}
			err = json.Unmarshal(typeMap["Node"], &node)
			nw.Node = &node
	{{end}}
	default:
		return fmt.Errorf("unknown node type: %q", typ)
	}

	return err
}
`
	tmpl := template.Must(template.New("unmarshal").Parse(unmarshaltmpl))
	if err := tmpl.Execute(&buf, nodespecs); err != nil {
		t.Fatal(err)
	}

	buf.WriteString(endMarker)
	buf.WriteString(after)

	fmt.Println(string(buf.Bytes()))
	src, err := format.Source(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(originalText, src) {
		// Write out the generated code to a temp file then atomically rename it
		// to the target
		if err := os.WriteFile("types.go.tmp", src, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename("types.go.tmp", "ast.go"); err != nil {
			t.Fatal(err)
		}
		t.Fatal("generated code differs from original code")
	}
}
