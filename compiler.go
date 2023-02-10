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
	upFileLayout
)

type compileProjectParams struct {
	// path to project root directory
	root string

	// path to app dir within project
	appDir string

	// path to output build directory
	outDir string

	// flag to skip code generation
	parseOnly bool

	// paths to Pushup project files
	files *projectFiles

	// flag to apply a set of code generation optimizations
	applyOptimizations bool

	// flag to enable layouts (FIXME)
	enableLayout bool

	// embed .up source files in project executable
	embedSource bool
}

func compileProject(c *compileProjectParams) error {
	if c.parseOnly {
		for _, pfile := range append(c.files.pages, c.files.layouts...) {
			path := pfile.path
			b, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading file %s: %w", path, err)
			}

			tree, err := parse(string(b))
			if err != nil {
				return fmt.Errorf("parsing file %s: %w", path, err)
			}

			prettyPrintTree(tree)
			fmt.Println()
		}
		os.Exit(0)
	}

	// compile layouts
	for _, pfile := range c.files.layouts {
		if err := compileUpFile(pfile, upFileLayout, c); err != nil {
			return err
		}
	}

	// compile pages
	for _, pfile := range c.files.pages {
		if err := compileUpFile(pfile, upFilePage, c); err != nil {
			return err
		}
	}

	// "compile" static files
	for _, pfile := range c.files.static {
		relpath := pfile.relpath()
		destDir := filepath.Join(c.outDir, "static", filepath.Dir(relpath))
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("making intermediate directory in static dir %s: %v", destDir, err)
		}
		destPath := filepath.Join(destDir, filepath.Base(relpath))
		if err := copyFile(destPath, pfile.path); err != nil {
			return fmt.Errorf("copying static file %s to %s: %w", pfile.path, destPath, err)
		}
	}

	// copy over Pushup runtime support Go code
	t := template.Must(template.ParseFS(runtimeFiles, filepath.Join("_runtime", "pushup_support.go")))
	f, err := os.Create(filepath.Join(c.outDir, "pushup_support.go"))
	if err != nil {
		return fmt.Errorf("creating pushup_support.go: %w", err)
	}
	if err := t.Execute(f, map[string]any{"EmbedStatic": c.enableLayout}); err != nil { // FIXME
		return fmt.Errorf("executing pushup_support.go template: %w", err)
	}
	f.Close()

	if c.embedSource {
		outSrcDir := filepath.Join(c.outDir, "src")
		for _, pfile := range c.files.pages {
			relpath := pfile.relpath()
			dir := filepath.Join(outSrcDir, "pages", filepath.Dir(relpath))
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			dest := filepath.Join(outSrcDir, "pages", relpath)
			if err := copyFile(dest, pfile.path); err != nil {
				return fmt.Errorf("copying page file %s to %s: %v", pfile.path, dest, err)
			}
		}
	}

	return nil
}

// compileUpFile compiles a single .up file in a Pushup project context. it
// outputs .go code to a file in the build directory.
func compileUpFile(pfile projectFile, ftype upFileType, projectParams *compileProjectParams) error {
	path := pfile.path
	sourceFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening source file %s: %w", path, err)
	}
	defer sourceFile.Close()
	destPath := filepath.Join(projectParams.outDir, compiledOutputPath(pfile, ftype))
	destDir := filepath.Dir(destPath)
	if err = os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("making destination file's directory %s: %w", destDir, err)
	}
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("opening destination file %s: %w", destPath, err)
	}
	defer destFile.Close()
	params := compileParams{
		source:             sourceFile,
		dest:               destFile,
		pfile:              pfile,
		ftype:              ftype,
		applyOptimizations: projectParams.applyOptimizations,
	}
	if err := compile(params); err != nil {
		return fmt.Errorf("compiling page file %s: %w", path, err)
	}
	return nil
}

// compiledOutputPath returns the filename for the .go file containing the
// generated code for the Pushup page.
func compiledOutputPath(pfile projectFile, ftype upFileType) string {
	rel, err := filepath.Rel(pfile.projectFilesSubdir, pfile.path)
	if err != nil {
		panic("internal error: relative path from project files subdir to .up file: " + err.Error())
	}
	// a .go file with a leading '$' in the name is invalid to the go tool
	if rel[0] == '$' {
		rel = "0x24" + rel[1:]
	}
	var dirs []string
	dir := filepath.Dir(rel)
	if dir != "." {
		dirs = strings.Split(dir, string([]rune{os.PathSeparator}))
	}
	file := filepath.Base(rel)
	base := strings.TrimSuffix(file, filepath.Ext(file))
	suffix := upFileExt
	if ftype == upFileLayout {
		suffix = ".layout.up"
	}
	result := strings.Join(append(dirs, base), "__") + suffix + ".go"
	return result
}

type compileParams struct {
	source             io.Reader
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
	case upFileLayout:
		layout, err := newLayoutFromTree(tree)
		if err != nil {
			return fmt.Errorf("getting layout from tree: %w", err)
		}
		codeGen := newLayoutCodeGen(layout, params.pfile, src)
		code, err = genCodeLayout(codeGen)
		if err != nil {
			return fmt.Errorf("generating code for a layout: %w", err)
		}
	case upFilePage:
		page, err := newPageFromTree(tree)
		if err != nil {
			return fmt.Errorf("getting page from tree: %w", err)
		}
		codeGen := newPageCodeGen(page, params.pfile, src)
		code, err = genCodePage(codeGen)
		if err != nil {
			return fmt.Errorf("generating code for a page: %w", err)
		}
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
