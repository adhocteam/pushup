package lexer

import (
	"bytes"
	"errors"
	"fmt"
	"go/scanner"
	"go/token"
	"io"
	"iter"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/source"
	"github.com/lmittmann/tint"
	"golang.org/x/net/html"
)

func init() {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, nil)))
}

type Lexer struct {
	source []byte
	state  state

	start   int
	pos     int
	current rune

	// html state
	hz          *html.Tokenizer
	htok        html.TokenType
	hraw        []byte
	hattrCursor *attrCursor

	// go state
	gfile    *token.File
	gfset    *token.FileSet
	gscanner *scanner.Scanner
}

func New(source []byte) *Lexer {
	l := &Lexer{
		source: source,
		state:  stateHTML,
	}
	return l
}

func (l *Lexer) advance() {
	var size int
	l.current, size = utf8.DecodeRune(l.src())
	l.pos += size
}

func (l *Lexer) Scan() iter.Seq[Token] {
	return func(yield func(Token) bool) {
		for {
			token := l.next()
			if !yield(token) {
				return
			}
			if token.tokType == EOF {
				return
			}
		}
	}
}

func (l *Lexer) src() []byte {
	slog.Debug("src", "slice", string(l.source[l.pos:]), "pos", l.pos)
	return l.source[l.pos:]
}

func (l *Lexer) next() Token {
	for {
		slog.Debug("next token", "state", l.state)

		switch l.state {
		case stateHTML:
			l.hz = html.NewTokenizer(bytes.NewReader(l.src()))
			l.htok = l.hz.Next()
			l.hraw = l.hz.Raw()
			slog.Debug("HTML token", "token", l.htok, "raw", string(l.hraw))

			switch l.htok {
			case html.ErrorToken:
				err := l.hz.Err()
				if errors.Is(err, io.EOF) {
					return l.emit(EOF)
				}
				slog.Error("HTML tokenizer", "error", err)

			case html.TextToken:
				idx := bytes.IndexRune(l.hraw, '^')
				slog.Debug("text", "idx", idx)
				if idx == -1 {
					l.pos += len(l.hraw)
					return l.emit(HTML_TEXT)
				}
				l.pos += idx
				token := l.emit(HTML_TEXT) // emit text preceding the transition
				l.pos += 1                 // skip past '^'
				l.start = l.pos
				l.switchState(stateGo)
				// don't emit an empty token
				if idx > 0 {
					return token
				}
				continue

			case html.StartTagToken, html.SelfClosingTagToken:
				tagName, hasAttrs := l.hz.TagName()
				n := len("<" + string(tagName))
				if hasAttrs {
					attrs, err := scanAttrs(string(l.hraw), l.pos)
					if err != nil {
						slog.Error("scanAttrs", "error", err)
						return l.emit(ILLEGAL)
					}
					l.hattrCursor = newAttrCursor(attrs)
					l.switchState(stateHTMLAttr)
					l.pos += n // move past the "<tagname "
					token := l.emit(HTML_START_TAG_NAME)
					return token
				} else {
					l.switchState(stateHTMLAfterLastAttr)
					l.pos += n
					return l.emit(HTML_START_TAG_NAME)
				}

			case html.EndTagToken:
				l.pos += len(l.hraw)
				return l.emit(HTML_END_TAG)

			case html.CommentToken:
				l.pos += len(l.hraw)
				return l.emit(HTML_TEXT)

			case html.DoctypeToken:
				l.pos += len(l.hraw)
				return l.emit(HTML_TEXT)

			default:
				panic(fmt.Sprintf("unexpected HTML token type: %v", l.htok))
			}

		case stateHTMLAttr:
			slog.Debug("html attr", "hasMore", l.hattrCursor.hasMore())
			if !l.hattrCursor.advance() {
				l.switchState(stateHTMLAfterLastAttr)
				continue
			}

			// advance to start of attribute name but that's
			// indexed from beginning of tag <, so adjust for that
			l.pos = int(l.hattrCursor.name.Start)
			l.start = l.pos

			l.switchState(stateHTMLAttrName)

		case stateHTMLAttrName:
			slog.Debug("html attr name", "src", string(l.src()), "start", l.hattrCursor.name.Start, "text", l.hattrCursor.name.Text)
			l.start = int(l.hattrCursor.name.Start)
			idx := strings.IndexRune(l.hattrCursor.name.Text, '^')
			// No transition
			if idx == -1 {
				l.pos = l.start + len(l.hattrCursor.name.Text)
				token := l.emit(HTML_ATTR_NAME_TEXT)
				l.switchState(stateHTMLAttrValue)
				l.pos = int(l.hattrCursor.value.Start)
				return token
			} else if idx > 0 {
				l.pos += idx
				token := l.emit(HTML_ATTR_NAME_TEXT)
				l.start++
				l.switchState(stateHTMLAttrNameGoExpr)
				return token
			}
			// idx == 0
			l.start++
			l.switchState(stateHTMLAttrNameGoExpr)
			continue

		case stateHTMLAttrValue:
			slog.Debug("html attr value", "src", string(l.src()), "start", l.hattrCursor.value.Start, "text", l.hattrCursor.value.Text)
			l.start = int(l.hattrCursor.value.Start)
			idx := strings.IndexRune(l.hattrCursor.value.Text, '^')
			slog.Debug("attr value transition", "idx", idx)
			// No transition
			if idx == -1 {
				l.pos = l.start + len(l.hattrCursor.value.Text)
				token := l.emit(HTML_ATTR_VALUE_TEXT)
				l.switchState(stateHTMLAttr)
				return token
			} else if idx > 0 {
				l.pos += idx
				token := l.emit(HTML_ATTR_VALUE_TEXT)
				l.start++
				l.switchState(stateHTMLAttrValueGoExpr)
				return token
			}
			// idx == 0
			l.start++
			l.pos += idx
			l.switchState(stateHTMLAttrValueGoExpr)
			continue

		case stateHTMLAttrNameGoExpr:
			slog.Debug("attr name go expr", "l.pos", l.pos, "src", l.src())
			l.pos = int(l.hattrCursor.name.Start) + len(l.hattrCursor.name.Text)
			l.switchState(stateHTMLAttrValue)
			return l.emit(HTML_ATTR_NAME_GO)

		case stateHTMLAttrValueGoExpr:
			slog.Debug("attr value go expr", "l.pos", l.pos, "src", string(l.src()))
			l.pos = int(l.hattrCursor.value.Start) + len(l.hattrCursor.value.Text)
			l.switchState(stateHTMLAttr)
			return l.emit(HTML_ATTR_VALUE_GO)

		case stateHTMLAfterLastAttr:
			// There may be arbitrary whitespace between the end of the
			// attributes (or tag name, if no attributes) and the > or /> of
			// the tag.
			src := l.src()
			for src[0] == ' ' || src[0] == '\n' || src[0] == '\t' {
				src = src[1:]
				l.pos++
			}
			slog.Debug("after last", "src", string(src))
			if bytes.HasPrefix(src, []byte(">")) {
				l.pos += 1
				l.switchState(stateHTML)
				return l.emit(HTML_GT)
			} else if bytes.HasPrefix(src, []byte("/>")) {
				l.pos += 2
				l.switchState(stateHTML)
				return l.emit(HTML_SELF_CLOSING_GT)
			} else {
				panic(fmt.Sprintf("expected '>' or '/>', found %q", src))
			}

		case stateGo:
			l.gscanner = new(scanner.Scanner)
			l.gfset = token.NewFileSet()
			src := l.src()
			l.gfile = l.gfset.AddFile("", l.pos, len(src))
			l.gscanner.Init(l.gfile, src, nil, scanner.ScanComments)
			pos, tok, lit := l.gscanner.Scan()
			slog.Debug("go scanner", "l.pos", l.pos, "pos", pos, "tok", tok, "lit", lit, "file.Offset(pos)", l.gfile.Offset(pos), "fset.Position(pos)", l.gfset.Position(pos))
			l.pos = int(pos)
			switch tok {
			case token.EOF:
				break
			case token.IF:
				l.pos += len("if")
				l.switchState(stateGoCondExpr)
				return l.emit(IF)
			default:
				panic(fmt.Sprintf("unhandled Go token: %v", tok))
			}

		case stateGoCondExpr:
			pos, tok, lit := l.gscanner.Scan()
			for tok != token.LBRACE {
				l.pos = int(pos)
				pos, tok, lit = l.gscanner.Scan()
				slog.Debug("stateGoCondExpr", "pos", pos, "tok", tok, "lit", lit)
			}
			l.pos = int(pos)
			token := l.emit(GO_EXPR)
			l.pos++ // skip past the {
			l.start = l.pos
			l.switchState(stateHTML)
			return token
		}
	}
}

func (l *Lexer) switchState(s state) {
	slog.Debug("switch state", "exiting", l.state, "entering", s)
	l.state = s
}

func (l *Lexer) backup() {
	l.pos -= utf8.RuneLen(l.current)
}

func (l *Lexer) emit(tokType TokenType) Token {
	token := Token{
		tokType: tokType,
		literal: l.source[l.start:l.pos],
		loc:     source.Span{Start: l.start, Len: l.pos - l.start},
	}
	l.start = l.pos
	slog.Debug("emitting token", "tokType", tokType, "literal", string(token.literal), "loc", token.loc)
	return token
}

var keywords = map[string]TokenType{
	"for":     FOR,
	"if":      IF,
	"import":  IMPORT,
	"param":   PARAM,
	"partial": PARTIAL,
}

func (l *Lexer) matchesPrefix(b []byte) bool {
	return bytes.HasPrefix(l.source[l.pos:], b)
}

func (l *Lexer) transition() Token {
	defer l.switchState(stateGo)

	for kw, tokType := range keywords {
		if l.matchesPrefix([]byte(kw)) {
			l.pos += len(kw)
			return l.emit(tokType)
		}
	}

	if l.current == '{' {
		l.advance()
		l.emit(GO_BLOCK_BEGIN)
	}

	if l.current == '(' {
		l.advance()
		l.emit(GO_EXPLICIT_EXPR_BEGIN)
	}

	return l.emit(GO_IMPLICIT_EXPR_BEGIN)
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isNewline(ch rune) bool {
	return ch == '\n' || ch == '\r'
}

type state int

const (
	stateHTML state = iota
	stateHTMLAttr
	stateHTMLAttrName
	stateHTMLAttrNameGoExpr
	stateHTMLAttrValue
	stateHTMLAttrValueGoExpr
	stateHTMLAfterLastAttr
	stateGo
	stateGoCondExpr
)

var states = [...]string{
	stateHTML:                "stateHTML",
	stateHTMLAttr:            "stateHTMLAttr",
	stateHTMLAttrName:        "stateHTMLAttrName",
	stateHTMLAttrNameGoExpr:  "stateHTMLAttrNameGoExpr",
	stateHTMLAttrValue:       "stateHTMLAttrValue",
	stateHTMLAttrValueGoExpr: "stateHTMLAttrValueGoExpr",
	stateHTMLAfterLastAttr:   "stateHTMLAfterLastAttr",
	stateGo:                  "stateGo",
	stateGoCondExpr:          "stateGoCondExpr",
}

func (s state) String() string {
	str := ""
	if 0 <= s && s < state(len(states)) {
		str = states[s]
	}
	if str == "" {
		str = "state(" + strconv.Itoa(int(s)) + ")"
	}
	return str
}

type attrCursor struct {
	attrs       []*ast.Attr
	current     int              // index into attrs
	name, value source.StringPos // convenience accessors
}

func newAttrCursor(attrs []*ast.Attr) *attrCursor {
	c := &attrCursor{
		attrs:   attrs,
		current: -1,
	}
	return c
}

func (c attrCursor) hasMore() bool {
	return c.current < len(c.attrs)
}

func (c *attrCursor) advance() bool {
	c.current++
	if !c.hasMore() {
		return false
	}
	c.name = c.attrs[c.current].Name
	c.value = c.attrs[c.current].Value
	return true
}

func matchesBlockOpen(text []byte) (bracePos int, ok bool) {
	i := 0

	for i < len(text) && isWhitespace(rune(text[i])) && !isNewline(rune(text[i])) {
		i++
	}

	if i >= len(text) || text[i] != '{' {
		return -1, false
	}
	bracePos = i
	i++

	for i < len(text) && isWhitespace(rune(text[i])) {
		if isNewline(rune(text[i])) {
			return bracePos, true
		}
		i++
	}

	return -1, false
}

func matchesBlockClose(text []byte) (bracePos int, ok bool) {
	i := 0

	for i < len(text) && isWhitespace(rune(text[i])) {
		i++
	}

	if i >= len(text) || text[i] != '}' {
		return -1, false
	}
	bracePos = i

	return bracePos, true
}
