package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  *syntaxTree
	}{
		{
			`<p>Hello, @name!</p>
@code {
	name := "world"
}
`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "<p>", typ: literalHTML, pos: span{start: 0, end: 3}},
					&nodeLiteral{str: "Hello, ", typ: literalHTML, pos: span{start: 3, end: 10}},
					&nodeGoStrExpr{expr: "name", pos: span{start: 11, end: 15}},
					&nodeLiteral{str: "!", typ: literalHTML, pos: span{start: 15, end: 16}},
					&nodeLiteral{str: "</p>", typ: literalHTML, pos: span{start: 16, end: 20}},
					&nodeLiteral{str: "\n", typ: literalHTML, pos: span{start: 20, end: 21}},
					&nodeGoCode{code: "name := \"world\"\n", pos: span{start: 28, end: 44}},
					&nodeLiteral{str: "\n", typ: literalHTML, pos: span{start: 47, end: 48}},
				}},
		},
		{
			`@if name != "" {
	<h1>Hello, @name!</h1>
} else {
	<h1>Hello, world!</h1>
}`,
			&syntaxTree{
				nodes: []node{
					&nodeIf{
						cond: &nodeGoStrExpr{
							expr: "name != \"\"",
							pos:  span{start: 4, end: 14},
						},
						then: &nodeStmtBlock{
							nodes: []node{
								&nodeLiteral{str: "<h1>", typ: literalHTML, pos: span{start: 18, end: 22}},
								&nodeLiteral{str: "Hello, ", typ: literalHTML, pos: span{start: 22, end: 29}},
								&nodeGoStrExpr{expr: "name", pos: span{start: 30, end: 34}},
								&nodeLiteral{str: "!", typ: literalHTML, pos: span{start: 34, end: 35}},
								&nodeLiteral{str: "</h1>", typ: literalHTML, pos: span{start: 35, end: 40}},
							},
						},
						alt: &nodeStmtBlock{
							nodes: []node{
								&nodeLiteral{str: "<h1>", typ: literalHTML, pos: span{start: 51, end: 55}},
								&nodeLiteral{str: "Hello, world!", typ: literalHTML, pos: span{start: 55, end: 68}},
								&nodeLiteral{str: "</h1>", typ: literalHTML, pos: span{start: 68, end: 73}},
							},
						},
					},
				},
			},
		},
		{
			`@code {
	type product struct {
		name string
		price float32
	}

	products := []product{{name: "Widget", price: 9.49}}
}`,
			&syntaxTree{
				nodes: []node{
					&nodeGoCode{
						code: `type product struct {
		name string
		price float32
	}

	products := []product{{name: "Widget", price: 9.49}}
`,
						pos: span{start: 7, end: 117},
					},
				},
			},
		},
	}
	opts := cmp.AllowUnexported(unexported...)
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, err := parse(test.input)
			if err != nil {
				t.Fatalf("unexpected error parsing input: %v", err)
			}
			if diff := cmp.Diff(test.want, got, opts); diff != "" {
				t.Errorf("expected parse diff (-want +got):\n%s", diff)
			}
		})
	}
}

var unexported = []any{
	nodeGoCode{},
	nodeGoStrExpr{},
	nodeIf{},
	nodeLiteral{},
	nodeStmtBlock{},
	span{},
	syntaxTree{},
}
