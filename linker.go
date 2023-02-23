package main

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
)

type linkerParams struct {
	output     *compiledOutput
	modPath    string
	projectDir string
	exeName    string
}

// linkProject puts a Pushup project together by linking together all the
// generated Go source code and a main() function.
func linkProject(ctx context.Context, params *linkerParams) error {
	projectDir := params.projectDir
	exeName := params.exeName

	// Generate http serve mux of all routes
	{
		f, err := os.Create(filepath.Join(params.projectDir, "servemux.go"))
		if err != nil {
			return fmt.Errorf("creating servemux.go: %w", err)
		}
		defer f.Close()

		b := new(bytes.Buffer)

		fmt.Fprintln(b, "// this file is mechanically generated, do not edit!")
		fmt.Fprintln(b, "package " + )
		fmt.Fprintf(b, "import \"%s\"\n", "TODO")
		fmt.Fprintln(b, "func main() {")
		fmt.Fprintln(b, "}")

		formatted, err := format.Source(b.Bytes())
		if err != nil {
			return fmt.Errorf("gofmt on generated main.go: %w", err)
		}

		if _, err := f.Write(formatted); err != nil {
			return fmt.Errorf("writing formatted source to main.go: %w", err)
		}
	}

	// Generate main.go
	{
		mainPkgPath := filepath.Join(projectDir, "cmd", exeName)
		if err := os.MkdirAll(mainPkgPath, 0755); err != nil {
			return fmt.Errorf("making main package dir: %w", err)
		}

		f, err := os.Create(filepath.Join(mainPkgPath, "main.go"))
		if err != nil {
			return fmt.Errorf("creating main.go: %w", err)
		}
		defer f.Close()

		b := new(bytes.Buffer)

		fmt.Fprintln(b, "// this file is mechanically generated, do not edit!")
		fmt.Fprintln(b, "package main")
		fmt.Fprintf(b, "import \"%s\"\n", "TODO")
		fmt.Fprintln(b, "func main() {")
		fmt.Fprintln(b, "}")

		formatted, err := format.Source(b.Bytes())
		if err != nil {
			return fmt.Errorf("gofmt on generated main.go: %w", err)
		}

		if _, err := f.Write(formatted); err != nil {
			return fmt.Errorf("writing formatted source to main.go: %w", err)
		}
	}

	// Run Go compiler

	return nil
}
