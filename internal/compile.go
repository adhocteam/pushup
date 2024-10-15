package internal

import (
	"fmt"
	"os"

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/codegen"
	"github.com/adhocteam/pushup/internal/parser"
)

// Compile a single .up file
func Compile(file string) error {
	fmt.Fprintf(os.Stderr, "\x1b[33mCompiling: %s\x1b[0m\n", file)

	text, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	doc, err := parser.Parse(string(text))
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}
	ast.PrettyPrintTree(doc)

	// TODO: take a flag for optimizations
	doc = ast.Optimize(doc)

	out, err := codegen.GeneratePage(doc, string(text), file, "TKTK", "TKTK")
	if err != nil {
		return fmt.Errorf("generating page: %w", err)
	}

	fmt.Println(string(out))

	return nil
}
