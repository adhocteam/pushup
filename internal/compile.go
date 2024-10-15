package internal

import (
	"fmt"
	"os"

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/parser"
)

// Compile a single .up file
func Compile(file string) error {
	fmt.Fprintf(os.Stderr, "\x1b[33mCompiling: %s\x1b[0m\n", file)

	doc, err := parser.ParseFile(file)
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}
	ast.PrettyPrintTree(doc)

	return nil
}
