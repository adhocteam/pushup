package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type upFileType int

const (
	upFilePage upFileType = iota
	upFileComponent
)

type compileProjectParams struct {
	// modPath is the Go module path specified in the project's go.mod file
	modPath string

	// path to output build directory
	outDir string

	// flag to skip code generation
	parseOnly bool

	// paths to Pushup project files
	files *projectFiles

	// flag to apply a set of code generation optimizations
	applyOptimizations bool

	// embed .up source files in project executable
	embedSource bool
}

type routeRole int

const (
	routePage routeRole = iota
	routePartial
)

type page struct {
	PkgPath string
	Name    string
	Route   string
	Role    routeRole
}

type compiledOutput struct {
	pages []*page
}

func compileProject(c *compileProjectParams) (*compiledOutput, error) {
	var output compiledOutput

	if c.parseOnly {
		for _, pfile := range c.files.pages {
			path := pfile.path
			b, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("reading file %s: %w", path, err)
			}

			tree, err := parse(string(b))
			if err != nil {
				return nil, fmt.Errorf("parsing file %s: %w", path, err)
			}

			prettyPrintTree(tree)
			fmt.Println()
		}
		os.Exit(0)
	}

	// compile pages
	for _, pfile := range c.files.pages {
		pages, err := compileUpFile(pfile, upFilePage, c)
		if err != nil {
			return nil, err
		}
		output.pages = append(output.pages, pages...)
	}

	return &output, nil
}

func packageName(path string) string {
	dir := filepath.Base(filepath.Dir(path))
	if dir == "." {
		return "main"
	}
	return dir
}

// compileUpFile compiles a single .up file in a Pushup project context. It
// outputs Go code to a .up.go file in the same directory as the .up file.
func compileUpFile(pfile projectFile, ftype upFileType, projectParams *compileProjectParams) ([]*page, error) {
	sourcePath := pfile.path
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("opening source file %s: %w", sourcePath, err)
	}
	defer sourceFile.Close()
	pkgName := packageName(sourcePath)
	destPath := compiledOutputPath(pfile, ftype)
	destFile, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("creating/truncating destination file %s: %w", destPath, err)
	}
	defer destFile.Close()
	params := &compileParams{
		source:             sourceFile,
		modPath:            projectParams.modPath,
		pkgName:            pkgName,
		dest:               destFile,
		pfile:              pfile,
		ftype:              ftype,
		applyOptimizations: projectParams.applyOptimizations,
	}
	pages, err := compile(params)
	if err != nil {
		return nil, fmt.Errorf("compiling page file %s: %w", sourcePath, err)
	}
	return pages, nil
}

// compiledOutputPath returns the filename for the .go file containing the
// generated code for the Pushup page.
func compiledOutputPath(pfile projectFile, ftype upFileType) string {
	path := pfile.path
	file := filepath.Base(path)
	base := strings.TrimSuffix(file, filepath.Ext(file))
	result := filepath.Join(filepath.Dir(path), base+compiledFileExt)
	return result
}

type compileParams struct {
	source             io.Reader
	modPath            string
	pkgName            string
	dest               io.Writer
	pfile              projectFile
	ftype              upFileType
	applyOptimizations bool
}

// compile compiles Pushup source code. it parses the source, applies
// optimizations to the resulting syntax tree, and generates Go code from the
// tree.
func compile(params *compileParams) ([]*page, error) {
	b, err := io.ReadAll(params.source)
	if err != nil {
		return nil, fmt.Errorf("reading source: %w", err)
	}
	src := string(b)

	tree, err := parse(src)
	if err != nil {
		return nil, fmt.Errorf("parsing source: %w", err)
	}

	if params.applyOptimizations {
		tree = optimize(tree)
	}

	var result *codeGenResult

	switch params.ftype {
	case upFilePage:
		page, err := newPageFromTree(tree)
		if err != nil {
			return nil, fmt.Errorf("getting page from tree: %w", err)
		}
		codeGen := newPageCodeGen(page, src, params)
		result, err = genCodePage(codeGen)
		if err != nil {
			return nil, fmt.Errorf("generating code for a page: %w", err)
		}
	case upFileComponent:
		panic("UNIMPLEMENTED")
	}
	if err != nil {
		return nil, fmt.Errorf("generating code: %w", err)
	}

	if _, err := params.dest.Write(result.code); err != nil {
		return nil, fmt.Errorf("writing generated page code: %w", err)
	}

	return result.Pages, nil
}

func copyFile(dest, src string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if err := os.WriteFile(dest, b, 0664); err != nil {
		return err
	}

	return nil
}
