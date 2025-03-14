package lexer

import (
	"fmt"
	"go/token"
	"strconv"
	"strings"

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

	HTMLTagOpen
	HTMLTagClose
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

func (t HTMLTokenType) String() string {
	switch t {
	case HTMLError:
		return "Error"
	case HTMLDoctype:
		return "Doctype"
	case HTMLComment:
		return "Comment"
	case HTMLText:
		return "Text"
	case HTMLStartTag:
		return "StartTag"
	case HTMLSelfClosingTag:
		return "SelfClosingTag"
	case HTMLEndTag:
		return "EndTag"
	case HTMLTagOpen:
		return "TagOpen"
	case HTMLTagClose:
		return "TagClose"
	default:
		return fmt.Sprintf("HTMLTokenType(%d)", int(t))
	}
}

type EOF int

var eofLit = []byte("EOF")

func (_ EOF) Lit() []byte {
	return eofLit
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

func (t Transition) String() string {
	return "^(" + strconv.Itoa(int(t)) + ")"
}

type AttrToken struct {
	lit []byte
	pos int
}

func (t AttrToken) Lit() []byte {
	return t.lit
}

func (t AttrToken) Pos() int {
	return t.pos
}

func (t AttrToken) String() string {
	return "AttrToken(" + string(t.lit) + ")"
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
	return "HTMLToken(" + strings.ReplaceAll(string(t.lit), "\n", "\\n") + ")"
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

func (t GoToken) Lit() []byte {
	if t.Type.IsLiteral() || t.Type.IsKeyword() || t.Type == token.SEMICOLON || t.Type == token.COMMENT || t.Type == token.ILLEGAL {
		return t.lit
	} else {
		return []byte(t.Type.String())
	}
}

func (t GoToken) Pos() int {
	return t.pos
}

func IsGoToken(t Token) bool {
	// TODO: handle pointers?
	_, ok := t.(GoToken)
	return ok
}

func (t GoToken) String() string {
	return "GoToken(" + strings.ReplaceAll(string(t.Lit()), "\n", "\\n") + ")"
}
