package lexer

import (
	"fmt"
	"strconv"
	"testing"
)

func TestLexer(t *testing.T) {
	source := `<!DOCTYPE html>
<ul>
    ^for i := range 3 {
        x := i * i
        <li>^x</li>
    }
</ul>
`
	l := New([]byte(source))
	for token := range l.Tokens() {
		fmt.Println(token.Pos(), strconv.Quote(string(token.Lit())))
	}
}
