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

// Start returns the offset location in the source code (.up file) of the token
// from the beginning of the file.
func (t Token) Start() int {
	return t.loc.Start
}

// Len returns the length of matched literal token from the source code.
func (t Token) Len() int {
	return len(t.literal)
}
