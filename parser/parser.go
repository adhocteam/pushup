package parser

import (
	"fmt"

	"github.com/adhocteam/pushup/lexer"
)

type Parser struct {
	l *lexer.Lexer
}

func New(source []byte) *Parser {
	p := &Parser{l: lexer.New(source)}
	return p
}

func (p *Parser) Parse() {
	p.parseDocument()
}

func (p *Parser) parseDocument() {
	l := p.l
	for {
		token := l.Next()
		if lexer.IsEOF(token) {
			break
		}
		fmt.Println(token)
	}
}
