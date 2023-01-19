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
		// escaped transition symbol
		{
			`^^foo`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "^", pos: span{start: 0, end: 2}},
					&nodeLiteral{str: "foo", pos: span{start: 2, end: 5}},
				},
			},
		},
		{
			`<a href="^^foo"></a>`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "<a ", pos: span{end: 3}},
					&nodeLiteral{str: "href", pos: span{start: 3, end: 7}},
					&nodeLiteral{str: `="`, pos: span{start: 7, end: 9}},
					&nodeLiteral{str: "^", pos: span{start: 9, end: 10}},
					&nodeLiteral{str: `foo">`, pos: span{start: 11, end: 16}},
					&nodeLiteral{str: "</a>", pos: span{start: 16, end: 20}},
				},
			},
		},
		{
			`<p>Hello, ^name!</p>
^{
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
				},
			},
		},
		{
			`^if name != "" {
	<h1>Hello, ^name!</h1>
} ^else {
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
									},
								},
							},
						},
						alt: &nodeBlock{
							nodes: []node{
								&nodeLiteral{str: "\n\t", pos: span{start: 50, end: 52}},
								&nodeElement{
									tag:           tag{name: "h1"},
									startTagNodes: []node{&nodeLiteral{str: "<h1>", pos: span{start: 52, end: 56}}},
									pos:           span{start: 52, end: 56},
									children: []node{
										&nodeLiteral{str: "Hello, world!", pos: span{start: 56, end: 69}},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			`^if name == "" {
    <div>
        <h1>Hello, world!</h1>
    </div>
} ^else {
    <div>
        <h1>Hello, ^name!</h1>
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
								&nodeLiteral{str: "\n    ", pos: span{start: 78, end: 83}},
								&nodeElement{
									tag:           tag{name: "div"},
									pos:           span{start: 83, end: 88},
									startTagNodes: []node{&nodeLiteral{str: "<div>", pos: span{start: 83, end: 88}}},
									children: []node{
										&nodeLiteral{str: "\n        ", pos: span{start: 88, end: 97}},
										&nodeElement{
											tag:           tag{name: "h1"},
											startTagNodes: []node{&nodeLiteral{str: "<h1>", pos: span{start: 97, end: 101}}},
											pos:           span{start: 97, end: 101},
											children: []node{
												&nodeLiteral{str: "Hello, ", pos: span{start: 101, end: 108}},
												&nodeGoStrExpr{expr: "name", pos: span{start: 109, end: 113}},
												&nodeLiteral{str: "!", pos: span{start: 113, end: 114}},
											},
										},
										&nodeLiteral{str: "\n    ", pos: span{start: 119, end: 124}},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			`^{
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
			`^layout !`,
			&syntaxTree{
				nodes: []node{
					&nodeLayout{name: "!", pos: span{start: 1, end: 9}},
				},
			},
		},
		{
			`^import "time"`,
			&syntaxTree{
				nodes: []node{
					&nodeImport{
						decl: importDecl{
							pkgName: "",
							path:    "\"time\"",
						},
						pos: span{},
					},
				},
			},
		},
		{
			`<a href="^url">x</a>`,
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
			`^if true { <a href="^url"><div data-^foo="bar"></div></a> }`,
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
												value: stringPos{string: "^url", start: pos(9)},
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
														name:  stringPos{string: "data-^foo", start: pos(5)},
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
		{
			`^section foo {<text>bar</text>}`,
			&syntaxTree{
				nodes: []node{
					&nodeSection{
						name: "foo",
						pos:  span{start: 8, end: 12},
						block: &nodeBlock{
							nodes: []node{
								&nodeBlock{nodes: []node{&nodeLiteral{str: "bar", pos: span{start: 20, end: 23}}}},
							},
						},
					},
				},
			},
		},
		{
			// example of expanded implicit/simple expression
			`^foo.bar("asd").baz.biz()`,
			&syntaxTree{
				nodes: []node{
					&nodeGoStrExpr{expr: `foo.bar("asd").baz.biz()`, pos: span{start: 1, end: 25}},
				},
			},
		},
		{
			// example of expanded implicit/simple expression
			`^quux[42]`,
			&syntaxTree{
				nodes: []node{
					&nodeGoStrExpr{expr: `quux[42]`, pos: span{start: 1, end: 9}},
				},
			},
		},
		{
			// space separates the Go ident `name` from the next `(`, ending the
			// expression and preventing it from being interpreted as a function
			// call
			`^p.name ($blah ...`,
			&syntaxTree{
				nodes: []node{
					&nodeGoStrExpr{expr: `p.name`, pos: span{start: 1, end: 7}},
					&nodeLiteral{str: ` ($blah ...`, pos: span{start: 7, end: 18}},
				},
			},
		},
		{
			// expanded implicit/simple expression with argument list in func call
			`^getParam(req, "name")`,
			&syntaxTree{
				nodes: []node{
					&nodeGoStrExpr{expr: `getParam(req, "name")`, pos: span{start: 1, end: 22}},
				},
			},
		},
		{
			`^partial foo {<text>bar</text>}`,
			&syntaxTree{
				nodes: []node{
					&nodePartial{
						name: "foo",
						pos:  span{start: 8, end: 12},
						block: &nodeBlock{
							nodes: []node{
								&nodeBlock{nodes: []node{&nodeLiteral{str: "bar", pos: span{start: 20, end: 23}}}},
							},
						},
					},
				},
			},
		},
		{
			`^partial foo {
				^if true {
					<p></p>
				}
			}`,
			&syntaxTree{
				nodes: []node{
					&nodePartial{
						name: "foo",
						pos:  span{start: 8, end: 12},
						block: &nodeBlock{
							nodes: []node{
								&nodeIf{
									cond: &nodeGoStrExpr{expr: "true", pos: span{23, 27}},
									then: &nodeBlock{
										nodes: []node{
											&nodeLiteral{str: "\n\t\t\t\t\t", pos: span{start: 29, end: 35}},
											&nodeElement{
												tag:           tag{name: "p"},
												startTagNodes: []node{&nodeLiteral{str: "<p>", pos: span{start: 35, end: 38}}},
												pos:           span{35, 38},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			`My name is ^name. What's yours?`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "My name is ", pos: span{end: 11}},
					&nodeGoStrExpr{expr: "name", pos: span{start: 12, end: 16}},
					&nodeLiteral{str: ". What's yours?", pos: span{start: 16, end: 31}},
				},
			},
		},
		{
			`<p>^foo.</p>`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "<p>", pos: span{end: 3}},
					&nodeGoStrExpr{expr: "foo", pos: span{start: 4, end: 7}},
					&nodeLiteral{str: ".", pos: span{start: 7, end: 8}},
					&nodeLiteral{str: "</p>", pos: span{start: 8, end: 12}},
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
	attr{},
	importDecl{},
	nodeBlock{},
	nodeElement{},
	nodeGoCode{},
	nodeGoStrExpr{},
	nodeIf{},
	nodeImport{},
	nodeLayout{},
	nodeLiteral{},
	nodeSection{},
	nodePartial{},
	span{},
	stringPos{},
	syntaxTree{},
	tag{},
}

func TestParseSyntaxErrors(t *testing.T) {
	tests := []struct {
		input string
		// expected error conditions
		lineNo int
		column int
	}{
		{"^if", 1, 4},
		{
			`^if true {
	<illegal />
}`, 2, 13,
		},
		// FIXME(paulsmith): add more syntax errors
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			tree, err := parse(tt.input)
			if tree != nil {
				t.Errorf("expected nil tree, got %v", tree)
			}
			if err == nil {
				t.Errorf("expected parse error, got nil")
			}
			serr, ok := err.(syntaxError)
			if !ok {
				t.Errorf("expected syntax error type, got %T", err)
			}
			if tt.lineNo != serr.lineNo || tt.column != serr.column {
				t.Errorf("line:column: want %d:%d, got %d:%d", tt.lineNo, tt.column, serr.lineNo, serr.column)
			}
		})
	}
}

func FuzzParser(f *testing.F) {
	seeds := []string{
		"",
		"^layout !\n",
		"<h1>Hello, world!</h1>",
		"^{ name := \"world\" }\n<h1>Hello, ^name!</h1>\n",
		"^if true {\n<a href=\"^req.URL.Path\">this page</a>\n}\n",
		"<div>^(3 + 4 * 5)</div>\n",
	}

	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, in []byte) {
		_, err := parse(string(in))
		if err != nil {
			if _, ok := err.(syntaxError); !ok {
				t.Errorf("expected syntax error, got %T %v", err, err)
			}
		}
	})
}

func TestTagString(t *testing.T) {
	tests := []struct {
		tag  tag
		want string
	}{
		{
			tag{name: "h1"},
			"h1",
		},
		{
			tag{name: "div", attrs: []*attr{{name: stringPos{string: "class"}, value: stringPos{string: "banner"}}}},
			"div class=\"banner\"",
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := test.tag.String()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("(-want, +got)\n%s", diff)
			}
		})
	}
}
