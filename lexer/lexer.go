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

	"golang.org/x/net/html"
)

type Lexer struct {
	source []byte
	start  int
	pos    int
	state  state

	tokenBuffer []Token

	bgs *bufGoScanner
}

type state int

const (
	stateHTML state = iota
	stateGoStart
	stateGoAccumulate
	stateGoAfterSemi
	stateGoAfterLessThan
	stateGoImplicitExpr
	stateGoExplicitExpr
)

func New(source []byte) *Lexer {
	return &Lexer{source: source, state: stateHTML}
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

func (l *Lexer) src() []byte {
	return l.source[l.pos:]
}

func (l *Lexer) Next() Token {
	if len(l.tokenBuffer) == 0 {
		l.fillBuffer()
	}
	t := l.tokenBuffer[0]
	l.tokenBuffer = l.tokenBuffer[1:]
	return t
}

func (l *Lexer) pushBuffer(t Token) {
	slog.Debug("pushBuffer", "token", t)
	l.tokenBuffer = append(l.tokenBuffer, t)
}

func (l *Lexer) switchState(new state) {
	slog.Info("switch state", "current", l.state, "new", new)
	l.state = new
}

func (l *Lexer) fillBuffer() {
	for {
		slog.Info("main switch", "l.state", l.state)
		switch l.state {
		case stateHTML:
			z := html.NewTokenizer(bytes.NewReader(l.src()))
			tok := z.Next()
			raw := z.Raw()
			slog.Info("tokenize html", "tok", tok, "raw", string(raw))
			switch tok {
			case html.ErrorToken:
				err := z.Err()
				if errors.Is(err, io.EOF) {
					l.pushBuffer(EOF(l.pos))
				}
				slog.Error("tokenizing HTML", "error", err)
				l.pushBuffer(EOF(l.pos))
				return
			case html.TextToken:
				idx := bytes.IndexRune(raw, '^')
				if idx == -1 {
					l.pos += len(raw)
					l.pushBuffer(l.makeHTMLToken(tok))
					return
				} else if idx > 0 {
					l.pos += idx
					l.switchState(stateGoStart)
					l.pushBuffer(l.makeHTMLToken(tok))
					return
				} else {
					l.switchState(stateGoStart)
				}
			case html.StartTagToken:
				l.pos += len(raw)
				l.pushBuffer(l.makeHTMLToken(tok))
				return
			case html.EndTagToken:
				l.pos += len(raw)
				l.pushBuffer(l.makeHTMLToken(tok))
				return
			case html.SelfClosingTagToken:
				l.pos += len(raw)
				l.pushBuffer(l.makeHTMLToken(tok))
				return
			case html.CommentToken:
				l.pos += len(raw)
				l.pushBuffer(l.makeHTMLToken(tok))
				return
			case html.DoctypeToken:
				l.pos += len(raw)
				l.pushBuffer(l.makeHTMLToken(tok))
				return
			default:
				panic(fmt.Sprintf("unexpected HTML token type %v", tok))
			}

		case stateGoStart:
			if ch := l.source[l.pos]; ch != '^' {
				panic(fmt.Sprintf("expected '^', got %q", ch))
			}
			l.pos++
			l.start = l.pos
			// TODO move to switchState()
			l.bgs = newBufGoScanner(l.src(), l.pos)
			_, tok, _ := l.bgs.Scan()
			l.bgs.Unscan()
			switch tok {
			case token.IDENT:
				l.switchState(stateGoImplicitExpr)
			case token.LPAREN:
				l.switchState(stateGoExplicitExpr)
				// TODO: { for if keywords ...
			default:
				l.switchState(stateGoAccumulate)
			}

		case stateGoImplicitExpr:
			panic("implicit")

		case stateGoExplicitExpr:
			panic("explicit")

		case stateGoAccumulate:
			pos, tok, lit := l.bgs.Scan()
			slog.Info("go token", "pos", pos, "tok", tok, "lit", lit)

			l.pos = int(pos)

			switch tok {
			case token.EOF:
				l.pushBuffer(EOF(l.pos))
				return
			case token.SEMICOLON:
				l.pushBuffer(l.makeGoToken(tok))
				l.switchState(stateGoAfterSemi)
			default:
				l.pushBuffer(l.makeGoToken(tok))
			}

		case stateGoAfterSemi:
			pos, tok, lit := l.bgs.Scan()
			slog.Info("go token", "pos", pos, "tok", tok, "lit", lit)

			l.pos = int(pos)

			if tok == token.EOF {
				l.pushBuffer(EOF(l.pos))
				return
			}

			switch tok {
			case token.EOF:
				l.pushBuffer(EOF(l.pos))
				return
			case token.LSS:
				l.pushBuffer(l.makeGoToken(tok))
				l.switchState(stateGoAfterLessThan)
			default:
				l.pushBuffer(l.makeGoToken(tok))
				l.switchState(stateGoAccumulate)
			}

		case stateGoAfterLessThan:
			pos, tok, lit := l.bgs.Scan()
			slog.Info("go token", "pos", pos, "tok", tok, "lit", lit)

			l.pos = int(pos)

			switch tok {
			case token.EOF:
				l.pushBuffer(EOF(l.pos))
				return
			case token.IDENT:
				l.pos -= len("<" + lit)
				l.start = l.pos
				l.switchState(stateHTML)
			default:
				l.pushBuffer(l.makeGoToken(tok))
				l.switchState(stateGoAccumulate)
			}

		default:
			panic(fmt.Sprintf("unexpected state %v", l.state))
		}
	}
}

func (l *Lexer) makeHTMLToken(tokType html.TokenType) HTMLToken {
	t := newHTMLToken(tokType, l.source[l.start:l.pos], l.pos)
	l.start = l.pos
	return t
}

func (l *Lexer) makeGoToken(tokType token.Token) GoToken {
	t := GoToken{Type: tokType, lit: l.source[l.start:l.pos], pos: l.pos}
	l.start = l.pos
	return t
}

type goToken struct {
	pos token.Pos
	tok token.Token
	lit string
}

type bufGoScanner struct {
	*scanner.Scanner
	buf  *goToken
	last *goToken
}

func newBufGoScanner(src []byte, baseOffset int) *bufGoScanner {
	scan := new(scanner.Scanner)
	fset := token.NewFileSet()
	file := fset.AddFile("", baseOffset, len(src))
	scan.Init(file, src, nil, scanner.ScanComments)
	return &bufGoScanner{Scanner: scan}
}

func (s *bufGoScanner) empty() bool {
	return s.buf == nil
}

func (s *bufGoScanner) Scan() (pos token.Pos, tok token.Token, lit string) {
	var t goToken
	if s.empty() {
		t.pos, t.tok, t.lit = s.Scanner.Scan()
		s.last = &t
	} else {
		t = *s.buf
		s.buf = nil
	}
	pos = t.pos
	tok = t.tok
	lit = t.lit
	return
}

func (s *bufGoScanner) Unscan() {
	if s.empty() && s.last != nil {
		s.buf = s.last
	} else {
		panic("unscan() before call to scan()")
	}
}
