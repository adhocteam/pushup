package lexer

import "strconv"

type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF

	HTML_TEXT
	HTML_START_TAG_NAME  // "<p" -- i.e., opening < and tag name but no attributes or closing >
	HTML_GT              // ">"
	HTML_SELF_CLOSING_GT // "/>"
	HTML_END_TAG         // "</p>"
	HTML_ATTR_NAME_TEXT  // attribute text in the name position
	HTML_ATTR_VALUE_TEXT // attribute text in the value position
	HTML_ATTR_NAME_GO    // attribute Go expression in the name position
	HTML_ATTR_VALUE_GO   // attribute Go expression in the value position

	GO_IDENT
	GO_EXPR                // x > 0
	GO_IMPLICIT_EXPR_BEGIN // ^
	GO_EXPLICIT_EXPR_BEGIN // ^(
	GO_BLOCK_BEGIN         // ^{
	GO_BLOCK_OPEN          // { at end of line
	GO_BLOCK_CLOSE         // } at start of line

	WHITESPACE // [ \n\t"]

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

	HTML_TEXT:            "HTML_TEXT",
	HTML_START_TAG_NAME:  "HTML_START_TAG_NAME",
	HTML_GT:              "HTML_GT",
	HTML_SELF_CLOSING_GT: "HTML_SELF_CLOSING_GT",
	HTML_END_TAG:         "HTML_END_TAG",
	HTML_ATTR_NAME_TEXT:  "HTML_ATTR_NAME_TEXT",
	HTML_ATTR_VALUE_TEXT: "HTML_ATTR_VALUE_TEXT",
	HTML_ATTR_NAME_GO:    "HTML_ATTR_NAME_GO",
	HTML_ATTR_VALUE_GO:   "HTML_ATTR_VALUE_GO",

	GO_IDENT:               "GO_IDENT",
	GO_EXPR:                "GO_EXPR",
	GO_IMPLICIT_EXPR_BEGIN: "^",
	GO_EXPLICIT_EXPR_BEGIN: "^(",
	GO_BLOCK_BEGIN:         "^{",
	GO_BLOCK_OPEN:          "GO_BLOCK_OPEN",
	GO_BLOCK_CLOSE:         "GO_BLOCK_CLOSE",

	WHITESPACE: "<ws>",

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
