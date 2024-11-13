package lexer

import (
	"strconv"

	"github.com/adhocteam/pushup/internal/source"
)

type Token struct {
	tokType TokenType
	literal []byte
	loc     source.Span
}

func (t Token) String() string {
	s := "Token{tokType:" + t.tokType.String() + " literal:" + strconv.Quote(string(t.literal)) + " loc:" + t.loc.String()
	return s
}
