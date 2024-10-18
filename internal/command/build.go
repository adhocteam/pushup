package command

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"

	"github.com/adhocteam/pushup/internal/compile"
	"github.com/adhocteam/pushup/internal/up"
	"golang.org/x/tools/go/packages"
)

func Build(root string) error {
	// TODO: take a logger optionally from the caller
	logger := slog.Default()
	logger.Info("Building", "root", root)

	for file := range up.Find(root, "up") {
		result, err := compile.Compile(file)
		if err != nil {
			return fmt.Errorf("compiling %q: %w", file, err)
		}
		logger.Info("Compiled", "source", file, "pkg", result.PkgName)
	}

	cfg := &packages.Config{Mode: packages.NeedModule}
	pkgs, err := packages.Load(cfg, root)
	if err != nil {
		return fmt.Errorf("loading package: %w", err)
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("expected at least one package, got none")
	}
	module := pkgs[0].Module
	if module == nil {
		return fmt.Errorf("no module found")
	}
	fmt.Println(module.Path)

	mainTmpl, err := template.New("main.go").Parse(mainDotGo)
	if err != nil {
		return fmt.Errorf("parsing main.go template: %w", err)
	}

	var mainSrc bytes.Buffer
	pagesPkg := module.Path + "/" + "pages"
	if err := mainTmpl.Execute(&mainSrc, map[string]any{"PagesPkg": pagesPkg}); err != nil {
		return fmt.Errorf("executing main.go template: %w", err)
	}

	if err := os.WriteFile(filepath.Join(root, "main.go"), mainSrc.Bytes(), 0664); err != nil {
		return fmt.Errorf("writing main.go: %w", err)
	}

	return nil
}

const mainDotGo = `package main

import (
    "log"
    "net/http"

    "github.com/adhocteam/pushup/route"

    _ "{{ .PagesPkg }}"
)

func main() {
    http.Handle("/", route.Handler())
    log.Fatal(http.ListenAndServe(":8080", nil))
}
`
