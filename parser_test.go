package main

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  *SyntaxTree
	}{
		// escaped transition symbol
		{
			`^^foo`,
			&SyntaxTree{
				Nodes: []Node{
					&NodeLiteral{Text: "^", Span: Span{Start: 0, End: 2}},
					&NodeLiteral{Text: "foo", Span: Span{Start: 2, End: 5}},
				},
			},
		},
		{
			`<a href="^^foo"></a>`,
			&SyntaxTree{
				Nodes: []Node{
					&NodeElement{
						Tag: Tag{
							Name: "a",
							Attrs: []*Attr{
								{
									Name:  StringPos{Text: "href", Start: pos(3)},
									Value: StringPos{Text: "^^foo", Start: pos(9)},
								},
							},
						},
						StartTagNodes: []Node{
							&NodeLiteral{Text: "<a ", Span: Span{End: 3}},
							&NodeLiteral{Text: "href", Span: Span{Start: 3, End: 7}},
							&NodeLiteral{Text: `="`, Span: Span{Start: 7, End: 9}},
							&NodeLiteral{Text: "^", Span: Span{Start: 9, End: 10}},
							&NodeLiteral{Text: `foo">`, Span: Span{Start: 11, End: 16}},
						},
						Span: Span{Start: 0, End: 16},
					},
				},
			},
		},
		{
			`<p>Hello, ^name!</p>
^{
	name := "world"
}
`,
			&SyntaxTree{
				Nodes: []Node{
					&NodeElement{
						Tag: Tag{
							Name: "p",
						},
						StartTagNodes: []Node{
							&NodeLiteral{Text: "<p>", Span: Span{End: 3}},
						},
						Children: []Node{
							&NodeLiteral{Text: "Hello, ", Span: Span{Start: 3, End: 10}},
							&NodeGoStrExpr{Expr: "name", Span: Span{Start: 11, End: 15}},
							&NodeLiteral{Text: "!", Span: Span{Start: 15, End: 16}},
						},
						Span: Span{
							Start: 0,
							End:   3,
						},
					},
					&NodeLiteral{Text: "\n", Span: Span{Start: 20, End: 21}},
					&NodeGoCode{Code: "name := \"world\"\n", Span: Span{Start: 23, End: 39}},
					&NodeLiteral{Text: "\n", Span: Span{Start: 42, End: 43}},
				},
			},
		},
		{
			`^if name != "" {
	<h1>Hello, ^name!</h1>
} ^else {
	<h1>Hello, world!</h1>
}`,
			&SyntaxTree{
				Nodes: []Node{
					&NodeIf{
						Cond: &NodeGoStrExpr{
							Expr: "name != \"\"",
							Span: Span{Start: 4, End: 14},
						},
						Then: &NodeBlock{
							Nodes: []Node{
								&NodeLiteral{Text: "\n\t", Span: Span{Start: 16, End: 18}},
								&NodeElement{
									Tag:           Tag{Name: "h1"},
									StartTagNodes: []Node{&NodeLiteral{Text: "<h1>", Span: Span{Start: 18, End: 22}}},
									Span:          Span{Start: 18, End: 22},
									Children: []Node{
										&NodeLiteral{Text: "Hello, ", Span: Span{Start: 22, End: 29}},
										&NodeGoStrExpr{Expr: "name", Span: Span{Start: 30, End: 34}},
										&NodeLiteral{Text: "!", Span: Span{Start: 34, End: 35}},
									},
								},
							},
						},
						Alt: &NodeBlock{
							Nodes: []Node{
								&NodeLiteral{Text: "\n\t", Span: Span{Start: 50, End: 52}},
								&NodeElement{
									Tag:           Tag{Name: "h1"},
									StartTagNodes: []Node{&NodeLiteral{Text: "<h1>", Span: Span{Start: 52, End: 56}}},
									Span:          Span{Start: 52, End: 56},
									Children: []Node{
										&NodeLiteral{Text: "Hello, world!", Span: Span{Start: 56, End: 69}},
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
			&SyntaxTree{
				Nodes: []Node{
					&NodeIf{
						Cond: &NodeGoStrExpr{
							Expr: "name == \"\"",
							Span: Span{Start: 4, End: 14},
						},
						Then: &NodeBlock{
							Nodes: []Node{
								&NodeLiteral{Text: "\n    ", Span: Span{Start: 16, End: 21}},
								&NodeElement{
									Tag:           Tag{Name: "div"},
									StartTagNodes: []Node{&NodeLiteral{Text: "<div>", Span: Span{Start: 21, End: 26}}},
									Span:          Span{Start: 21, End: 26},
									Children: []Node{
										&NodeLiteral{Text: "\n        ", Span: Span{Start: 26, End: 35}},
										&NodeElement{
											Tag:           Tag{Name: "h1"},
											StartTagNodes: []Node{&NodeLiteral{Text: "<h1>", Span: Span{Start: 35, End: 39}}},
											Span:          Span{Start: 35, End: 39},
											Children: []Node{
												&NodeLiteral{Text: "Hello, world!", Span: Span{Start: 39, End: 52}},
											},
										},
										&NodeLiteral{Text: "\n    ", Span: Span{Start: 57, End: 62}},
									},
								},
							},
						},
						Alt: &NodeBlock{
							Nodes: []Node{
								&NodeLiteral{Text: "\n    ", Span: Span{Start: 78, End: 83}},
								&NodeElement{
									Tag:           Tag{Name: "div"},
									Span:          Span{Start: 83, End: 88},
									StartTagNodes: []Node{&NodeLiteral{Text: "<div>", Span: Span{Start: 83, End: 88}}},
									Children: []Node{
										&NodeLiteral{Text: "\n        ", Span: Span{Start: 88, End: 97}},
										&NodeElement{
											Tag:           Tag{Name: "h1"},
											StartTagNodes: []Node{&NodeLiteral{Text: "<h1>", Span: Span{Start: 97, End: 101}}},
											Span:          Span{Start: 97, End: 101},
											Children: []Node{
												&NodeLiteral{Text: "Hello, ", Span: Span{Start: 101, End: 108}},
												&NodeGoStrExpr{Expr: "name", Span: Span{Start: 109, End: 113}},
												&NodeLiteral{Text: "!", Span: Span{Start: 113, End: 114}},
											},
										},
										&NodeLiteral{Text: "\n    ", Span: Span{Start: 119, End: 124}},
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
			&SyntaxTree{
				Nodes: []Node{
					&NodeGoCode{
						Code: `type product struct {
		name string
		price float32
	}

	products := []product{{name: "Widget", price: 9.49}}
`,
						Span: Span{Start: 2, End: 112},
					},
				},
			},
		},
		{
			`^import "time"`,
			&SyntaxTree{
				Nodes: []Node{
					&NodeImport{
						Decl: ImportDecl{
							PkgName: "",
							Path:    "\"time\"",
						},
						Span: Span{},
					},
				},
			},
		},
		{
			`<a href="^url">x</a>`,
			&SyntaxTree{
				Nodes: []Node{
					&NodeElement{
						Tag: Tag{
							Name: "a",
							Attrs: []*Attr{
								{
									Name:  StringPos{Text: "href", Start: 3},
									Value: StringPos{Text: "^url", Start: 9},
								},
							},
						},
						StartTagNodes: []Node{
							&NodeLiteral{Text: "<a ", Span: Span{Start: 0, End: 3}},
							&NodeLiteral{Text: "href", Span: Span{Start: 3, End: 7}},
							&NodeLiteral{Text: "=\"", Span: Span{Start: 7, End: 9}},
							&NodeGoStrExpr{Expr: "url", Span: Span{Start: 0, End: 3}}, // FIXME(paulsmith): should be 9 and 13 but this part of the parser is not correct yet
							&NodeLiteral{Text: "\">", Span: Span{Start: 13, End: 15}},
						},
						Children: []Node{
							&NodeLiteral{Text: "x", Span: Span{Start: 15, End: 16}},
						},
						Span: Span{Start: 0, End: 15},
					},
				},
			},
		},
		{
			`^if true { <a href="^url"><div data-^foo="bar"></div></a> }`,
			&SyntaxTree{
				Nodes: []Node{
					&NodeIf{
						Cond: &NodeGoStrExpr{Expr: "true", Span: Span{Start: 4, End: 8}},
						Then: &NodeBlock{
							Nodes: []Node{
								&NodeLiteral{Text: " ", Span: Span{Start: 10, End: 11}},
								&NodeElement{
									Tag: Tag{
										Name: "a",
										Attrs: []*Attr{
											{
												Name:  StringPos{Text: "href", Start: pos(3)},
												Value: StringPos{Text: "^url", Start: pos(9)},
											},
										},
									},
									StartTagNodes: []Node{
										&NodeLiteral{Text: "<a ", Span: Span{Start: 11, End: 14}},
										&NodeLiteral{Text: "href", Span: Span{Start: 14, End: 18}},
										&NodeLiteral{Text: `="`, Span: Span{Start: 18, End: 20}},
										&NodeGoStrExpr{Expr: "url", Span: Span{End: 3}},
										&NodeLiteral{Text: `">`, Span: Span{Start: 24, End: 26}},
									},
									Children: []Node{
										&NodeElement{
											Tag: Tag{
												Name: "div",
												Attrs: []*Attr{
													{
														Name:  StringPos{Text: "data-^foo", Start: pos(5)},
														Value: StringPos{Text: "bar", Start: pos(16)},
													},
												},
											},
											StartTagNodes: []Node{
												&NodeLiteral{Text: "<div ", Span: Span{Start: 26, End: 31}},
												&NodeLiteral{Text: "data-", Span: Span{Start: 31, End: 36}},
												&NodeGoStrExpr{Expr: "foo", Span: Span{Start: 0, End: 3}}, // FIXME
												&NodeLiteral{Text: "=\"", Span: Span{Start: 40, End: 42}},
												&NodeLiteral{Text: "bar", Span: Span{Start: 42, End: 45}},
												&NodeLiteral{Text: "\">", Span: Span{Start: 45, End: 47}},
											},
											Span: Span{Start: 26, End: 47},
										},
									},
									Span: Span{Start: 11, End: 26},
								},
							},
						},
					},
				},
			},
		},
		{
			// example of expanded implicit/simple expression
			`^foo.bar("asd").baz.biz()`,
			&SyntaxTree{
				Nodes: []Node{
					&NodeGoStrExpr{Expr: `foo.bar("asd").baz.biz()`, Span: Span{Start: 1, End: 25}},
				},
			},
		},
		{
			// example of expanded implicit/simple expression
			`^quux[42]`,
			&SyntaxTree{
				Nodes: []Node{
					&NodeGoStrExpr{Expr: `quux[42]`, Span: Span{Start: 1, End: 9}},
				},
			},
		},
		{
			// space separates the Go ident `name` from the next `(`, ending the
			// expression and preventing it from being interpreted as a function
			// call
			`^p.name ($blah ...`,
			&SyntaxTree{
				Nodes: []Node{
					&NodeGoStrExpr{Expr: `p.name`, Span: Span{Start: 1, End: 7}},
					&NodeLiteral{Text: ` ($blah ...`, Span: Span{Start: 7, End: 18}},
				},
			},
		},
		{
			// expanded implicit/simple expression with argument list in func call
			`^getParam(req, "name")`,
			&SyntaxTree{
				Nodes: []Node{
					&NodeGoStrExpr{Expr: `getParam(req, "name")`, Span: Span{Start: 1, End: 22}},
				},
			},
		},
		{
			`^partial foo {<text>bar</text>}`,
			&SyntaxTree{
				Nodes: []Node{
					&NodePartial{
						Name: "foo",
						Span: Span{Start: 8, End: 12},
						Block: &NodeBlock{
							Nodes: []Node{
								&NodeBlock{Nodes: []Node{&NodeLiteral{Text: "bar", Span: Span{Start: 20, End: 23}}}},
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
			&SyntaxTree{
				Nodes: []Node{
					&NodePartial{
						Name: "foo",
						Span: Span{Start: 8, End: 12},
						Block: &NodeBlock{
							Nodes: []Node{
								&NodeIf{
									Cond: &NodeGoStrExpr{Expr: "true", Span: Span{23, 27}},
									Then: &NodeBlock{
										Nodes: []Node{
											&NodeLiteral{Text: "\n\t\t\t\t\t", Span: Span{Start: 29, End: 35}},
											&NodeElement{
												Tag:           Tag{Name: "p"},
												StartTagNodes: []Node{&NodeLiteral{Text: "<p>", Span: Span{Start: 35, End: 38}}},
												Span:          Span{35, 38},
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
			&SyntaxTree{
				Nodes: []Node{
					&NodeLiteral{Text: "My name is ", Span: Span{End: 11}},
					&NodeGoStrExpr{Expr: "name", Span: Span{Start: 12, End: 16}},
					&NodeLiteral{Text: ". What's yours?", Span: Span{Start: 16, End: 31}},
				},
			},
		},
		{
			`<p>^foo.</p>`,
			&SyntaxTree{
				Nodes: []Node{
					&NodeElement{
						Tag: Tag{Name: "p"},
						StartTagNodes: []Node{
							&NodeLiteral{Text: "<p>", Span: Span{End: 3}},
						},
						Children: []Node{
							&NodeGoStrExpr{Expr: "foo", Span: Span{Start: 4, End: 7}},
							&NodeLiteral{Text: ".", Span: Span{Start: 7, End: 8}},
						},
						Span: Span{Start: 0, End: 3},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, err := parse(test.input)
			if err != nil {
				t.Fatalf("unexpected error parsing input: %v", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("expected parse diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseGolden(t *testing.T) {
	/*
		in := `^import "strings"

			^if true {
				<text>Hi</text>
			} else {
				<text>Bye</text>
			}

			^{
				x := strings.Fields(" asd = 123 ")
			}
			<div class="container">
				^if true {
					<p>Hello, ^x[0]!</p>
				}
			</div>
			`
	*/
	in := `^if true {<text>Hi</text>}`
	//in := `^{ x := 0 }`
	actual, err := parse(in)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	t.Logf("actual parse result:")
	b, err := json.MarshalIndent(actual, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(b))
	var expected SyntaxTree
	if err := json.Unmarshal([]byte(b), &expected); err != nil {
		t.Fatalf("failed to unmarshal golden: %v", err)
	}
	if diff := cmp.Diff(&expected, actual); diff != "" {
		t.Errorf("unexpected parse result (-expected +actual):\n%s", diff)
	}
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
		tag  Tag
		want string
	}{
		{
			Tag{Name: "h1"},
			"h1",
		},
		{
			Tag{Name: "div", Attrs: []*Attr{{Name: StringPos{Text: "class"}, Value: StringPos{Text: "banner"}}}},
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
