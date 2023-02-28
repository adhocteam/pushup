package main

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
)

type linkerParams struct {
	compiledOutput *compiledOutput
	modPath        string
	projectDir     string
	exeName        string
	destDir        string
}

const filename = "pushup_linker.go"

// linkProject puts a Pushup project together by linking together all the
// generated Go source code and a main() function.
func linkProject(ctx context.Context, params *linkerParams) (string, error) {
	projectDir := params.projectDir
	exeName := params.exeName
	modPath := params.modPath
	pkgName := filepath.Base(modPath)
	pages := params.compiledOutput.pages

	// Generate http serve mux of all routes
	{
		f, err := os.Create(filepath.Join(params.projectDir, filename))
		if err != nil {
			return "", fmt.Errorf("creating servemux.go: %w", err)
		}
		defer f.Close()

		b := new(bytes.Buffer)

		importPaths := map[string]bool{}
		for _, page := range pages {
			importPaths[page.PkgPath] = true
		}
		importPaths["github.com/adhocteam/pushup/api"] = true
		if dirExists("static") {
			importPaths["embed"] = true
		}

		fmt.Fprintln(b, "// this file is mechanically generated, do not edit!")
		fmt.Fprintln(b, "package "+pkgName)
		for path := range importPaths {
			fmt.Fprintln(b, "import \""+path+"\"")
		}
		fmt.Fprintln(b, "var Router *api.Router")
		if dirExists("static") {
			fmt.Fprintln(b, "//go:embed static")
			fmt.Fprintln(b, "var static embed.FS")
		}
		fmt.Fprintln(b, "func init() {")
		fmt.Fprintln(b, "routes := new(api.Routes)")
		for _, page := range pages {
			pkgName := filepath.Base(page.PkgPath)
			var role string
			switch page.Role {
			case routePage:
				role = "api.RoutePage"
			case routePartial:
				role = "api.RoutePartial"
			}
			fmt.Fprintf(b, "routes.Add(\"%s\", &%s.%s{}, %s)\n",
				page.Route, pkgName, page.Name, role)
		}
		fmt.Fprintln(b, "Router = api.NewRouter(routes)")
		if dirExists("static") {
			fmt.Fprintln(b, "Router.AddStatic(static)")
		}
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b, "")

		formatted, err := format.Source(b.Bytes())
		if err != nil {
			return "", fmt.Errorf("gofmt on generated servemux.go: %w", err)
		}

		if _, err := f.Write(formatted); err != nil {
			return "", fmt.Errorf("writing formatted source to servemux.go: %w", err)
		}
	}

	// Generate main.go
	{
		mainPkgPath := filepath.Join(projectDir, "cmd", exeName)
		if err := os.MkdirAll(mainPkgPath, 0755); err != nil {
			return "", fmt.Errorf("making main package dir: %w", err)
		}

		f, err := os.Create(filepath.Join(mainPkgPath, "main.go"))
		if err != nil {
			return "", fmt.Errorf("creating main.go: %w", err)
		}
		defer f.Close()

		b := new(bytes.Buffer)

		fmt.Fprintln(b, "// this file is mechanically generated, do not edit!")
		fmt.Fprintln(b, "package main")
		fmt.Fprintf(b, "import \"%s\"\n", modPath)
		fmt.Fprintln(b, "import \"github.com/adhocteam/pushup/api\"")
		fmt.Fprintln(b, "func main() {")
		fmt.Fprintf(b, "api.Main(%s.Router)\n", pkgName)
		fmt.Fprintln(b, "}")

		formatted, err := format.Source(b.Bytes())
		if err != nil {
			return "", fmt.Errorf("gofmt on generated main.go: %w", err)
		}

		if _, err := f.Write(formatted); err != nil {
			return "", fmt.Errorf("writing formatted source to main.go: %w", err)
		}
	}

	// Run Go compiler
	exePath := filepath.Join(params.destDir, params.exeName)
	{
		args := []string{"build", "-o", exePath, filepath.Join(modPath, "cmd", exeName)}
		cmd := exec.Command("go", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("building project main executable: %w", err)
		}
	}

	return exePath, nil
}
