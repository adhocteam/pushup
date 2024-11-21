package lexer

import (
	"go/token"

	"golang.org/x/net/html"
)

type Token interface {
	Lit() []byte
	Pos() int
}

type HTMLTokenType int

const (
	HTMLError HTMLTokenType = iota
	HTMLDoctype
	HTMLComment
	HTMLText
	HTMLStartTag
	HTMLSelfClosingTag
	HTMLEndTag
)

var htmlTypeMap = [...]HTMLTokenType{
	html.ErrorToken:          HTMLError,
	html.DoctypeToken:        HTMLDoctype,
	html.CommentToken:        HTMLComment,
	html.TextToken:           HTMLText,
	html.StartTagToken:       HTMLStartTag,
	html.SelfClosingTagToken: HTMLSelfClosingTag,
	html.EndTagToken:         HTMLEndTag,
}

type EOF int

var eof = []byte("EOF")

func (_ EOF) Lit() []byte {
	return eof
}

func (e EOF) Pos() int {
	return int(e)
}

func IsEOF(t Token) bool {
	_, ok := t.(EOF)
	return ok
}

type Transition int

var up = []byte("^")

func (_ Transition) Lit() []byte {
	return up
}

func (t Transition) Pos() int {
	return int(t)
}

func IsTransition(t Token) bool {
	_, ok := t.(Transition)
	return ok
}

type HTMLToken struct {
	Type HTMLTokenType
	lit  []byte
	pos  int
}

func newHTMLToken(tokType html.TokenType, lit []byte, pos int) HTMLToken {
	t := HTMLToken{
		Type: htmlTypeMap[tokType],
		lit:  lit,
		pos:  pos,
	}
	return t
}

func (t HTMLToken) String() string {
	return string(t.lit)
}

func (ht HTMLToken) Lit() []byte {
	return ht.lit
}

func (ht HTMLToken) Pos() int {
	return ht.pos
}

func IsHTMLToken(t Token) bool {
	// TODO: handle pointers?
	_, ok := t.(HTMLToken)
	return ok
}

type GoToken struct {
	Type token.Token
	lit  []byte
	pos  int
}

func (gt GoToken) Lit() []byte {
	return gt.lit
}

func (gt GoToken) Pos() int {
	return gt.pos
}

func IsGoToken(t Token) bool {
	// TODO: handle pointers?
	_, ok := t.(GoToken)
	return ok
}

func (t GoToken) String() string {
	return string(t.lit)
}
