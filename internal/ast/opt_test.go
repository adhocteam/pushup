package ast

import (
	"testing"

	"github.com/adhocteam/pushup/internal/source"
	"github.com/google/go-cmp/cmp"
)

func TestOptimize(t *testing.T) {
	tests := []struct {
		name     string
		input    *Document
		expected *Document
	}{
		{
			name: "Single Literal Node",
			input: &Document{
				Nodes: []Node{
					&NodeLiteral{Text: "Hello", Span: source.Span{Start: 0, End: 5}},
				},
			},
			expected: &Document{
				Nodes: []Node{
					&NodeLiteral{Text: "Hello", Span: source.Span{Start: 0, End: 5}},
				},
			},
		},
		{
			name: "Consecutive Literals to Coalesce",
			input: &Document{
				Nodes: []Node{
					&NodeLiteral{Text: "Hello", Span: source.Span{Start: 0, End: 5}},
					&NodeLiteral{Text: " ", Span: source.Span{Start: 5, End: 6}},
					&NodeLiteral{Text: "world", Span: source.Span{Start: 6, End: 11}},
				},
			},
			expected: &Document{
				Nodes: []Node{
					&NodeLiteral{Text: "Hello world", Span: source.Span{Start: 0, End: 11}},
				},
			},
		},
		{
			name: "Mixed Nodes - No Coalescing Between Non-Literals",
			input: &Document{
				Nodes: []Node{
					&NodeLiteral{Text: "Hello", Span: source.Span{Start: 0, End: 5}},
					&NodeGoStrExpr{Expr: "testExpr", Span: source.Span{Start: 5, End: 13}},
					&NodeLiteral{Text: "world", Span: source.Span{Start: 13, End: 18}},
				},
			},
			expected: &Document{
				Nodes: []Node{
					&NodeLiteral{Text: "Hello", Span: source.Span{Start: 0, End: 5}},
					&NodeGoStrExpr{Expr: "testExpr", Span: source.Span{Start: 5, End: 13}},
					&NodeLiteral{Text: "world", Span: source.Span{Start: 13, End: 18}},
				},
			},
		},
		{
			name: "Multiple Groups of Literals Coalesced Separately",
			input: &Document{
				Nodes: []Node{
					&NodeLiteral{Text: "Hello", Span: source.Span{Start: 0, End: 5}},
					&NodeLiteral{Text: ", ", Span: source.Span{Start: 5, End: 7}},
					&NodeLiteral{Text: "world", Span: source.Span{Start: 7, End: 12}},
					&NodeGoStrExpr{Expr: "testExpr", Span: source.Span{Start: 12, End: 20}},
					&NodeLiteral{Text: "More text", Span: source.Span{Start: 20, End: 29}},
					&NodeLiteral{Text: " here", Span: source.Span{Start: 29, End: 34}},
				},
			},
			expected: &Document{
				Nodes: []Node{
					&NodeLiteral{Text: "Hello, world", Span: source.Span{Start: 0, End: 12}},
					&NodeGoStrExpr{Expr: "testExpr", Span: source.Span{Start: 12, End: 20}},
					&NodeLiteral{Text: "More text here", Span: source.Span{Start: 20, End: 34}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Optimize(tt.input)
			if diff := cmp.Diff(tt.expected, result); diff != "" {
				t.Errorf("Optimize() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
