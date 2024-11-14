package lexer

import (
	"bytes"
	"go/scanner"
	"go/token"
	"iter"
	"log/slog"
	"unicode/utf8"

	"github.com/adhocteam/pushup/internal/source"
	"golang.org/x/net/html"
)

type Lexer struct {
	source  []byte
	state   state
	start   int
	pos     int
	current rune
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
	slog.Info("next()", "state", l.state)
	switch l.state {
	case stateHTML:
		z := html.NewTokenizer(bytes.NewReader(l.src()))
		token := z.Next()
		raw := z.Raw()
		slog.Info("HTML token", "token", token, "raw", raw)
		switch token {
		case html.ErrorToken:
			return l.emit(EOF)
		case html.TextToken:
			idx := bytes.IndexRune(raw, '^')
			slog.Info("text", "idx", idx)
			if idx == -1 {
				l.pos += len(raw)
				return l.emit(HTML_TEXT)
			}
			l.pos += idx + 1 // skip past '^'
			l.switchState(stateGo)
			return l.next()
		case html.StartTagToken:
		case html.SelfClosingTagToken:
		case html.EndTagToken:
		case html.CommentToken:
			l.pos += len(raw)
			return l.emit(HTML_TEXT)
		case html.DoctypeToken:
			l.pos += len(raw)
			return l.emit(HTML_TEXT)
		}
		// TODO: push multiple tokens on stack to emit
		l.pos += len(raw)
		return l.emit(LT)

	case stateGo:
		var s scanner.Scanner
		fset := token.NewFileSet()
		file := fset.AddFile("", l.pos, len(l.src()))
		s.Init(file, l.src(), nil, scanner.ScanComments)
		for {
			pos, tok, lit := s.Scan()
			// TODO: next two lines are a hack
			l.pos = int(pos)
			l.advance()
			slog.Info("go scanner", "l.pos", l.pos, "pos", pos, "tok", tok, "lit", lit, "file.Offset(pos)", file.Offset(pos), "fset.Position(pos)", fset.Position(pos))
			switch tok {
			case token.EOF:
				break
			case token.LBRACE:
				l.advance()
				l.switchState(stateHTML)
				return l.emit(GO_EXPR)
			}
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
	stateGo
)

func (s state) String() string {
	switch s {
	case stateHTML:
		return "stateHTML"
	case stateGo:
		return "stateGo"
	default:
		panic("")
	}
}
