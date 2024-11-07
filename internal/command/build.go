package command

import (
	"bytes"
	"fmt"
	"go/format"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/adhocteam/pushup/internal/compiler"
	"github.com/adhocteam/pushup/internal/scan"
	"github.com/adhocteam/pushup/internal/up"
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

	packagePaths := make(map[string]bool)
	packagePaths[modulePath+"/pages"] = true
	// This assumes scanner.Scan() has already been called
	for _, file := range project.Files {
		if file.Kind != up.Page {
			continue
		}

		dir := filepath.Dir(file.RelPath)
		if dir != "pages" {
			pkgPath := modulePath + "/" + dir
			packagePaths[pkgPath] = true
		}
	}

	mainTmpl, err := template.New("main.go").Parse(mainDotGo)
	if err != nil {
		return fmt.Errorf("parsing main.go template: %w", err)
	}

	var mainSrc bytes.Buffer
	data := map[string]any{
		"PackagePaths": sortPackagePaths(packagePaths),
		"ModulePath":   modulePath,
		"StaticDir":    scanner.Project().StaticDir,
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

func sortPackagePaths(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for path := range m {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

const mainDotGo = `package main

import (
    "log"

    "github.com/adhocteam/pushup/route"
    "github.com/adhocteam/pushup/server"

    {{ range .PackagePaths }}_ "{{ . }}"
    {{ end }}{{ if .StaticDir }}"{{ .ModulePath }}/static"{{ end }}
)

func main() {
    server := server.New(":8080")
    server.Mux().Handle("/", route.Handler())
    {{ if .StaticDir }}server.Mux().Handle("/static/", static.Handler()){{ end }}
    log.Fatal(server.ListenAndServe())
}
`
