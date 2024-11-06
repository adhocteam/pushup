package command

import (
	"bytes"
	"fmt"
	"go/format"
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

	if err := generateMainGo(scanner); err != nil {
		return fmt.Errorf("generating main.go: %w", err)
	}

	return nil
}

func generateMainGo(scanner *scan.Scanner) error {
	project := scanner.Project()
	modulePath := project.Module.Path

	mainTmpl, err := template.New("main.go").Parse(mainDotGo)
	if err != nil {
		return fmt.Errorf("parsing main.go template: %w", err)
	}

	var mainSrc bytes.Buffer
	pagesPkg := modulePath + "/" + "pages"
	data := map[string]any{
		"PagesPkg":   pagesPkg,
		"ModulePath": modulePath,
		"StaticDir":  scanner.Project().StaticDir,
	}
	if err := mainTmpl.Execute(&mainSrc, data); err != nil {
		return fmt.Errorf("executing main.go template: %w", err)
	}

	content, err := format.Source(mainSrc.Bytes())
	if err != nil {
		return fmt.Errorf("formatting main.go: %w", err)
	}

	if err := os.WriteFile(filepath.Join(project.RootDir, "main.go"), content, 0664); err != nil {
		return fmt.Errorf("writing main.go: %w", err)
	}

	return nil
}

const mainDotGo = `package main

import (
    "log"

    "github.com/adhocteam/pushup/route"
    "github.com/adhocteam/pushup/server"

    _ "{{ .PagesPkg }}"
    {{ if .StaticDir }}"{{ .ModulePath }}/static"{{ end }}
)

func main() {
    server := server.New(":8080")
    server.Mux().Handle("/", route.Handler())
    {{ if .StaticDir }}server.Mux().Handle("/static/", static.Handler()){{ end }}
    log.Fatal(server.ListenAndServe())
}
`
