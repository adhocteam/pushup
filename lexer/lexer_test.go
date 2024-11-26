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

func TestWindow(t *testing.T) {
	tests := []struct {
		input    string
		position int
		n        int
		want     string
	}{
		{"Hello, World!", 0, 5, "Hello"},          // Start of string
		{"Hello, World!", 6, 5, "o, Wo"},          // Middle of string
		{"Hello, World!", 12, 5, "orld!"},         // End of string
		{"Hi", 0, 5, "Hi"},                        // String shorter than window
		{"Hello, World!", 7, 1, "W"},              // Window size 1
		{"", 0, 5, ""},                            // Empty string
		{"Hello, World!", -1, 5, "Hello"},         // Invalid position (negative)
		{"Hello, World!", 100, 5, "orld!"},        // Invalid position (too large)
		{"Hello, World!", 6, 0, ""},               // Invalid window size
		{"Hello, World!", 6, 13, "Hello, World!"}, // Window size equals string length
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			result := window(tc.input, tc.position, tc.n)
			if result != tc.want {
				t.Errorf("want %q, got %q", tc.want, result)
			}
		})
	}
}
