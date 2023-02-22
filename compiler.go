package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type upFileType int

const (
	upFilePage upFileType = iota
	upFileComponent
)

type compileProjectParams struct {
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

type page struct {
	path    string
	urlPath string
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
		if err := compileUpFile(pfile, upFilePage, c); err != nil {
			return nil, err
		}
	}

	// "compile" static files
	for _, pfile := range c.files.static {
		relpath := pfile.relpath()
		destDir := filepath.Join(c.outDir, "static", filepath.Dir(relpath))
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return nil, fmt.Errorf("making intermediate directory in static dir %s: %v", destDir, err)
		}
		destPath := filepath.Join(destDir, filepath.Base(relpath))
		if err := copyFile(destPath, pfile.path); err != nil {
			return nil, fmt.Errorf("copying static file %s to %s: %w", pfile.path, destPath, err)
		}
	}

	// TODO(paulsmith): move this to linking step
	// copy over Pushup runtime support Go code
	t := template.Must(template.ParseFS(runtimeFiles, filepath.Join("_runtime", "pushup_support.go")))
	f, err := os.Create(filepath.Join(c.outDir, "pushup_support.go"))
	if err != nil {
		return nil, fmt.Errorf("creating pushup_support.go: %w", err)
	}
	if err := t.Execute(f, map[string]any{"EmbedStatic": true}); err != nil { // FIXME
		return nil, fmt.Errorf("executing pushup_support.go template: %w", err)
	}
	f.Close()

	// TODO(paulsmith): move this to linking step
	if c.embedSource {
		outSrcDir := filepath.Join(c.outDir, "src")
		for _, pfile := range c.files.pages {
			relpath := pfile.relpath()
			dir := filepath.Join(outSrcDir, "pages", filepath.Dir(relpath))
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, err
			}
			dest := filepath.Join(outSrcDir, "pages", relpath)
			if err := copyFile(dest, pfile.path); err != nil {
				return nil, fmt.Errorf("copying page file %s to %s: %v", pfile.path, dest, err)
			}
		}
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
func compileUpFile(pfile projectFile, ftype upFileType, projectParams *compileProjectParams) error {
	sourcePath := pfile.path
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("opening source file %s: %w", sourcePath, err)
	}
	defer sourceFile.Close()
	pkgName := packageName(sourcePath)
	destPath := compiledOutputPath(pfile, ftype)
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating/truncating destination file %s: %w", destPath, err)
	}
	defer destFile.Close()
	params := compileParams{
		source:             sourceFile,
		pkgName:            pkgName,
		dest:               destFile,
		pfile:              pfile,
		ftype:              ftype,
		applyOptimizations: projectParams.applyOptimizations,
	}
	if err := compile(params); err != nil {
		return fmt.Errorf("compiling page file %s: %w", sourcePath, err)
	}
	return nil
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
	pkgName            string
	dest               io.Writer
	pfile              projectFile
	ftype              upFileType
	applyOptimizations bool
}

// compile compiles Pushup source code. it parses the source, applies
// optimizations to the resulting syntax tree, and generates Go code from the
// tree.
func compile(params compileParams) error {
	b, err := io.ReadAll(params.source)
	if err != nil {
		return fmt.Errorf("reading source: %w", err)
	}
	src := string(b)

	tree, err := parse(src)
	if err != nil {
		return fmt.Errorf("parsing source: %w", err)
	}

	if params.applyOptimizations {
		tree = optimize(tree)
	}

	var code []byte

	switch params.ftype {
	case upFilePage:
		page, err := newPageFromTree(tree)
		if err != nil {
			return fmt.Errorf("getting page from tree: %w", err)
		}
		codeGen := newPageCodeGen(page, params.pfile, src, params.pkgName)
		code, err = genCodePage(codeGen)
		if err != nil {
			return fmt.Errorf("generating code for a page: %w", err)
		}
	case upFileComponent:
		panic("UNIMPLEMENTED")
	}
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	if _, err := params.dest.Write(code); err != nil {
		return fmt.Errorf("writing generated page code: %w", err)
	}

	return nil
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
