package tree_sitter_pushup_test

import (
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_pushup "github.com/adhocteam/pushup/bindings/go"
)

func TestCanLoadGrammar(t *testing.T) {
	language := tree_sitter.NewLanguage(tree_sitter_pushup.Language())
	if language == nil {
		t.Errorf("Error loading Pushup grammar")
	}
}
