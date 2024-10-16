package command

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/codegen"
	"github.com/adhocteam/pushup/internal/parser"
)

// Compile a single .up file
func Compile(file string) error {
	fmt.Fprintf(os.Stderr, "\x1b[33mCompiling: %s\x1b[0m\n", file)

	pkgName, err := goPackageName(filepath.Dir(file))
	if err != nil {
		return fmt.Errorf("Go package name: %w", err)
	}

	text, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	doc, err := parser.Parse(string(text))
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	// TODO: take a flag for optimizations
	doc = ast.Optimize(doc)

	out, err := codegen.GeneratePage(doc, string(text), file, "TKTK", pkgName)
	if err != nil {
		return fmt.Errorf("generating page: %w", err)
	}

	fmt.Println(string(out))

	return nil
}

func PrettyPrintAST(file string) error {
	text, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	doc, err := parser.Parse(string(text))
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	ast.PrettyPrintTree(doc)

	return nil
}

func goPackageName(dir string) (string, error) {
	// TODO: make sure this works in lots of different module/package
	// scenarios and relative calling working dirs
	pkg, err := build.ImportDir(dir, build.ImportComment)
	if err != nil {
		return "", fmt.Errorf("importing Go package at directory %q: %w", dir, err)
	}
	pkgName := pkg.Name
	return pkgName, nil
}
