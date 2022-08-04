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
		// escaped '@'
		{
			`@@foo`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "@", pos: span{start: 0, end: 2}},
					&nodeLiteral{str: "foo", pos: span{start: 2, end: 5}},
				},
			},
		},
		{
			`<a href="@@foo"></a>`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "<a ", pos: span{end: 3}},
					&nodeLiteral{str: "href", pos: span{start: 3, end: 7}},
					&nodeLiteral{str: `="`, pos: span{start: 7, end: 9}},
					&nodeLiteral{str: "@", pos: span{start: 9, end: 10}},
					&nodeLiteral{str: `foo">`, pos: span{start: 11, end: 16}},
					&nodeLiteral{str: "</a>", pos: span{start: 16, end: 20}},
				},
			},
		},
		{
			`<p>Hello, @name!</p>
@{
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
					&nodeGoCode{code: "name := \"world\"\n", pos: span{start: 23, end: 39}},
					&nodeLiteral{str: "\n", pos: span{start: 42, end: 43}},
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
						then: &nodeBlock{
							nodes: []node{
								&nodeLiteral{str: "\n\t", pos: span{start: 16, end: 18}},
								&nodeElement{
									tag:           tag{name: "h1"},
									startTagNodes: []node{&nodeLiteral{str: "<h1>", pos: span{start: 18, end: 22}}},
									pos:           span{start: 18, end: 22},
									children: []node{
										&nodeLiteral{str: "Hello, ", pos: span{start: 22, end: 29}},
										&nodeGoStrExpr{expr: "name", pos: span{start: 30, end: 34}},
										&nodeLiteral{str: "!", pos: span{start: 34, end: 35}},
									}},
							},
						},
						alt: &nodeBlock{
							nodes: []node{
								&nodeLiteral{str: "\n\t", pos: span{start: 49, end: 51}},
								&nodeElement{
									tag:           tag{name: "h1"},
									startTagNodes: []node{&nodeLiteral{str: "<h1>", pos: span{start: 51, end: 55}}},
									pos:           span{start: 51, end: 55},
									children: []node{
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
						then: &nodeBlock{
							nodes: []node{
								&nodeLiteral{str: "\n    ", pos: span{start: 16, end: 21}},
								&nodeElement{
									tag:           tag{name: "div"},
									startTagNodes: []node{&nodeLiteral{str: "<div>", pos: span{start: 21, end: 26}}},
									pos:           span{start: 21, end: 26},
									children: []node{
										&nodeLiteral{str: "\n        ", pos: span{start: 26, end: 35}},
										&nodeElement{
											tag:           tag{name: "h1"},
											startTagNodes: []node{&nodeLiteral{str: "<h1>", pos: span{start: 35, end: 39}}},
											pos:           span{start: 35, end: 39},
											children: []node{
												&nodeLiteral{str: "Hello, world!", pos: span{start: 39, end: 52}},
											},
										},
										&nodeLiteral{str: "\n    ", pos: span{start: 57, end: 62}},
									},
								},
							},
						},
						alt: &nodeBlock{
							nodes: []node{
								&nodeLiteral{str: "\n    ", pos: span{start: 77, end: 82}},
								&nodeElement{
									tag:           tag{name: "div"},
									pos:           span{start: 82, end: 87},
									startTagNodes: []node{&nodeLiteral{str: "<div>", pos: span{start: 82, end: 87}}},
									children: []node{
										&nodeLiteral{str: "\n        ", pos: span{start: 87, end: 96}},
										&nodeElement{
											tag:           tag{name: "h1"},
											startTagNodes: []node{&nodeLiteral{str: "<h1>", pos: span{start: 96, end: 100}}},
											pos:           span{start: 96, end: 100},
											children: []node{
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
			`@{
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
						pos: span{start: 2, end: 112},
					},
				},
			},
		},
		{
			`@layout !`,
			&syntaxTree{
				nodes: []node{
					&nodeLayout{name: "!", pos: span{start: 1, end: 9}},
				},
			},
		},
		{
			`@import "time"`,
			&syntaxTree{
				nodes: []node{
					&nodeImport{
						decl: importDecl{
							pkgName: "",
							path:    "\"time\"",
						},
						pos: span{}},
				},
			},
		},
		{
			`<a href="@url">x</a>`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "<a ", pos: span{start: 0, end: 3}},
					&nodeLiteral{str: "href", pos: span{start: 3, end: 7}},
					&nodeLiteral{str: "=\"", pos: span{start: 7, end: 9}},
					&nodeGoStrExpr{expr: "url", pos: span{start: 0, end: 3}}, // FIXME(paulsmith): should be 9 and 13 but this part of the parser is not correct yet
					&nodeLiteral{str: "\">", pos: span{start: 13, end: 15}},
					&nodeLiteral{str: "x", pos: span{start: 15, end: 16}},
					&nodeLiteral{str: "</a>", pos: span{start: 16, end: 20}},
				},
			},
		},
		{
			`@if true { <a href="@url"><div data-@foo="bar"></div></a> }`,
			&syntaxTree{
				nodes: []node{
					&nodeIf{
						cond: &nodeGoStrExpr{expr: "true", pos: span{start: 4, end: 8}},
						then: &nodeBlock{
							nodes: []node{
								&nodeLiteral{str: " ", pos: span{start: 10, end: 11}},
								&nodeElement{
									tag: tag{
										name: "a",
										attrs: []*attr{
											{
												name:  stringPos{string: "href", start: pos(3)},
												value: stringPos{string: "@url", start: pos(9)},
											},
										},
									},
									startTagNodes: []node{
										&nodeLiteral{str: "<a ", pos: span{start: 11, end: 14}},
										&nodeLiteral{str: "href", pos: span{start: 14, end: 18}},
										&nodeLiteral{str: `="`, pos: span{start: 18, end: 20}},
										&nodeGoStrExpr{expr: "url", pos: span{end: 3}},
										&nodeLiteral{str: `">`, pos: span{start: 24, end: 26}},
									},
									children: []node{
										&nodeElement{
											tag: tag{
												name: "div",
												attrs: []*attr{
													{
														name:  stringPos{string: "data-@foo", start: pos(5)},
														value: stringPos{string: "bar", start: pos(16)},
													},
												},
											},
											startTagNodes: []node{
												&nodeLiteral{str: "<div ", pos: span{start: 26, end: 31}},
												&nodeLiteral{str: "data-", pos: span{start: 31, end: 36}},
												&nodeGoStrExpr{expr: "foo", pos: span{start: 0, end: 3}}, // FIXME
												&nodeLiteral{str: "=\"", pos: span{start: 40, end: 42}},
												&nodeLiteral{str: "bar", pos: span{start: 42, end: 45}},
												&nodeLiteral{str: "\">", pos: span{start: 45, end: 47}},
											},
											pos: span{start: 26, end: 47},
										},
									},
									pos: span{start: 11, end: 26},
								},
							},
						},
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
	importDecl{},
	nodeBlock{},
	nodeElement{},
	nodeGoCode{},
	nodeGoStrExpr{},
	nodeIf{},
	nodeImport{},
	nodeLayout{},
	nodeLiteral{},
	span{},
	syntaxTree{},
	tag{},
	attr{},
	stringPos{},
}
