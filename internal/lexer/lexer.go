package lexer

import (
	"bytes"
	"errors"
	"go/scanner"
	"go/token"
	"io"
	"iter"
	"log/slog"
	"strconv"
	"unicode/utf8"

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/source"
	"golang.org/x/net/html"
)

type Lexer struct {
	source []byte
	state  state

	start   int
	pos     int
	current rune

	// html state
	hz     *html.Tokenizer
	htok   html.TokenType
	hraw   []byte
	hattrs []*ast.Attr
	hatidx int

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
	slog.Info("src", "slice", l.source[l.pos:], "pos", l.pos)
	return l.source[l.pos:]
}

func (l *Lexer) next() Token {
	for {
		slog.Info("next()", "state", l.state)

		switch l.state {
		case stateHTML:
			l.hz = html.NewTokenizer(bytes.NewReader(l.src()))
			l.htok = l.hz.Next()
			l.hraw = l.hz.Raw()
			slog.Info("HTML token", "token", l.htok, "raw", l.hraw)

			switch l.htok {
			case html.ErrorToken:
				err := l.hz.Err()
				if errors.Is(err, io.EOF) {
					return l.emit(EOF)
				}
				slog.Error("HTML tokenizer", "error", err)
			case html.TextToken:
				idx := bytes.IndexRune(l.hraw, '^')
				slog.Info("text", "idx", idx)
				if idx == -1 {
					l.pos += len(l.hraw)
					return l.emit(HTML_TEXT)
				}
				l.pos += idx
				token := l.emit(HTML_TEXT) // emit text preceding the transition
				l.pos += 1                 // skip past '^'
				l.start = l.pos
				l.switchState(stateGo)
				return token
			case html.StartTagToken, html.SelfClosingTagToken:
				tagName, hasAttrs := l.hz.TagName()
				n := len("<" + string(tagName))
				l.pos += n
				if hasAttrs {
					// TODO: need to fix up the source position values of the attr
					// names and values - they come back from scanAttrs() relative to
					// the raw tag string, not the overall source
					var err error
					l.hatidx = 0
					l.hattrs, err = scanAttrs(string(l.hraw))
					if err != nil {
						slog.Error("scanAttrs", "error", err)
						return l.emit(ILLEGAL)
					}
					l.switchState(stateHTMLAttr)
				} else {
					l.switchState(stateHTMLAfterLastAttr)
				}
				l.hraw = l.hraw[n:]
				return l.emit(HTML_START_TAG_NAME)
			case html.EndTagToken:
				l.pos += len(l.hraw)
				return l.emit(HTML_END_TAG)
			case html.CommentToken:
				l.pos += len(l.hraw)
				return l.emit(HTML_TEXT)
			case html.DoctypeToken:
				l.pos += len(l.hraw)
				return l.emit(HTML_TEXT)
			}

			l.pos += len(l.hraw)
			return l.emit(LT)

		case stateHTMLAttr:
			if l.hatidx < len(l.hattrs)-1 {
				attr := l.hattrs[l.hatidx]
				l.hatidx++
				slog.Info("attrs", "idx", l.hatidx, "name", attr.Name, "value", attr.Value)
			}

		case stateHTMLAfterLastAttr:
			for l.hraw[0] == ' ' || l.hraw[0] == '\n' || l.hraw[0] == '\t' {
				l.hraw = l.hraw[1:]
				l.pos++
			}
			if bytes.Equal(l.hraw, []byte(">")) {
				l.pos += 1
				l.switchState(stateHTML)
				return l.emit(HTML_GT)
			} else if bytes.Equal(l.hraw, []byte("/>")) {
				l.pos += 2
				l.switchState(stateHTML)
				return l.emit(HTML_SELF_CLOSING_GT)
			} else {
				slog.Info("remaining", "l.hraw", l.hraw)
			}

		case stateGo:
			l.gscanner = new(scanner.Scanner)
			l.gfset = token.NewFileSet()
			l.gfile = l.gfset.AddFile("", l.pos, len(l.src()))
			l.gscanner.Init(l.gfile, l.src(), nil, scanner.ScanComments)
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
			}

		case stateGoCondExpr:
			pos, tok, lit := l.gscanner.Scan()
			for tok != token.LBRACE {
				l.pos = int(pos)
				pos, tok, lit = l.gscanner.Scan()
				slog.Info("stateGoCondExpr", "pos", pos, "tok", tok, "lit", lit)
			}
			l.pos = int(pos)
			token := l.emit(GO_EXPR)
			l.pos++ // skip past the {
			l.start = l.pos
			l.switchState(stateHTML)
			return token
		}
	}

	return l.emit(EOF)
}

func (l *Lexer) switchState(s state) {
	slog.Info("switch state", "exiting", l.state, "entering", s)
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
	return ch == ' ' || ch == '\t' || ch == '\n'
}

type state int

const (
	stateHTML state = iota
	stateHTMLAttr
	stateHTMLAfterLastAttr
	stateGo
	stateGoCondExpr
)

var states = [...]string{
	stateHTML:              "stateHTML",
	stateHTMLAttr:          "stateHTMLAttr",
	stateHTMLAfterLastAttr: "stateHTMLAfterLastAttr",
	stateGo:                "stateGo",
	stateGoCondExpr:        "stateGoCondExpr",
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
