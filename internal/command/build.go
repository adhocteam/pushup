package command

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"

	"github.com/adhocteam/pushup/internal/compiler"
	"github.com/adhocteam/pushup/internal/scan"
)

func Build(root string) error {
	slog.Info("Building", "root", root)

	scanner, err := scan.New(root)
	if err != nil {
		return fmt.Errorf("getting new scanner: %w", err)
	}

	if err := scanner.Scan(); err != nil {
		return fmt.Errorf("scanning for Pushup files: %w", err)
	}

	compiler := compiler.New()

	if err := scanner.CompileFiles(compiler); err != nil {
		return fmt.Errorf("compiling files: %w", err)
	}

	project := scanner.Project()

	if err := generateMainGo(root, project.Module.Path); err != nil {
		return fmt.Errorf("generating main.go: %w", err)
	}

	return nil
}

func generateMainGo(root, modulePath string) error {
	mainTmpl, err := template.New("main.go").Parse(mainDotGo)
	if err != nil {
		return fmt.Errorf("parsing main.go template: %w", err)
	}

	var mainSrc bytes.Buffer
	pagesPkg := modulePath + "/" + "pages"
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
