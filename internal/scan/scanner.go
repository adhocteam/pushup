package scan

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/adhocteam/pushup/internal/compiler"
	"github.com/adhocteam/pushup/internal/up"
	"golang.org/x/mod/modfile"
)

// Scanner is responsible for discovering and processing .up files within a Go
// project. It maintains information about the project's module and its
// associated .up files. The scanner can locate both page and component .up
// files, and supports compiling them into Go source code using a provided
// compiler.
type Scanner struct {
	project *up.Project
}

func New(rootDir string) (*Scanner, error) {
	modPath, modDir, err := findModulePath(rootDir)
	if err != nil {
		return nil, fmt.Errorf("finding go.mod: %w", err)
	}

	scanner := &Scanner{
		project: &up.Project{
			RootDir:   rootDir,
			Module:    up.Module{Path: modPath},
			ModuleDir: modDir,
			Files:     []up.File{},
		},
	}

	return scanner, nil
}

func (s *Scanner) Project() *up.Project {
	return s.project
}

func (s *Scanner) Scan() error {
	for file := range up.Find(s.project.RootDir, up.DotUp) {
		relPath, err := filepath.Rel(s.project.RootDir, file)
		if err != nil {
			return fmt.Errorf("determining relative path: %w", err)
		}

		var kind up.Kind
		if up.IsPage(file) {
			kind = up.Page
		} else {
			kind = up.Component
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		s.project.Files = append(s.project.Files, up.File{
			Path:    file,
			RelPath: relPath,
			Kind:    kind,
			Content: content,
		})
	}

	return nil
}

func (s *Scanner) CompileFiles(compiler compiler.Compiler) error {
	for _, file := range s.project.Files {
		slog.Info("compiling", "source", file.Path)
		code, err := compiler.Compile(s.project, file)
		if err != nil {
			return fmt.Errorf("compiling: %w", err)
		}
		target := file.Path + ".go"
		if err := os.WriteFile(target, code, 0644); err != nil {
			return fmt.Errorf("writing generated Go file: %w", err)
		}
	}
	return nil
}

func loadGoModule(root string) (mod up.Module, err error) {
	file := filepath.Join(root, "go.mod")
	content, err := os.ReadFile(file)
	if err != nil {
		err = fmt.Errorf("reading %q: %w", file, err)
		return
	}
	module, err := modfile.Parse(file, content, nil)
	if err != nil {
		err = fmt.Errorf("parsing go.mod: %w", err)
	}
	mod.Path = module.Module.Mod.Path
	return
}

func findModulePath(dir string) (modulePath string, goModDir string, err error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}

	current := absDir
	for {
		goModPath := filepath.Join(current, "go.mod")
		content, err := os.ReadFile(goModPath)
		if err == nil {
			f, err := modfile.Parse(goModPath, content, nil)
			if err != nil {
				return "", "", err
			}
			if f.Module == nil {
				return "", "", fmt.Errorf("go.mod file does not contain module declaration")
			}
			return f.Module.Mod.Path, current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", "", fmt.Errorf("go.mod file not found")
		}
		current = parent
	}
}
