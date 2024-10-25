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
	"github.com/adhocteam/pushup/internal/up"
)

type Compilation struct {
	// SourceFile is the path to the source Pushup file
	SourceFile string
	// Target is the path to the generated Go file
	Target string
	// Kind is the type of Pushup file - page or component
	Kind up.Kind
	// PkgName is the Go package of the generated Go file
	PkgName string
	// GeneratedGo is the generated Go code
	GeneratedGo string
}

// Page compiles a Pushup file into a Go file.
// It compiles a routable page, meaning a file with the .up extension in the
// "pages" package or subpackage.
func Page(file string) (*Compilation, error) {
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

	result := &Compilation{
		SourceFile:  file,
		Target:      target,
		Kind:        up.Page,
		PkgName:     pkgName,
		GeneratedGo: string(gcode),
	}

	return result, nil
}

func Component(file string) (*Compilation, error) {
	// TODO ...

	result := &Compilation{
		SourceFile:  file,
		Target:      "",
		Kind:        up.Component,
		PkgName:     "",
		GeneratedGo: "",
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
