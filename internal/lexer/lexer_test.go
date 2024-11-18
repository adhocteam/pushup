package lexer

import "testing"

func TestLexer(t *testing.T) {
	src := []byte(`^if x == 1 {
    <p style="color: red" class=^class>
        hello
    </p>
}
`)
	l := New(src)
	for token := range l.Scan() {
		t.Logf("token: %v", token)
	}
}
