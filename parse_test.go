package main

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  parseResult
	}{
		{
			`<p>Hello, @name!</p>
@code {
	name := "world"
}
`,
			parseResult{
				exprs: []expr{
					exprLiteral{str: "<p>", typ: literalHTML, pos: span{start: 0, end: 3}},
					exprLiteral{str: "Hello, ", typ: literalRawString, pos: span{start: 3, end: 10}},
					exprVar{name: "name", pos: span{start: 10, end: 15}},
					exprLiteral{str: "!", typ: literalRawString, pos: span{start: 15, end: 16}},
					exprLiteral{str: "</p>", typ: literalHTML, pos: span{start: 16, end: 20}},
					exprCode{code: "\n\tname := \"world\"\n", pos: span{start: 20, end: 48}},
				}},
		},
		{
			`<p class="greeting">I @emotion Chicago!</p>
@code {
	emotion := "<em><3</em> & <strong>blue circle</strong>"
}

<div id="end">More</div>
`,
			parseResult{
				exprs: []expr{
					exprLiteral{str: "<p class=\"greeting\">", typ: 0, pos: span{start: 0, end: 20}},
					exprLiteral{str: "I ", typ: 1, pos: span{start: 20, end: 22}},
					exprVar{name: "emotion", pos: span{start: 22, end: 30}},
					exprLiteral{str: " Chicago!", typ: 1, pos: span{start: 30, end: 39}},
					exprLiteral{str: "</p>", typ: 0, pos: span{start: 39, end: 43}},
					exprCode{code: "\n\temotion := \"<em><3</em> & <strong>blue circle</strong>\"\n", pos: span{start: 43, end: 112}},
					exprLiteral{str: "<div id=\"end\">", typ: 0, pos: span{start: 112, end: 126}},
					exprLiteral{str: "More", typ: 1, pos: span{start: 126, end: 130}},
					exprLiteral{str: "</div>", typ: 0, pos: span{start: 130, end: 136}},
					exprLiteral{str: "\n", typ: 1, pos: span{start: 136, end: 137}}},
			},
		},
		{
			`<p>Don't break Go code</p>
@code {
	var foo = "This is a variable"
	bar := func(a int, b int) (bool, error) {
		if (a < b) {
			return false, nil
		}
		return true, fmt.Errorf("error")
	}
}
`,
			parseResult{
				exprs: []expr{exprLiteral{str: "<p>", typ: 0, pos: span{start: 0, end: 3}},
					exprLiteral{str: "Don't break Go code", typ: 1, pos: span{start: 3, end: 22}},
					exprLiteral{str: "</p>", typ: 0, pos: span{start: 22, end: 26}},
					exprCode{code: "\n\tvar foo = \"This is a variable\"\n\tbar := func(a int, b int) (bool, error) {\n\t\tif (a < b) {\n\t\t\treturn false, nil\n\t\t}\n\t\treturn true, fmt.Errorf(\"error\")\n\t}\n", pos: span{start: 26, end: 190}}},
			},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, err := parsePushup(test.input)
			if err != nil {
				t.Fatalf("unexpected error parsing input: %v", err)
			}
			if err := testParseResultsEqual(test.want, got); err != nil {
				t.Errorf("parse results not equal: %v", err)
			}
		})
	}
}

func testParseResultsEqual(a, b parseResult) error {
	if len(a.exprs) != len(b.exprs) {
		return fmt.Errorf("# of exprs: want: %d got %d\nwant:\n%#v\n===\ngot:\n%#v", len(a.exprs), len(b.exprs), a.exprs, b.exprs)
	}
	for i, e := range a.exprs {
		if e != b.exprs[i] {
			return fmt.Errorf("expr %d: want:\n%#v\ngot:\n%#v", i, e, b.exprs[i])
		}
	}
	return nil
}
