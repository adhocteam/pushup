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
	mode   mode
	state  state

	start   int
	pos     int
	current rune

	// html state
	hattrCursor *attrCursor

	// go state
	gscanner *bufGoScanner
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
			z := html.NewTokenizer(bytes.NewReader(l.src()))
			tok := z.Next()
			raw := z.Raw()
			slog.Debug("HTML token", "token", tok, "raw", string(raw))

			switch tok {
			case html.ErrorToken:
				err := z.Err()
				if errors.Is(err, io.EOF) {
					return l.emit(EOF)
				}
				slog.Error("HTML tokenizer", "error", err)

			case html.TextToken:
				idx := bytes.IndexRune(raw, '^')
				slog.Debug("text", "idx", idx)

				// transition detected
				if idx >= 0 {
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
				}

				if bracePos := matchesBlockClose(raw); bracePos != -1 {
					slog.Debug("matchesBlockClose", "raw", string(raw), "bracePos", bracePos, "l.src()", string(l.src()))
					l.pos += bracePos
					token := l.emit(HTML_TEXT)
					l.switchState(stateGoBlockClose)
					return token
				}

				l.pos += len(raw)
				return l.emit(HTML_TEXT)

			case html.StartTagToken, html.SelfClosingTagToken:
				tagName, hasAttrs := z.TagName()
				n := len("<" + string(tagName))
				if hasAttrs {
					attrs, err := scanAttrs(string(raw), l.pos)
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
				l.pos += len(raw)
				return l.emit(HTML_END_TAG)

			case html.CommentToken:
				l.pos += len(raw)
				return l.emit(HTML_TEXT)

			case html.DoctypeToken:
				l.pos += len(raw)
				return l.emit(HTML_TEXT)

			default:
				panic(fmt.Sprintf("unexpected HTML token type: %v", tok))
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
			tok := l.gscanner.get()
			l.pos = int(tok.pos)
			switch tok.tok {
			case token.EOF:
				break
			case token.IF:
				l.pos += len("if")
				l.switchState(stateGoCondExpr)
				return l.emit(IF)
			case token.FOR:
				l.pos += len("for")
				l.switchState(stateGoCondExpr)
				return l.emit(FOR)
			default:
				panic(fmt.Sprintf("unhandled Go token: %v", tok))
			}

		case stateGoCondExpr:
			tok := l.gscanner.get()
			for tok.tok != token.LBRACE {
				l.pos = int(tok.pos)
				tok = l.gscanner.get()
				slog.Debug("stateGoCondExpr", "pos", tok.pos, "tok", tok.tok, "lit", tok.lit)
			}
			l.gscanner.unget()
			l.pos = int(tok.pos)
			token := l.emit(GO_EXPR)
			l.switchState(stateGoBlockOpen)
			return token

		case stateGoBlockOpen:
			tok := l.gscanner.get()
			if tok.tok != token.LBRACE {
				panic(fmt.Sprintf("want LBRACE, got %v", tok.tok))
			}
			l.pos = int(tok.pos) + 1
			l.switchState(stateHTML)
			return l.emit(GO_BLOCK_OPEN)

		case stateGoBlockClose:
			tok := l.gscanner.get()
			if tok.tok != token.RBRACE {
				panic(fmt.Sprintf("want RBRACE, got %v", tok.tok))
			}
			l.pos = int(tok.pos) + 1
			l.switchState(stateHTML)
			return l.emit(GO_BLOCK_CLOSE)
		}
	}
}

// syncGoScanner synchronizes the current furthest read state of the Pushup
// source with the Go tokenizer. It should be called each time there is a
// transition from an HTML state to a Go state.
func (l *Lexer) syncGoScanner() {
	scan := new(scanner.Scanner)
	fset := token.NewFileSet()
	src := l.src()
	file := fset.AddFile("", l.pos, len(src))
	scan.Init(file, src, nil, scanner.ScanComments)
	l.gscanner = newBufGoScanner(scan)
}

func (l *Lexer) expectChar(ch byte) {
	if current := l.src()[0]; current != ch {
		panic(fmt.Sprintf("unexpected char: want %q, got %q", ch, current))
	}
	l.pos++
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

func (l *Lexer) switchState(s state) {
	mode, ok := stateModeMap[s]
	if !ok {
		panic(fmt.Sprintf("state not mapped to a mode: %v", s))
	}
	slog.Debug("switch state", "exiting", l.state, "entering", s)
	if l.mode == modeHTML && mode == modeGo {
		l.syncGoScanner()
	}
	l.mode = mode
	l.state = s
}

type mode int

const (
	modeHTML mode = iota
	modeGo
)

var modes = [...]string{
	modeHTML: "modeHTML",
	modeGo:   "modeGo",
}

func (m mode) String() string {
	str := ""
	if 0 <= m && m < mode(len(modes)) {
		str = modes[m]
	}
	if str == "" {
		str = "mode(" + strconv.Itoa(int(m)) + ")"
	}
	return str
}

var stateModeMap = map[state]mode{
	stateHTML:                modeHTML,
	stateHTMLAttr:            modeHTML,
	stateHTMLAttrName:        modeHTML,
	stateHTMLAttrNameGoExpr:  modeHTML,
	stateHTMLAttrValue:       modeHTML,
	stateHTMLAttrValueGoExpr: modeHTML,
	stateHTMLAfterLastAttr:   modeHTML,
	stateGo:                  modeGo,
	stateGoCondExpr:          modeGo,
	stateGoBlockOpen:         modeGo,
	stateGoBlockClose:        modeGo,
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
	stateGoBlockOpen
	stateGoBlockClose
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
	stateGoBlockOpen:         "stateGoBlockOpen",
	stateGoBlockClose:        "stateGoBlockClose",
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

// In a block of plain text (eg. HTML text token), matches whether a { is at
// the end of a line, possibly preceded by whitespace.
func matchesBlockOpen(text []byte) (bracePos int) {
	i := 0

	for i < len(text) && isWhitespace(rune(text[i])) && !isNewline(rune(text[i])) {
		i++
	}

	if i >= len(text) || text[i] != '{' {
		return -1
	}
	bracePos = i
	i++

	for i < len(text) && isWhitespace(rune(text[i])) {
		if isNewline(rune(text[i])) {
			return bracePos
		}
		i++
	}

	return -1
}

// In a block of plain text (eg. HTML text token), matches whether a } is at
// the start of a line, possibly preceded by whitespace.
func matchesBlockClose(text []byte) (bracePos int) {
	i := 0

	for i < len(text) && isWhitespace(rune(text[i])) {
		i++
	}

	if i >= len(text) || text[i] != '}' {
		return -1
	}
	bracePos = i

	return
}

type goToken struct {
	pos token.Pos
	tok token.Token
	lit string
}

type bufGoScanner struct {
	scanner *scanner.Scanner
	buf     *goToken
	last    *goToken
}

func newBufGoScanner(s *scanner.Scanner) *bufGoScanner {
	return &bufGoScanner{scanner: s}
}

func (s *bufGoScanner) bufEmpty() bool {
	return s.buf == nil
}

func (s *bufGoScanner) get() goToken {
	if s.bufEmpty() {
		var tok goToken
		tok.pos, tok.tok, tok.lit = s.scanner.Scan()
		s.last = &tok
		s.buf = s.last
	}
	tok := *s.buf
	s.buf = nil
	return tok
}

func (s *bufGoScanner) unget() {
	if s.bufEmpty() && s.last != nil {
		s.buf = s.last
	} else {
		panic("unget() before call to get()")
	}
}
