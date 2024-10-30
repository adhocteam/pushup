package command

import (
	"fmt"
	"os"

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/parser"
)

func PrettyPrintAST(file string) error {
	text, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	parser := parser.New()
	doc, err := parser.Parse(text)
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	ast.PrettyPrintTree(doc)

	return nil
}
