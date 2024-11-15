package lexer

import "strconv"

type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF

	HTML_IDENT // a, div, layout.Base
	HTML_TEXT
	HTML_START_TAG_NAME  // "<p" -- i.e., opening < and tag name but no attributes or closing >
	HTML_GT              // ">"
	HTML_SELF_CLOSING_GT // "/>"
	HTML_END_TAG         // "</p>"

	GO_IDENT
	GO_EXPR                // x > 0
	GO_IMPLICIT_EXPR_BEGIN // ^
	GO_EXPLICIT_EXPR_BEGIN // ^(
	GO_BLOCK_BEGIN         // ^{

	CARET      // escaped transition symbol, "^^" in source but '^' literal
	LBRACE     // {
	RBRACE     // }
	LPAREN     // (
	RPAREN     // )
	WHITESPACE // [ \n\t"]
	LT         // <
	GT         // >
	DOT        // .
	START      // *

	// Keywords
	keywords_begin
	IMPORT  // ^import
	IF      // ^if
	FOR     // ^for
	PARAM   // ^param
	PARTIAL // ^partial
	keywords_end
)

var tokenTypes = [...]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",

	HTML_IDENT:           "HTML_IDENT",
	HTML_TEXT:            "HTML_TEXT",
	HTML_START_TAG_NAME:  "HTML_START_TAG_NAME",
	HTML_GT:              "HTML_GT",
	HTML_SELF_CLOSING_GT: "HTML_SELF_CLOSING_GT",
	HTML_END_TAG:         "HTML_END_TAG",

	GO_IDENT:               "GO_IDENT",
	GO_EXPR:                "GO_EXPR",
	GO_IMPLICIT_EXPR_BEGIN: "^",
	GO_EXPLICIT_EXPR_BEGIN: "^(",
	GO_BLOCK_BEGIN:         "^{",

	CARET:      "^",
	LBRACE:     "{",
	RBRACE:     "}",
	LPAREN:     "(",
	RPAREN:     ")",
	WHITESPACE: "<ws>",
	LT:         "<",
	GT:         ">",
	DOT:        ".",
	START:      "*",

	IMPORT:  "^import",
	IF:      "^if",
	FOR:     "^for",
	PARAM:   "^param",
	PARTIAL: "^partial",
}

func (t TokenType) String() string {
	var s string
	if 0 <= t && t < TokenType(len(tokenTypes)) {
		s = tokenTypes[t]
	}
	if s == "" {
		s = "token(" + strconv.Itoa(int(t)) + ")"
	}
	return s
}
