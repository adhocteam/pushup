package compile

import (
	"errors"
	"fmt"
	"go/build"
	"os"
	"path/filepath"

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/codegen"
	"github.com/adhocteam/pushup/internal/parser"
)

type Result struct {
	PkgName string
	GenGo   string
}

func Compile(file string) (*Result, error) {
	pkgName, err := goPackageName(filepath.Dir(file))
	if err != nil {
		return nil, fmt.Errorf("Go package name: %w", err)
	}

	text, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	doc, err := parser.Parse(string(text))
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	// TODO: take a flag for optimizations
	doc = ast.Optimize(doc)

	gcode, err := codegen.GeneratePage(doc, string(text), file, "TKTK", pkgName)
	if err != nil {
		return nil, fmt.Errorf("generating page: %w", err)
	}

	target := file + ".go"
	if err := os.WriteFile(target, gcode, 0644); err != nil {
		return nil, fmt.Errorf("writing file %q: %w", target, err)
	}

	result := &Result{
		PkgName: pkgName,
		GenGo:   string(gcode),
	}

	return result, nil
}

func goPackageName(dir string) (string, error) {
	// TODO: make sure this works in lots of different module/package
	// scenarios and relative calling working dirs
	pkg, err := build.ImportDir(dir, build.ImportComment)
	var noGoErr *build.NoGoError
	if err != nil {
		if errors.As(err, &noGoErr) {
			// use dir name as package name
			// TODO: might be wrong
			name := filepath.Base(dir)
			if name == "." {
				name = "main"
			}
			return name, nil
		}
		return "", fmt.Errorf("importing Go package at directory %q: %w", dir, err)
	}
	return pkg.Name, nil
}
