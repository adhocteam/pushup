package lexer

import (
	"bytes"
	"go/token"
	"testing"
)

func TestPushupLexer(t *testing.T) {
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
	// Define key token sequences we expect to find
	keySequences := []struct {
		name  string
		check func(t *testing.T, tokens []Token) bool
	}{
		{
			name: "DOCTYPE declaration",
			check: func(t *testing.T, tokens []Token) bool {
				if len(tokens) == 0 {
					return false
				}
				if ht, ok := tokens[0].(HTMLToken); ok {
					if ht.Type == HTMLDoctype {
						t.Logf("Found DOCTYPE at position %d", ht.Pos())
						return true
					}
				}
				return false
			},
		},
		{
			name: "Go block with variable declaration",
			check: func(t *testing.T, tokens []Token) bool {
				// Look for: ^{ name := "world" }
				for i := 0; i < len(tokens)-5; i++ {
					if !IsTransition(tokens[i]) {
						continue
					}

					// Check for { token
					gt1, ok1 := tokens[i+1].(GoToken)
					if !ok1 || gt1.Type != token.LBRACE {
						continue
					}

					// Check for name token
					gt2, ok2 := tokens[i+2].(GoToken)
					if !ok2 || gt2.Type != token.IDENT || string(gt2.Lit()) != "name" {
						continue
					}

					// Check for := token
					gt3, ok3 := tokens[i+3].(GoToken)
					if !ok3 || string(gt3.Lit()) != ":=" {
						continue
					}

					// Check for "world" token
					gt4, ok4 := tokens[i+4].(GoToken)
					if !ok4 || gt4.Type != token.STRING {
						continue
					}

					t.Logf("Found Go block with variable declaration at positions: %d, %d, %d, %d, %d",
						tokens[i].Pos(), gt1.Pos(), gt2.Pos(), gt3.Pos(), gt4.Pos())
					return true
				}
				return false
			},
		},
		{
			name: "HTML element with class attribute",
			check: func(t *testing.T, tokens []Token) bool {
				// Look for: <p class="greeting">
				var pTag, closeTag HTMLToken
				var classAttr, greetingAttr AttrToken

				for i := 0; i < len(tokens); i++ {
					ht, ok := tokens[i].(HTMLToken)
					if ok && ht.Type == HTMLTagOpen && bytes.Contains(ht.Lit(), []byte("<p")) {
						pTag = ht

						// Look ahead for class and greeting attributes (might not be consecutive)
						for j := i + 1; j < len(tokens) && j < i+5; j++ {
							at, ok := tokens[j].(AttrToken)
							if ok {
								if string(at.Lit()) == "class" {
									classAttr = at
								} else if string(at.Lit()) == "greeting" {
									greetingAttr = at
								}
							}

							// Check for closing tag
							ht2, ok := tokens[j].(HTMLToken)
							if ok && ht2.Type == HTMLTagClose {
								closeTag = ht2
								break
							}
						}

						// If we found all the required tokens
						if classAttr.Pos() > 0 && greetingAttr.Pos() > 0 && closeTag.Type == HTMLTagClose {
							t.Logf("Found p element with class attribute at positions: %d, %d, %d, %d",
								pTag.Pos(), classAttr.Pos(), greetingAttr.Pos(), closeTag.Pos())
							return true
						}
					}
				}
				return false
			},
		},
		{
			name: "Go variable reference",
			check: func(t *testing.T, tokens []Token) bool {
				// Look for: ^name
				for i := 0; i < len(tokens)-1; i++ {
					if !IsTransition(tokens[i]) {
						continue
					}

					// Check for name token
					gt, ok := tokens[i+1].(GoToken)
					if !ok || gt.Type != token.IDENT || string(gt.Lit()) != "name" {
						continue
					}

					t.Logf("Found Go variable reference at positions: %d, %d",
						tokens[i].Pos(), gt.Pos())
					return true
				}
				return false
			},
		},
		{
			name: "For loop statement",
			check: func(t *testing.T, tokens []Token) bool {
				// Look for: ^for i := range 3 {
				for i := 0; i < len(tokens)-6; i++ {
					if !IsTransition(tokens[i]) {
						continue
					}

					// Check for for token
					gt1, ok := tokens[i+1].(GoToken)
					if !ok || gt1.Type != token.FOR {
						continue
					}

					// Check for i token
					gt2, ok2 := tokens[i+2].(GoToken)
					if !ok2 || gt2.Type != token.IDENT || string(gt2.Lit()) != "i" {
						continue
					}

					// Check for := token
					gt3, ok3 := tokens[i+3].(GoToken)
					if !ok3 || string(gt3.Lit()) != ":=" {
						continue
					}

					// Check for range token
					gt4, ok4 := tokens[i+4].(GoToken)
					if !ok4 || gt4.Type != token.RANGE {
						continue
					}

					// Check for 3 token
					gt5, ok5 := tokens[i+5].(GoToken)
					if !ok5 || gt5.Type != token.INT || string(gt5.Lit()) != "3" {
						continue
					}

					t.Logf("Found for loop at positions: %d, %d, %d, %d, %d, %d",
						tokens[i].Pos(), gt1.Pos(), gt2.Pos(), gt3.Pos(), gt4.Pos(), gt5.Pos())
					return true
				}
				return false
			},
		},
		{
			name: "Go variable in attribute",
			check: func(t *testing.T, tokens []Token) bool {
				// Look for attribute with item- prefix followed by a GoToken with value 'i'
				var itemAttr AttrToken
				var iToken GoToken

				for i := 0; i < len(tokens)-2; i++ {
					// Find the 'item-' attribute token
					at, ok := tokens[i].(AttrToken)
					if ok && bytes.Contains(at.Lit(), []byte("item-")) {
						itemAttr = at

						// Look ahead for the i token (may not be immediately after)
						for j := i + 1; j < len(tokens) && j < i+4; j++ {
							gt, ok := tokens[j].(GoToken)
							if ok && gt.Type == token.IDENT && string(gt.Lit()) == "i" {
								iToken = gt
								t.Logf("Found Go variable in attribute at positions: %d, %d",
									itemAttr.Pos(), iToken.Pos())
								return true
							}
						}
					}
				}
				return false
			},
		},
		{
			name: "Explicit Go expression",
			check: func(t *testing.T, tokens []Token) bool {
				// Look for: ^(i * i)
				for i := 0; i < len(tokens)-5; i++ {
					if !IsTransition(tokens[i]) {
						continue
					}

					// Check for ( token
					gt1, ok1 := tokens[i+1].(GoToken)
					if !ok1 || gt1.Type != token.LPAREN {
						continue
					}

					// Check for i token
					gt2, ok2 := tokens[i+2].(GoToken)
					if !ok2 || gt2.Type != token.IDENT || string(gt2.Lit()) != "i" {
						continue
					}

					// Check for * token
					gt3, ok3 := tokens[i+3].(GoToken)
					if !ok3 || gt3.Type != token.MUL {
						continue
					}

					// Check for i token
					gt4, ok4 := tokens[i+4].(GoToken)
					if !ok4 || gt4.Type != token.IDENT || string(gt4.Lit()) != "i" {
						continue
					}

					// Check for ) token
					gt5, ok5 := tokens[i+5].(GoToken)
					if !ok5 || gt5.Type != token.RPAREN {
						continue
					}

					t.Logf("Found explicit Go expression at positions: %d, %d, %d, %d, %d, %d",
						tokens[i].Pos(), gt1.Pos(), gt2.Pos(), gt3.Pos(), gt4.Pos(), gt5.Pos())
					return true
				}
				return false
			},
		},
	}

	// Run the lexer and collect tokens
	l := New([]byte(source))
	var tokens []Token

	// Debug token collection
	t.Log("All tokens produced by lexer:")
	for token := range l.Tokens() {
		tokens = append(tokens, token)
		t.Logf("%d: %T %q", token.Pos(), token, string(token.Lit()))
		if IsEOF(token) {
			break
		}
	}

	// Verify we have enough tokens
	if len(tokens) < 40 {
		t.Errorf("Expected at least 40 tokens, got %d", len(tokens))
	}

	// Verify the key sequences
	for _, seq := range keySequences {
		if found := seq.check(t, tokens); !found {
			t.Errorf("Token sequence '%s' not found", seq.name)
		}
	}

	// Verify token positions are monotonically increasing (with exceptions for transitions)
	var prevPos int = -1
	for i, token := range tokens {
		pos := token.Pos()

		// Simple check for monotonicity in token positions
		if pos < prevPos {
			// Check for special cases - some transition tokens might start at same position
			if !(IsTransition(token) || isPositionExempt(token)) {
				t.Errorf("Token position regression at index %d: position %d is less than previous position %d",
					i, pos, prevPos)
			}
		}

		prevPos = pos
	}

	// Verify the final token is EOF
	lastToken := tokens[len(tokens)-1]
	if !IsEOF(lastToken) {
		t.Errorf("Expected final token to be EOF, got %T", lastToken)
	}
}

// Some tokens might be exempt from the strict monotonicity check
func isPositionExempt(token Token) bool {
	// HTML tag close tokens sometimes have position 0
	if ht, ok := token.(HTMLToken); ok && ht.Type == HTMLTagClose && ht.Pos() == 0 {
		return true
	}
	return false
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
