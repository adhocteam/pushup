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

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/source"
	"github.com/lmittmann/tint"
	"golang.org/x/net/html"
)

func init() {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, nil)))
}

type Lexer struct {
	input            []byte
	start            int // start of current token being scanned
	pos              int // furthest position read
	width            int // length of the last token
	state            state
	attrs            []*ast.Attr // attributes of last tokenized HTML start/self-close tag
	current          any         // current token scanned, HTML or Go
	currentHtmlToken htmlToken
	currentGoToken   goToken
	currentAttr      *ast.Attr
	goScanner        *scanner.Scanner // Go scanner as of last transition (needed because of consequtive tokens for state tracking)
	htmlTokenizer    *html.Tokenizer  // HTML tokenizer as of last transition
	flag             flag

	tmp     []Token // a temporary buffer of tokens before they get emitted or discarded
	emitted []Token // a buffer of emitted tokens (see Next())
}

type flag int

const (
	flagSelfClosing flag = 1 << iota
)

type state int

const (
	stateHTML state = iota
	stateHTMLStartOrSelfCloseTag
	stateHTMLAttr
	stateHTMLAttrName
	stateHTMLAttrNameGoExpr
	stateHTMLAttrValue
	stateHTMLAttrValueGoExpr
	stateHTMLStartOrSelfCloseTagEnd
	stateError
	stateGoStart
	stateGoIf
	stateGoImport
	stateGoFor
	stateGoPartial
	stateGoParam
	stateGoImplicitExpr
	stateGoExplicitExpr
	stateGoAccumulate
	stateGoAfterSemi
	stateGoAfterLessThan
	stateGoAfterQuotient
	stateGoAfterIdent
)

var states = [...]string{
	stateHTML:                       "stateHTML",
	stateHTMLStartOrSelfCloseTag:    "stateHTMLStartOrSelfCloseTag",
	stateHTMLAttr:                   "stateHTMLAttr",
	stateHTMLAttrName:               "stateHTMLAttrName",
	stateHTMLAttrNameGoExpr:         "stateHTMLAttrNameGoExpr",
	stateHTMLAttrValue:              "stateHTMLAttrValue",
	stateHTMLAttrValueGoExpr:        "stateHTMLAttrValueGoExpr",
	stateHTMLStartOrSelfCloseTagEnd: "stateHTMLStartOrSelfCloseTagEnd",
	stateError:                      "stateError",
	stateGoStart:                    "stateGoStart",
	stateGoIf:                       "stateGoIf",
	stateGoImport:                   "stateGoImport",
	stateGoFor:                      "stateGoFor",
	stateGoPartial:                  "stateGoPartial",
	stateGoParam:                    "stateGoParam",
	stateGoImplicitExpr:             "stateGoImplicitExpr",
	stateGoExplicitExpr:             "stateGoExplicitExpr",
	stateGoAccumulate:               "stateGoAccumulate",
	stateGoAfterSemi:                "stateGoAfterSemi",
	stateGoAfterLessThan:            "stateGoAfterLessThan",
	stateGoAfterQuotient:            "stateGoAfterQuotient",
	stateGoAfterIdent:               "stateGoAfterIdent",
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

// mode determines which underlying tokenizer (HTML, HTML attribute, or Go) to
// get the next internal token from
type mode int

const (
	modeHTML mode = iota
	modeHTMLAttr
	modeGo
)

var modes = [...]string{
	modeHTML:     "modeHTML",
	modeHTMLAttr: "modeHTMLAttr",
	modeGo:       "modeGo",
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

var stateModeMap = [...]mode{
	stateHTML:                       modeHTML,
	stateHTMLStartOrSelfCloseTag:    modeHTML,
	stateHTMLAttr:                   modeHTMLAttr,
	stateHTMLAttrName:               modeHTMLAttr,
	stateHTMLAttrNameGoExpr:         modeGo,
	stateHTMLAttrValue:              modeHTMLAttr,
	stateHTMLAttrValueGoExpr:        modeGo,
	stateHTMLStartOrSelfCloseTagEnd: modeHTML,
	stateGoStart:                    modeGo,
	stateGoIf:                       modeGo,
	stateGoFor:                      modeGo,
	stateGoPartial:                  modeGo,
	stateGoParam:                    modeGo,
	stateGoImplicitExpr:             modeGo,
	stateGoExplicitExpr:             modeGo,
	stateGoAccumulate:               modeGo,
	stateGoAfterSemi:                modeGo,
	stateGoAfterLessThan:            modeGo,
	stateGoAfterQuotient:            modeGo,
	stateGoAfterIdent:               modeGo,
}

func New(source []byte) *Lexer {
	l := &Lexer{input: source, state: stateHTML}
	l.sync()
	return l
}

func (l *Lexer) Tokens() iter.Seq[Token] {
	return func(yield func(Token) bool) {
		for {
			token := l.Next()
			if !yield(token) || IsEOF(token) {
				break
			}
		}
	}
}

func (l *Lexer) sync() {
	mode := stateModeMap[l.state]
	slog.Info("sync", "mode", mode)
	switch mode {
	case modeHTML:
		l.syncHTMLTokenizer()
	case modeHTMLAttr:
		// t := l.current.(htmlToken)
		// if t.tok != html.StartTagToken && t.tok != html.SelfClosingTagToken {
		// 	panic(fmt.Sprintf("unexpected HTML token type %v", t.tok))
		// }
		// attrs, err := scanAttrs(string(t.raw), l.pos)
		// if err != nil {
		// 	// TODO: call error method on lexer
		// 	panic(fmt.Sprintf("scanning HTML attributes: %v", err))
		// }
		// l.hattrCursor = newAttrCursor(attrs)
	case modeGo:
		l.syncGoScanner()
	}
}

func (l *Lexer) syncHTMLTokenizer() {
	slog.Info("sync HTML tokenizer", "src", window(string(l.input), l.pos, 5))
	l.htmlTokenizer = html.NewTokenizer(bytes.NewReader(l.input[l.pos:]))
}

func (l *Lexer) syncGoScanner() {
	slog.Info("sync go scanner")
	fset := token.NewFileSet()
	file := fset.AddFile("", l.pos, len(l.input[l.pos:]))
	var scan scanner.Scanner
	scan.Init(file, l.input[l.pos:], nil, scanner.ScanComments)
	l.goScanner = &scan
}

func (l *Lexer) pushBuffer(t Token) {
	l.tmp = append(l.tmp, t)
}

func (l *Lexer) clearBuffer() {
	l.tmp = l.tmp[:0]
}

func (l *Lexer) emitBuffer() {
	l.emitted = append(l.emitted, l.tmp...)
	l.clearBuffer()
}

func (l *Lexer) rewindToStartOfBuffer() {
	l.pos = l.tmp[0].Pos()
}

func (l *Lexer) Next() Token {
	if len(l.emitted) == 0 {
		l.run()
	}
	t := l.emitted[0]
	l.emitted = l.emitted[1:]
	return t
}

func (l *Lexer) emit(t Token) {
	slog.Info("emit", "token", t)
	l.emitted = append(l.emitted, t)
}

func (l *Lexer) switchState(new state) {
	mode := stateModeMap[l.state]
	slog.Info("switch state", "mode", mode, "current", l.state, "new", new)
	nextMode := stateModeMap[new]
	l.state = new
	if mode != nextMode {
		slog.Info("switch mode", "new", nextMode)
		l.sync()
	}
}

func (l *Lexer) nextHtmlToken() (t htmlToken) {
	z := l.htmlTokenizer
	t.tok = z.Next()
	raw := z.Raw()
	t.raw = make([]byte, len(raw))
	copy(t.raw, raw)
	l.width = len(raw)
	t.err = z.Err()
	if t.tok == html.StartTagToken || t.tok == html.SelfClosingTagToken {
		t.tagName, t.hasAttrs = z.TagName()
		if t.hasAttrs {
			attrs, err := scanAttrs(string(t.raw), l.pos)
			if err != nil {
				panic(fmt.Sprintf("scanning HTML attributes: %v", err))
			}
			l.attrs = attrs
		}
	}
	t.pos = l.pos
	// This is the whole tag (if start/self-close), but position of attribute
	// names and values will be carried by them along as they are consumed
	l.pos += l.width
	l.current = t
	l.currentHtmlToken = t
	return
}

func (l *Lexer) nextHtmlAttribute() *ast.Attr { // return type
	if len(l.attrs) > 0 {
		a := l.attrs[0]
		l.attrs = l.attrs[1:]
		l.current = a
		l.currentAttr = a
		return a
	}
	return nil
}

func (l *Lexer) nextGoToken() (t goToken) {
	var lit string
	t.pos, t.tok, lit = l.goScanner.Scan()
	if t.tok.IsLiteral() ||
		t.tok.IsKeyword() ||
		t.tok == token.SEMICOLON ||
		t.tok == token.COMMENT ||
		t.tok == token.ILLEGAL {
		t.lit = lit
	} else {
		t.lit = t.tok.String()
	}
	l.width = len(t.lit)
	l.pos = int(t.pos) + l.width
	l.current = t
	l.currentGoToken = t
	slog.Info("next go token", "token", t)
	return
}

func (l *Lexer) next() any { // return type???
	switch mode := stateModeMap[l.state]; mode {
	case modeHTML:
		return l.nextHtmlToken()
	case modeHTMLAttr:
		return l.nextHtmlAttribute()
	case modeGo:
		return l.nextGoToken()
	default:
		panic(fmt.Sprintf("unexpected mode: %v", mode))
	}
}

func (l *Lexer) backup() {
	l.pos -= l.width
	switch mode := stateModeMap[l.state]; mode {
	case modeGo:
		l.syncGoScanner()
	case modeHTML:
		l.syncHTMLTokenizer()
	}
}

func (l *Lexer) backupForTransition() {
	l.pos -= l.width
}

func (l *Lexer) ignore() {
	l.start = l.pos
}

func (l *Lexer) peek() any {
	token := l.next()
	l.backup()
	return token
}

func (l *Lexer) run() {
	for {
		slog.Info("run", "current", l.state, "start", l.start, "pos", l.pos, "window(5)", window(string(l.input), l.pos, 5))
		switch l.state {
		case stateHTML:
			// FIXME: runtime type assertion - use type-safe accessor?
			t := l.next().(htmlToken)
			slog.Info("tokenize html", "tok", t.tok, "raw", string(t.raw))
			switch t.tok {
			case html.ErrorToken:
				if errors.Is(t.err, io.EOF) {
					// TODO: wrap the emit EOF and emit Transition
					l.emit(EOF(l.start))
				}
				slog.Error("tokenizing HTML", "error", t.err)
				l.emit(EOF(l.start))
				return

			case html.TextToken:
				idx := bytes.IndexRune(t.raw, '^')
				// No transition
				if idx == -1 {
					l.emit(l.makeHTMLToken())
					return
					// Transition but emit leading text first
				} else if idx > 0 {
					l.pos = l.start + idx
					l.switchState(stateGoStart)
					l.emit(l.makeHTMLToken())
					return
					// Transition immediately
				} else {
					l.backupForTransition() // reset position to start of text token
					l.switchState(stateGoStart)
				}

			case html.StartTagToken, html.SelfClosingTagToken:
				lit := append([]byte("<"), t.tagName...)
				result := HTMLToken{
					Type: HTMLTagOpen,
					lit:  lit,
					pos:  l.start,
				}
				l.emit(result)
				l.ignore()
				if t.tok == html.SelfClosingTagToken {
					l.flag |= flagSelfClosing
				}
				l.switchState(stateHTMLAttr)
				return

			case html.EndTagToken:
				l.emit(l.makeHTMLToken())
				return

			case html.CommentToken, html.DoctypeToken:
				l.emit(l.makeHTMLToken())
				return

			default:
				panic(fmt.Sprintf("unexpected HTML token type %v", t.tok))
			}

		case stateHTMLAttr:
			attr := l.next().(*ast.Attr)
			slog.Info("next attr", "attr", attr)
			if attr == nil {
				l.switchState(stateHTMLStartOrSelfCloseTagEnd)
			} else {
				l.switchState(stateHTMLAttrName)
			}

		case stateHTMLStartOrSelfCloseTagEnd:
			lit := ">"
			if l.flag&flagSelfClosing > 0 {
				l.flag &^= flagSelfClosing
				lit = "/>"
			}
			l.emit(HTMLToken{
				Type: HTMLTagClose,
				lit:  []byte(lit),
			})
			curr := l.currentHtmlToken
			// TODO: synchronizing the state of the HTML tokenizer after
			// attribute processing should probably live in the state
			// transition function
			l.pos = curr.pos + len(curr.raw)
			l.ignore()
			l.syncHTMLTokenizer()
			l.switchState(stateHTML)
			return

		case stateHTMLAttrName:
			attr := l.current.(*ast.Attr)
			name := attr.Name.Text
			idx := strings.IndexRune(name, '^')
			// No transition
			if idx == -1 {
				l.emit(l.makeAttrToken(name, int(attr.Name.Start)))
				l.switchState(stateHTMLAttrValue)
				return
				// Transition but emit leading text first
			} else if idx > 0 {
				l.emit(l.makeAttrToken(name[:idx], int(attr.Name.Start)))
				l.switchState(stateHTMLAttrNameGoExpr)
				return
				// Transition immediately
			} else {
				l.switchState(stateHTMLAttrNameGoExpr)
			}

		case stateHTMLAttrValue:
			attr := l.current.(*ast.Attr)
			value := attr.Value
			text := value.Text
			start := int(value.Start)
			idx := strings.IndexRune(text, '^')
			// No transition
			if idx == -1 {
				l.emit(l.makeAttrToken(text, start))
				l.switchState(stateHTMLAttr)
				return
				// Transition but emit leading text first
			} else if idx > 0 {
				l.emit(l.makeAttrToken(text[:idx], start))
				l.pos = start + idx + 1
				l.switchState(stateHTMLAttrValueGoExpr)
				return
				// Transition immediately
			} else {
				l.pos = start + 1
				l.switchState(stateHTMLAttrValueGoExpr)
			}

		case stateHTMLAttrNameGoExpr:
			// TODO: allow explicit expression - peek next token for ( or ident
			l.parseImplicitExpr()
			l.switchState(stateHTMLAttrValue)
			return

		case stateHTMLAttrValueGoExpr:
			// TODO: allow explicit expression - peek next token for ( or ident
			l.parseImplicitExpr()
			l.switchState(stateHTMLAttr)
			return

		case stateGoStart:
			t := l.next().(goToken)
			// FIXME: candidate for accept() method
			if t.tok != token.XOR {
				panic(fmt.Sprintf("expected '^', got %v", t.tok))
			}
			l.emit(l.makeTransitionToken())

			// TODO: peek instead
			t = l.next().(goToken)

			switch t.tok {
			case token.FOR:
				l.emit(l.makeGoToken())
				l.switchState(stateGoFor)
				return
			case token.IF:
				l.switchState(stateGoIf)
			case token.IMPORT:
				l.switchState(stateGoImport)
			case token.IDENT:
				switch t.lit {
				case "param":
					l.switchState(stateGoParam)
				case "partial":
					l.switchState(stateGoPartial)
				default:
					l.backup()
					l.switchState(stateGoImplicitExpr)
				}
			case token.LPAREN:
				l.backup()
				l.switchState(stateGoExplicitExpr)
				// TODO: { for if keywords ...
			case token.LBRACE:
				l.emit(l.makeGoToken())
				l.switchState(stateGoAccumulate)
				// TODO: { for if keywords ...
			default:
				panic(fmt.Sprintf("unexpected transition to Go token: %s %q", t.tok, t.lit))
			}

		case stateGoFor:
			// TODO: consume the for token here
			l.switchState(stateGoAccumulate)

		case stateGoIf:
			panic("if")

		case stateGoImport:
			panic("import")

		case stateGoImplicitExpr:
			l.parseImplicitExpr()
			l.switchState(stateHTML)
			return

		case stateGoExplicitExpr:
			l.parseExplicitExpr()
			l.switchState(stateHTML)
			return

		case stateGoAccumulate:
			t := l.next().(goToken)
			slog.Info("go token", "pos", t.pos, "tok", t.tok, "lit", t.lit)

			switch t.tok {
			case token.EOF:
				l.emit(EOF(l.start))
				return
			case token.LBRACE:
				l.emit(l.makeGoToken())
				l.switchState(stateGoAfterSemi)
			case token.SEMICOLON:
				l.emit(l.makeGoToken())
				l.switchState(stateGoAfterSemi)
			default:
				l.emit(l.makeGoToken())
			}

		case stateGoAfterSemi:
			t := l.next().(goToken)
			slog.Info("go token", "pos", t.pos, "tok", t.tok, "lit", t.lit)

			switch t.tok {
			case token.EOF:
				l.emit(EOF(l.pos))
				return
			case token.LSS:
				l.pushBuffer(l.makeGoToken())
				l.switchState(stateGoAfterLessThan)
			default:
				l.emit(l.makeGoToken())
				l.switchState(stateGoAccumulate)
			}

		case stateGoAfterLessThan:
			t := l.next().(goToken)
			slog.Info("go token", "pos", t.pos, "tok", t.tok, "lit", t.lit)

			if t.tok == token.EOF {
				l.emit(EOF(l.pos))
				return
			}

			switch t.tok {
			case token.EOF:
				l.emit(EOF(l.start))
				return
			case token.IDENT:
				slog.Info("transition Go->HTML")
				l.rewindToStartOfBuffer()
				l.clearBuffer()
				l.ignore()
				l.switchState(stateHTML)
			default:
				l.emitBuffer()
				l.emit(l.makeGoToken())
				l.switchState(stateGoAccumulate)
			}

		default:
			panic(fmt.Sprintf("unexpected state %v", l.state))
		}
	}
}

func (l *Lexer) parseImplicitExpr() {
	if !l.acceptAndEmitGo(token.IDENT) {
		panic("expected Go IDENT")
	}

	if l.nextCharIsWhitespace() {
		l.emit(l.makeGoToken())
		return
	}

	for {
		t := l.next().(goToken)

		switch t.tok {
		case token.PERIOD:
			t = l.next().(goToken)
			if !l.acceptAndEmitGo(token.IDENT) {
				panic("expected Go IDENT after dot")
			}

			if l.nextCharIsWhitespace() {
				l.emit(l.makeGoToken())
				return
			}

		case token.LPAREN:
			for !l.acceptAndEmitGo(token.RPAREN) {
			}
			return

		case token.LBRACK:
			for !l.acceptAndEmitGo(token.RBRACK) {
			}
			return

		default:
			l.backupForTransition()
			return
		}
	}
}

func (l *Lexer) parseExplicitExpr() {
	if !l.acceptAndEmitGo(token.LPAREN) {
		panic("expected Go LPAREN")
	}

	nested := 1

	for {
		t := l.next().(goToken)

		switch t.tok {
		case token.LPAREN:
			nested++

		case token.RPAREN:
			nested--

		case token.EOF:
			return
		}
		l.emit(l.makeGoToken())
		if nested == 0 {
			l.backupForTransition()
			return
		}
	}
}

func (l *Lexer) acceptAndEmitGo(tt token.Token) bool {
	if l.next().(goToken).tok == tt {
		l.emit(l.makeGoToken())
		return true
	}
	l.backup()
	return false
}

func (l *Lexer) makeAttrToken(text string, pos int) AttrToken {
	return AttrToken{
		lit: []byte(text),
		pos: pos,
	}
}

func (l *Lexer) nextCharIsWhitespace() bool {
	ch := l.input[l.pos]
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func (l *Lexer) makeHTMLToken() HTMLToken {
	t := newHTMLToken(l.current.(htmlToken).tok, l.input[l.start:l.pos], l.pos)
	l.start = l.pos
	return t
}

func (l *Lexer) makeGoToken() GoToken {
	gt := l.current.(goToken)
	t := GoToken{Type: gt.tok, lit: []byte(gt.lit), pos: l.start}
	l.start = l.pos
	return t
}

func (l *Lexer) makeTransitionToken() Transition {
	t := Transition(l.start)
	l.start = l.pos
	return t
}

type goToken struct {
	pos token.Pos // starting character position of the Go token
	tok token.Token
	lit string
}

type htmlToken struct {
	tok      html.TokenType // z.Next()
	raw      []byte         // z.Raw()
	err      error          // z.Err()
	tagName  []byte         // z.TagName()
	hasAttrs bool
	pos      int
}

func (t htmlToken) Pos() int {
	return t.pos
}

func (t htmlToken) Lit() string {
	return string(t.raw)
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

// window returns a window of n characters around the given position in the input string.
// If the string is shorter than n, it returns the entire string.
// For positions near the start or end, it returns the first or last n characters respectively.
func window(input string, position int, n int) string {
	// Handle invalid inputs
	if n <= 0 {
		return ""
	}
	if len(input) == 0 {
		return ""
	}
	if position < 0 {
		position = 0
	}
	if position >= len(input) {
		position = len(input) - 1
	}

	// If string is shorter than window size, return entire string
	if len(input) <= n {
		return input
	}

	// Calculate window boundaries
	halfWindow := n / 2
	start := position - halfWindow
	end := start + n

	// Adjust for start of string
	if start < 0 {
		start = 0
		end = n
	}

	// Adjust for end of string
	if end > len(input) {
		end = len(input)
		start = end - n
	}

	return input[start:end]
}
