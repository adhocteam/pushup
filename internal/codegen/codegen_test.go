package codegen

import (
	"strings"
	"testing"

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/source"
	"github.com/adhocteam/pushup/internal/up"
)

func TestContentTypeHeader(t *testing.T) {
	gen := New()
	unit := &up.CompileUnit{
		Package:  "test",
		TypeName: "TestPage",
		File: &up.File{
			Kind: up.Page,
		},
	}

	code, err := gen.Generate(unit)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify correct Content-Type header is set
	expected := `w.Header().Set("Content-Type", "text/html; charset=utf-8")`
	if !strings.Contains(string(code), expected) {
		t.Errorf("Generated code missing expected Content-Type header.\nWant: %s\nGot: %s",
			expected, string(code))
	}
}

func TestElementAttributeGeneration(t *testing.T) {
	tests := []struct {
		name     string
		element  *ast.NodeElement
		expected string
	}{
		{
			name: "basic attributes",
			element: &ast.NodeElement{
				Tag: ast.Tag{
					Name: "div",
					Attrs: []*ast.Attr{
						{
							NameNodes: []ast.Node{
								&ast.NodeLiteral{Text: "class"},
							},
							ValueNodes: []ast.Node{
								&ast.NodeLiteral{Text: "test"},
							},
						},
					},
				}},
			expected: `<div class="test"></div>`,
		},
		{
			name: "multiple attributes",
			element: &ast.NodeElement{
				Tag: ast.Tag{
					Name: "div",
					Attrs: []*ast.Attr{
						{
							NameNodes: []ast.Node{
								&ast.NodeLiteral{Text: "class"},
							},
							ValueNodes: []ast.Node{
								&ast.NodeLiteral{Text: "test"},
							},
						},
						{
							NameNodes: []ast.Node{
								&ast.NodeLiteral{Text: "itemprop"},
							},
							ValueNodes: []ast.Node{},
						},
					},
				},
			},
			expected: `<div class="test" itemprop></div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := &outputCollector{}
			gen := New()
			gen.unit = up.NewCompileUnit(&up.Project{}, &up.File{})

			gen.gatherOutputOps(tt.element, collector)
			ops := collector.optimize()

			// Combine static content
			var output string
			for _, op := range ops {
				if op.kind == opStatic {
					output += op.content
				}
			}

			if output != tt.expected {
				t.Errorf("Element generation failed\nwant: %s\ngot: %s",
					tt.expected, output)
			}
		})
	}
}

func TestGatherOutputOpsForGoStrExpr(t *testing.T) {
	gen := New()
	collector := &outputCollector{}

	// Test direct expression output without fmt.Sprint wrapping
	expr := &ast.NodeGoStrExpr{
		Expr: "myVar",
		Span: source.Span{Start: 0, End: 5},
	}

	gen.gatherOutputOps(expr, collector)
	ops := collector.optimize()

	if len(ops) != 1 {
		t.Fatalf("Expected 1 output op, got %d", len(ops))
	}

	op := ops[0]
	if op.kind != opDynamic {
		t.Errorf("Expected opDynamic, got %v", op.kind)
	}

	// Verify the expression is output directly without fmt.Sprint
	if op.content != "myVar" {
		t.Errorf("Expected direct expression 'myVar', got '%s'", op.content)
	}
}
