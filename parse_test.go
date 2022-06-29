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
					&nodeLiteral{str: "<p>", pos: span{start: 0, end: 3}},
					&nodeLiteral{str: "Hello, ", pos: span{start: 3, end: 10}},
					&nodeGoStrExpr{expr: "name", pos: span{start: 11, end: 15}},
					&nodeLiteral{str: "!", pos: span{start: 15, end: 16}},
					&nodeLiteral{str: "</p>", pos: span{start: 16, end: 20}},
					&nodeLiteral{str: "\n", pos: span{start: 20, end: 21}},
					&nodeGoCode{code: "name := \"world\"\n", pos: span{start: 28, end: 44}},
					&nodeLiteral{str: "\n", pos: span{start: 47, end: 48}},
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
								&nodeElement{tag: tag{name: "h1"}, pos: span{start: 18, end: 22}, children: []node{
									&nodeLiteral{str: "Hello, ", pos: span{start: 22, end: 29}},
									&nodeGoStrExpr{expr: "name", pos: span{start: 30, end: 34}},
									&nodeLiteral{str: "!", pos: span{start: 34, end: 35}},
								}},
							},
						},
						alt: &nodeStmtBlock{
							nodes: []node{
								&nodeElement{tag: tag{name: "h1"}, pos: span{start: 51, end: 55}, children: []node{
									&nodeLiteral{str: "Hello, world!", pos: span{start: 55, end: 68}},
								}},
							},
						},
					},
				},
			},
		},
		{
			`@if name == "" {
    <div>
        <h1>Hello, world!</h1>
    </div>
} else {
    <div>
        <h1>Hello, @name!</h1>
    </div>
}`,
			&syntaxTree{
				nodes: []node{
					&nodeIf{
						cond: &nodeGoStrExpr{
							expr: "name == \"\"",
							pos:  span{start: 4, end: 14},
						},
						then: &nodeStmtBlock{
							nodes: []node{
								&nodeElement{tag: tag{name: "div"}, pos: span{start: 21, end: 26}, children: []node{
									&nodeLiteral{str: "\n        ", pos: span{start: 26, end: 35}},
									&nodeElement{tag: tag{name: "h1"}, pos: span{start: 35, end: 39}, children: []node{
										&nodeLiteral{str: "Hello, world!", pos: span{start: 39, end: 52}},
									},
									},
									&nodeLiteral{str: "\n    ", pos: span{start: 57, end: 62}},
								},
								},
							},
						},
						alt: &nodeStmtBlock{
							nodes: []node{
								&nodeElement{tag: tag{name: "div"}, pos: span{start: 82, end: 87}, children: []node{
									&nodeLiteral{str: "\n        ", pos: span{start: 87, end: 96}},
									&nodeElement{tag: tag{name: "h1"}, pos: span{start: 96, end: 100}, children: []node{
										&nodeLiteral{str: "Hello, ", pos: span{start: 100, end: 107}},
										&nodeGoStrExpr{expr: "name", pos: span{start: 108, end: 112}},
										&nodeLiteral{str: "!", pos: span{start: 112, end: 113}},
									},
									},
									&nodeLiteral{str: "\n    ", pos: span{start: 118, end: 123}},
								},
								},
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
	nodeElement{},
	nodeGoCode{},
	nodeGoStrExpr{},
	nodeIf{},
	nodeLiteral{},
	nodeStmtBlock{},
	span{},
	syntaxTree{},
	tag{},
}
