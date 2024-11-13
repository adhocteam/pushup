package lexer

import (
	"bytes"
	"fmt"
	"iter"
	"unicode/utf8"

	"github.com/adhocteam/pushup/internal/source"
)

type Lexer struct {
	source  []byte
	mode    mode
	start   int
	pos     int
	current rune
}

func New(source []byte) *Lexer {
	l := &Lexer{
		source: source,
		mode:   modeHTML,
	}
	l.advance()
	return l
}

func (l *Lexer) advance() {
	var size int
	l.current, size = utf8.DecodeRune(l.source[l.pos:])
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

func (l *Lexer) next() Token {
	switch l.mode {
	case modeHTML:
		if l.current == utf8.RuneError {
			return l.emit(ILLEGAL)
		}

		switch l.current {
		case ' ', '\t', '\n':
			for {
				if !isWhitespace(l.current) {
					l.backup()
					break
				}
				l.advance()
			}
			return l.emit(WHITESPACE)
		case '^':
			l.advance()
			if l.current == '^' {
				// TODO: handle escaped ^^
			}
			return l.transition()
		default:
			panic(fmt.Sprintf("unhandled rune: %q", l.current))
		}

	case modeGo:
	}

	return l.emit(EOF)
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
	l.mode = modeGo

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

type mode int

const (
	modeHTML mode = iota
	modeGo
)
