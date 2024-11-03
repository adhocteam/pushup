package compiler

import (
	"errors"
	"fmt"
	"go/build"
	"path/filepath"

	"github.com/adhocteam/pushup/internal/analyzer"
	"github.com/adhocteam/pushup/internal/codegen"
	"github.com/adhocteam/pushup/internal/parser"
	"github.com/adhocteam/pushup/internal/up"
)

type Compiler interface {
	Compile(project *up.Project, file up.File) ([]byte, error)
}

type compiler struct {
	parser    parser.Parser
	generator codegen.Generator
}

func New() *compiler {
	return &compiler{
		parser:    parser.New(),
		generator: codegen.New(),
	}
}

func (c *compiler) Compile(project *up.Project, file up.File) ([]byte, error) {
	unit := up.NewCompileUnit(project, &file)

	pkg, err := deriveGoPackage(filepath.Dir(file.Path))
	if err != nil {
		return nil, fmt.Errorf("Go package name: %w", err)
	}
	unit.Package = pkg

	doc, err := c.parser.Parse(file.Content)
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}

	if err := analyzer.Analyze(doc, unit); err != nil {
		return nil, fmt.Errorf("analyzing parse: %w", err)
	}

	code, err := c.generator.Generate(unit)
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}

	return code, nil
}

func deriveGoPackage(dir string) (string, error) {
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
