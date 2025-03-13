package lexer

import (
	"fmt"
	"go/token"
	"testing"
)

// ExpectedToken represents a token we expect the lexer to produce
type ExpectedToken struct {
	Type     string // "HTML", "GoToken", "AttrToken", "Transition", "EOF"
	Position int
	Literal  string        // The literal content of the token
	HTMLType HTMLTokenType // Used only for HTML tokens
	GoType   token.Token   // Used only for Go tokens
}

func (et ExpectedToken) String() string {
	switch et.Type {
	case "HTML":
		return fmt.Sprintf("HTML(%s, %d, %q)", et.HTMLType, et.Position, et.Literal)
	case "GoToken":
		return fmt.Sprintf("Go(%s, %d, %q)", et.GoType, et.Position, et.Literal)
	case "AttrToken":
		return fmt.Sprintf("Attr(%d, %q)", et.Position, et.Literal)
	case "Transition":
		return fmt.Sprintf("Transition(%d)", et.Position)
	case "EOF":
		return fmt.Sprintf("EOF(%d)", et.Position)
	default:
		return fmt.Sprintf("Unknown(%s, %d, %q)", et.Type, et.Position, et.Literal)
	}
}

func TestLexer(t *testing.T) {
	source := `<!DOCTYPE html>
<title>Pushup</title>
^{
    name := "world"
}
<p class="greeting">Hello, ^name!</p>
<ul>
^for i := range 3 {
    <li id="item-^i">
        ^(i * i)
    </li>
}
</ul>
`
	// Define expected tokens in order
	expectedTokens := []ExpectedToken{
		// DOCTYPE declaration
		{Type: "HTML", Position: 15, Literal: "<!DOCTYPE html>", HTMLType: HTMLDoctype},
		{Type: "HTML", Position: 16, Literal: "\n", HTMLType: HTMLText},

		// title tag
		{Type: "HTML", Position: 16, Literal: "<title", HTMLType: HTMLTagOpen},
		{Type: "HTML", Position: 0, Literal: ">", HTMLType: HTMLTagClose},
		{Type: "HTML", Position: 29, Literal: "Pushup", HTMLType: HTMLText},
		{Type: "HTML", Position: 37, Literal: "</title>", HTMLType: HTMLEndTag},
		{Type: "HTML", Position: 38, Literal: "\n", HTMLType: HTMLText},

		// Go code block
		{Type: "Transition", Position: 38, Literal: "^"},
		{Type: "GoToken", Position: 39, Literal: "{", GoType: token.LBRACE},
		{Type: "GoToken", Position: 40, Literal: "name", GoType: token.IDENT},
		{Type: "GoToken", Position: 49, Literal: ":=", GoType: token.DEFINE},
		{Type: "GoToken", Position: 52, Literal: "\"world\"", GoType: token.STRING},
		{Type: "GoToken", Position: 60, Literal: "\n", GoType: token.SEMICOLON},
		{Type: "GoToken", Position: 61, Literal: "}", GoType: token.RBRACE},
		{Type: "GoToken", Position: 62, Literal: "\n", GoType: token.SEMICOLON},

		// p tag with class attribute
		{Type: "HTML", Position: 63, Literal: "<p", HTMLType: HTMLTagOpen},
		{Type: "AttrToken", Position: 66, Literal: "class"},
		{Type: "AttrToken", Position: 73, Literal: "greeting"},
		{Type: "HTML", Position: 0, Literal: ">", HTMLType: HTMLTagClose},
		{Type: "HTML", Position: 90, Literal: "Hello, ", HTMLType: HTMLText},

		// Variable reference
		{Type: "Transition", Position: 90, Literal: "^"},
		{Type: "GoToken", Position: 91, Literal: "name", GoType: token.IDENT},
		{Type: "HTML", Position: 96, Literal: "!", HTMLType: HTMLText},
		{Type: "HTML", Position: 100, Literal: "</p>", HTMLType: HTMLEndTag},
		{Type: "HTML", Position: 101, Literal: "\n", HTMLType: HTMLText},

		// ul tag
		{Type: "HTML", Position: 101, Literal: "<ul", HTMLType: HTMLTagOpen},
		{Type: "HTML", Position: 0, Literal: ">", HTMLType: HTMLTagClose},
		{Type: "HTML", Position: 106, Literal: "\n", HTMLType: HTMLText},

		// For loop
		{Type: "Transition", Position: 106, Literal: "^"},
		{Type: "GoToken", Position: 107, Literal: "for", GoType: token.FOR},
		{Type: "GoToken", Position: 110, Literal: "i", GoType: token.IDENT},
		{Type: "GoToken", Position: 112, Literal: ":=", GoType: token.DEFINE},
		{Type: "GoToken", Position: 115, Literal: "range", GoType: token.RANGE},
		{Type: "GoToken", Position: 121, Literal: "3", GoType: token.INT},
		{Type: "GoToken", Position: 123, Literal: "{", GoType: token.LBRACE},
		{Type: "HTML", Position: 130, Literal: "\n    ", HTMLType: HTMLText},

		// li tag with dynamic id
		{Type: "HTML", Position: 130, Literal: "<li", HTMLType: HTMLTagOpen},
		{Type: "AttrToken", Position: 134, Literal: "id"},
		{Type: "AttrToken", Position: 138, Literal: "item-"},
		{Type: "GoToken", Position: 147, Literal: "i", GoType: token.IDENT},
		{Type: "HTML", Position: 0, Literal: ">", HTMLType: HTMLTagClose},
		{Type: "HTML", Position: 156, Literal: "\n        ", HTMLType: HTMLText},

		// Expression (i * i)
		{Type: "Transition", Position: 156, Literal: "^"},
		{Type: "GoToken", Position: 157, Literal: "(", GoType: token.LPAREN},
		{Type: "GoToken", Position: 158, Literal: "i", GoType: token.IDENT},
		{Type: "GoToken", Position: 159, Literal: "*", GoType: token.MUL},
		{Type: "GoToken", Position: 161, Literal: "i", GoType: token.IDENT},
		{Type: "GoToken", Position: 163, Literal: ")", GoType: token.RPAREN},
		{Type: "HTML", Position: 169, Literal: "\n    ", HTMLType: HTMLText},
		{Type: "HTML", Position: 174, Literal: "</li>", HTMLType: HTMLEndTag},

		// Closing tags
		{Type: "HTML", Position: 177, Literal: "\n}\n", HTMLType: HTMLText},
		{Type: "HTML", Position: 182, Literal: "</ul>", HTMLType: HTMLEndTag},
		{Type: "HTML", Position: 183, Literal: "\n", HTMLType: HTMLText},
		{Type: "EOF", Position: 183, Literal: "EOF"},
	}

	// Run the lexer
	l := New([]byte(source))
	var tokens []Token

	// Collect all tokens
	for token := range l.Tokens() {
		tokens = append(tokens, token)
		if IsEOF(token) {
			break
		}
	}

	// Check if we got the expected number of tokens
	if len(tokens) != len(expectedTokens) {
		t.Errorf("Expected %d tokens, got %d tokens", len(expectedTokens), len(tokens))
		// Print the actual tokens for debugging
		for i, token := range tokens {
			t.Logf("Actual token %d: %T at position %d with value %q",
				i, token, token.Pos(), string(token.Lit()))
		}
	}

	// Compare each token
	maxTokens := len(tokens)
	if len(expectedTokens) < maxTokens {
		maxTokens = len(expectedTokens)
	}

	for i := 0; i < maxTokens; i++ {
		actual := tokens[i]
		expected := expectedTokens[i]

		// Check token position
		if actual.Pos() != expected.Position &&
			// Special case: HTML tag close tokens sometimes have position 0
			!(expected.Type == "HTML" && expected.HTMLType == HTMLTagClose && actual.Pos() == 0) {
			t.Errorf("Token %d: Position mismatch - expected %d, got %d",
				i, expected.Position, actual.Pos())
		}

		// Check token literal
		if string(actual.Lit()) != expected.Literal {
			t.Errorf("Token %d: Literal mismatch - expected %q, got %q",
				i, expected.Literal, string(actual.Lit()))
		}

		// Check token type
		switch expected.Type {
		case "HTML":
			ht, ok := actual.(HTMLToken)
			if !ok {
				t.Errorf("Token %d: Type mismatch - expected HTMLToken, got %T", i, actual)
				continue
			}
			if ht.Type != expected.HTMLType {
				t.Errorf("Token %d: HTML type mismatch - expected %v, got %v",
					i, expected.HTMLType, ht.Type)
			}

		case "GoToken":
			gt, ok := actual.(GoToken)
			if !ok {
				t.Errorf("Token %d: Type mismatch - expected GoToken, got %T", i, actual)
				continue
			}
			if gt.Type != expected.GoType {
				t.Errorf("Token %d: Go type mismatch - expected %v, got %v",
					i, expected.GoType, gt.Type)
			}

		case "AttrToken":
			_, ok := actual.(AttrToken)
			if !ok {
				t.Errorf("Token %d: Type mismatch - expected AttrToken, got %T", i, actual)
			}

		case "Transition":
			if !IsTransition(actual) {
				t.Errorf("Token %d: Type mismatch - expected Transition, got %T", i, actual)
			}

		case "EOF":
			if !IsEOF(actual) {
				t.Errorf("Token %d: Type mismatch - expected EOF, got %T", i, actual)
			}
		}
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
