package scan

import (
	"fmt"
	"go/format"
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
			StaticDir: "",
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

	if staticDir, has := s.hasStaticDir(); has {
		s.project.StaticDir = staticDir
		slog.Debug("static assets directory detected", "path", staticDir)
		if err := s.generateStaticPackage(); err != nil {
			return fmt.Errorf("generating static package: %w", err)
		}
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

func (s *Scanner) hasStaticDir() (string, bool) {
	// TODO: allow for static name to be configured/overridden
	staticDir := filepath.Join(s.project.RootDir, "static")
	if info, err := os.Stat(staticDir); err == nil {
		return staticDir, info.IsDir()
	}
	return "", false
}

func (s *Scanner) generateStaticPackage() error {
	staticDir := filepath.Join(s.project.RootDir, "static")
	staticGo := filepath.Join(staticDir, "static.go")
	slog.Info("generating static package", "path", staticGo)
	// TODO: allow for package name to be configured/overridden
	content := []byte(`package static

import (
    "embed"
    "net/http"
    "log/slog"
)

//go:embed *
var assets embed.FS

func Handler() http.Handler {
    return http.StripPrefix("/static/", http.FileServer(http.FS(assets)))
}
`)

	content, err := format.Source(content)
	if err != nil {
		return fmt.Errorf("formatting static.go: %w", err)
	}

	if err := os.WriteFile(staticGo, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing static.go: %w", err)
	}

	return nil
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
