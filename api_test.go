package pushup

import (
	"net/http/httptest"
	"testing"
)

func TestResponseWriter_DiscardCapability(t *testing.T) {
	tests := []struct {
		name           string
		writeSequence  []string
		toggleSequence []bool // true = SetDiscard, false = UnsetDiscard
		expected       string
	}{
		{
			name:           "No discard",
			writeSequence:  []string{"hello", " world"},
			toggleSequence: nil,
			expected:       "hello world",
		},
		{
			name:           "Full discard",
			writeSequence:  []string{"hello", " world"},
			toggleSequence: []bool{true},
			expected:       "",
		},
		{
			name:           "Partial discard then write",
			writeSequence:  []string{"hello", " world", "!"},
			toggleSequence: []bool{true, false},
			expected:       " world!",
		},
		{
			name:           "Write then discard then write",
			writeSequence:  []string{"hello", " world", "!"},
			toggleSequence: []bool{false, true, false},
			expected:       "hello!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			w := NewResponseWriter(rec)

			toggleIdx := 0
			for i, data := range tt.writeSequence {
				// Apply any toggles that should occur before this write
				if tt.toggleSequence != nil && toggleIdx < len(tt.toggleSequence) {
					if tt.toggleSequence[toggleIdx] {
						w.SetDiscard()
					} else {
						w.UnsetDiscard()
					}
					toggleIdx++
				}

				n, err := w.Write([]byte(data))
				if err != nil {
					t.Errorf("unexpected error writing data: %v", err)
				}
				if n != len(data) {
					t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
				}

				if i < len(tt.writeSequence)-1 {
					continue
				}

				w.Flush()
			}

			result := rec.Body.String()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestResponseWriter_SetUnsetDiscard(t *testing.T) {
	rec := httptest.NewRecorder()
	w := NewResponseWriter(rec)

	// Initial state should not be discarding
	w.Write([]byte("first"))
	w.Flush()
	if rec.Body.String() != "first" {
		t.Errorf("expected 'first', got %q", rec.Body.String())
	}

	w.SetDiscard()
	w.Write([]byte("second"))
	w.Flush()
	if rec.Body.String() != "first" {
		t.Errorf("expected 'first', got %q", rec.Body.String())
	}

	w.UnsetDiscard()
	w.Write([]byte("third"))
	w.Flush()
	if rec.Body.String() != "firstthird" {
		t.Errorf("expected 'firstthird', got %q", rec.Body.String())
	}
}

func TestResponseWriter_DiscardReturnsCorrectLength(t *testing.T) {
	rec := httptest.NewRecorder()
	w := NewResponseWriter(rec)

	w.SetDiscard()
	testData := []byte("test data")
	n, err := w.Write(testData)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if n != len(testData) {
		t.Errorf("expected length %d, got %d", len(testData), n)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body, got length %d", rec.Body.Len())
	}
}
