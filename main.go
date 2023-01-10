// Pushup web framework
package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"go/format"
	goparser "go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/fs"
	"log"
	"math"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"
	"text/template"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/sync/errgroup"
)

const upFileExt = ".up"

func main() {
	var version bool

	flag.Usage = printPushupHelp
	flag.BoolVar(&version, "version", false, "Print the version number and exit")

	flag.Parse()

	if version {
		printVersion(os.Stdout)
		os.Exit(0)
	}

	log.SetFlags(0)

	// Check that Go is installed
	// TODO(paulsmith): check that a minimum Go version is installed
	if _, err := exec.LookPath("go"); err != nil {
		log.Fatalf("Pushup requires Go to be installed.")
	}

	printBanner()

	if flag.NArg() == 0 {
		printPushupHelp()
		os.Exit(1)
	}

	cmdName := flag.Arg(0)
	args := flag.Args()[1:]

	for _, c := range cliCmds {
		if c.name == cmdName {
			cmd := c.fn(args)
			if err := cmd.do(); err != nil {
				log.Fatalf("%s command: %v", c.name, err)
			} else {
				os.Exit(0)
			}
		}
	}
	fmt.Fprintf(os.Stderr, "unknown command %q\n", cmdName)
	flag.Usage()
	os.Exit(1)
}

type regexString struct {
	re  *regexp.Regexp
	val string
}

func newRegexString(pat string, defaultStr string) *regexString {
	return &regexString{re: regexp.MustCompile(pat), val: defaultStr}
}

func (r *regexString) String() string {
	return r.val
}

func (r *regexString) Set(value string) error {
	if r.re.MatchString(value) {
		r.val = value
	} else {
		return errors.New("supplied flag value does not match regex")
	}
	return nil
}

type stringSlice []string

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *stringSlice) String() string {
	return strings.Join(*s, " ")
}

type doer interface {
	do() error
}

type newCmd struct {
	projectDir string
	moduleName string
}

func newNewCmd(arguments []string) *newCmd {
	flags := flag.NewFlagSet("pushup new", flag.ExitOnError)
	moduleNameFlag := newRegexString(`^\w[\w-]*$`, "example/myproject")
	flags.Var(moduleNameFlag, "module", "name of Go module of the new Pushup app")
	flags.Parse(arguments)
	if flags.NArg() > 1 {
		log.Fatalf("extra unprocessed argument(s)")
	}
	projectDir := "."
	if flags.NArg() == 1 {
		projectDir = flags.Arg(0)
	}
	return &newCmd{projectDir: projectDir, moduleName: moduleNameFlag.String()}
}

//go:embed scaffold
var scaffold embed.FS

func (n *newCmd) do() error {
	// check for existing files, bail if any exist
	if dirExists(n.projectDir) {
		if files, err := os.ReadDir(n.projectDir); err != nil {
			return fmt.Errorf("reading directory: %w", err)
		} else if len(files) > 0 {
			return fmt.Errorf("existing files in directory, refusing to overwrite")
		}
	}

	// create project directory structure
	for _, name := range []string{"pages", "layouts", "pkg", "static"} {
		path := filepath.Join(n.projectDir, appDirName, name)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("creating project directory %s: %w", path, err)
		}
	}

	scaffoldFiles := []string{
		"layouts/default.up",
		"pages/index.up",
		"static/style.css",
		"static/htmx.min.js",
		"pkg/app.go",
	}
	for _, name := range scaffoldFiles {
		dest := filepath.Join(n.projectDir, "app", name)
		src := filepath.Join("scaffold", name)
		if err := copyFileFS(scaffold, dest, src); err != nil {
			return fmt.Errorf("copying scaffold file to project dir %w", err)
		}
	}

	if err := createGoModFile(n.projectDir, n.moduleName); err != nil {
		return err
	}

	if err := initVcs(n.projectDir, vcsGit); err != nil {
		return err
	}

	return nil
}

func createGoModFile(destDir string, moduleName string) error {
	cmd := exec.Command("go", "mod", "init", moduleName)
	cmd.Dir = destDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("creating new go.mod file: %w", err)
	}
	return nil
}

type vcs int

const (
	vcsGit vcs = iota
)

func initVcs(projectDir string, vcs vcs) error {
	switch vcs {
	case vcsGit:
		path, err := exec.LookPath("git")
		if err != nil {
			log.Printf("WARN: git not found in $PATH")
			return nil
		}

		cmd := exec.Command(path, "init")
		cmd.Dir = projectDir
		if err = cmd.Run(); err != nil {
			return fmt.Errorf("git init: %w", err)
		}

		f, err := os.Create(filepath.Join(projectDir, ".gitignore"))
		if err != nil {
			return fmt.Errorf("creating .gitignore: %w", err)
		}
		defer f.Close()
		fmt.Fprintln(f, "/build")
	default:
		panic("internal error: unimplemented VCS")
	}

	return nil
}

type buildCmd struct {
	projectName        *regexString
	projectDir         string
	buildPkg           string
	applyOptimizations bool
	parseOnly          bool
	codeGenOnly        bool
	compileOnly        bool
	outDir             string
	embedSource        bool
	pages              stringSlice

	files  *projectFiles
	appDir string
}

func setBuildFlags(flags *flag.FlagSet, b *buildCmd) {
	b.projectName = newRegexString(`^\w+`, "myproject")
	flags.Var(b.projectName, "project", "name of Pushup project")
	flags.StringVar(&b.buildPkg, "build-pkg", "example/myproject/build", "name of package of compiled Pushup app")
	flags.BoolVar(&b.applyOptimizations, "O", false, "apply simple optimizations to the parse tree")
	flags.BoolVar(&b.parseOnly, "parse-only", false, "exit after dumping parse result")
	flags.BoolVar(&b.codeGenOnly, "codegen-only", false, "codegen only, don't compile")
	flags.BoolVar(&b.compileOnly, "compile-only", false, "compile only, don't start web server after")
	flags.StringVar(&b.outDir, "out-dir", "./build", "path to output build directory")
	flags.BoolVar(&b.embedSource, "embed-source", true, "embed the source .up files in executable")
	flags.Var(&b.pages, "page", "path to a Pushup page. mulitple can be given")
}

const appDirName = "app"

func newBuildCmd(arguments []string) *buildCmd {
	flags := flag.NewFlagSet("pushup build", flag.ExitOnError)
	b := new(buildCmd)
	setBuildFlags(flags, b)
	flags.Parse(arguments)
	if flags.NArg() == 1 {
		b.projectDir = flags.Arg(0)
	} else {
		b.projectDir = "."
	}
	b.appDir = filepath.Join(b.projectDir, appDirName)
	return b
}

func (b *buildCmd) rescanProjectFiles() error {
	if len(b.pages) == 0 {
		var err error
		b.files, err = findProjectFiles(b.appDir)
		if err != nil {
			return err
		}
	} else {
		pfiles := &projectFiles{}
		for _, page := range b.pages {
			pfiles.pages = append(pfiles.pages, projectFile{path: page, projectFilesSubdir: ""})
		}
		b.files = pfiles
	}
	return nil
}

func (b *buildCmd) do() error {
	if err := os.RemoveAll(b.outDir); err != nil {
		return fmt.Errorf("removing build dir: %w", err)
	}

	// FIXME(paulsmith): remove singleFile (and -single flag) and replace with
	// configurable project root, leading path strip, and optional file paths.
	if err := b.rescanProjectFiles(); err != nil {
		return err
	}

	// FIXME(paulsmith): dedupe this with runCmd.do()
	{
		params := &compileProjectParams{
			root:               b.projectDir,
			appDir:             b.appDir,
			outDir:             b.outDir,
			parseOnly:          b.parseOnly,
			files:              b.files,
			applyOptimizations: b.applyOptimizations,
			enableLayout:       len(b.pages) == 0, // FIXME
			embedSource:        b.embedSource,
		}

		if err := compileProject(params); err != nil {
			return fmt.Errorf("parsing and compiling: %w", err)
		}
	}

	if b.parseOnly || b.codeGenOnly {
		return nil
	}

	{
		params := buildParams{
			projectName:       b.projectName.String(),
			pkgName:           b.buildPkg,
			compiledOutputDir: b.outDir,
			buildDir:          b.outDir,
		}
		if err := buildProject(context.Background(), params); err != nil {
			return fmt.Errorf("building project: %w", err)
		}
	}

	return nil
}

type runCmd struct {
	*buildCmd
	host       string
	port       string
	unixSocket string
	devReload  bool
}

func newRunCmd(arguments []string) *runCmd {
	flags := flag.NewFlagSet("pushup run", flag.ExitOnError)
	b := new(buildCmd)
	setBuildFlags(flags, b)
	host := flags.String("host", "0.0.0.0", "host to listen on")
	port := flags.String("port", "8080", "port to listen on with TCP IPv4")
	unixSocket := flags.String("unix-socket", "", "path to listen on with Unix socket")
	devReload := flags.Bool("dev", false, "compile and run the Pushup app and reload on changes")
	flags.Parse(arguments)
	// FIXME this logic is duplicated with newBuildCmd
	if flags.NArg() == 1 {
		b.projectDir = flags.Arg(0)
	} else {
		b.projectDir = "."
	}
	// FIXME this logic is duplicated with newBuildCmd
	b.appDir = filepath.Join(b.projectDir, appDirName)
	return &runCmd{buildCmd: b, host: *host, port: *port, unixSocket: *unixSocket, devReload: *devReload}
}

func (r *runCmd) do() error {
	if err := r.buildCmd.do(); err != nil {
		return fmt.Errorf("build command: %w", err)
	}

	if r.compileOnly {
		return nil
	}

	// TODO(paulsmith): add a linkOnly flag (or a releaseMode flag,
	// alternatively?)
	ctx := newPushupContext(context.Background())

	if r.devReload {
		var mu sync.Mutex
		buildComplete := sync.NewCond(&mu)
		reload := make(chan struct{})
		tmpdir, err := os.MkdirTemp("", "pushupdev")
		if err != nil {
			return fmt.Errorf("creating temp dir: %v", err)
		}
		defer os.RemoveAll(tmpdir)
		socketPath := filepath.Join(tmpdir, "pushup-"+strconv.Itoa(os.Getpid())+".sock")
		if err = startReloadRevProxy(socketPath, buildComplete, r.port); err != nil {
			return fmt.Errorf("starting reverse proxy: %v", err)
		}
		ln, err := net.Listen("unix", socketPath)
		if err != nil {
			return fmt.Errorf("listening on Unix socket: %v", err)
		}
		go func() {
			for {
				select {
				case <-reload:
					ctx.fileChangeCancel()
				case <-ctx.sigNotifyCtx.Done():
					ctx.sigStop()
					return
				}
			}
		}()
		for {
			select {
			case <-ctx.sigNotifyCtx.Done():
				ctx.sigStop()
				return nil
			default:
			}

			if err := r.rescanProjectFiles(); err != nil {
				return fmt.Errorf("scanning for project files: %v", err)
			}

			{
				params := &compileProjectParams{
					root:               r.projectDir,
					appDir:             r.appDir,
					outDir:             r.outDir,
					parseOnly:          r.parseOnly,
					files:              r.files,
					applyOptimizations: r.applyOptimizations,
					enableLayout:       len(r.pages) == 0, // FIXME
					embedSource:        r.embedSource,
				}
				if err := compileProject(params); err != nil {
					return fmt.Errorf("parsing and compiling: %v", err)
				}
			}

			ctx = newPushupContext(context.Background())

			{
				params := buildParams{
					projectName:       r.projectName.String(),
					pkgName:           r.buildPkg,
					compiledOutputDir: r.outDir,
					buildDir:          r.outDir,
				}
				if err := buildProject(ctx, params); err != nil {
					return fmt.Errorf("building Pushup project: %v", err)
				}
			}

			buildComplete.Broadcast()
			go func() {
				watchForReload(ctx, ctx.fileChangeCancel, r.appDir, reload)
			}()
			if err := runProject(ctx, filepath.Join(r.outDir, "bin", r.projectName.String()+".exe"), ln); err != nil {
				return fmt.Errorf("building and running generated Go code: %v", err)
			}
		}
	} else {
		var err error
		var ln net.Listener
		if r.unixSocket != "" {
			ln, err = net.Listen("unix", r.unixSocket)
			if err != nil {
				return fmt.Errorf("listening on Unix socket: %v", err)
			}
		} else {
			addr := fmt.Sprintf("%s:%s", r.host, r.port)
			ln, err = net.Listen("tcp4", addr)
			if err != nil {
				return fmt.Errorf("listening on TCP socket: %v", err)
			}
		}
		if err := runProject(ctx, filepath.Join(r.outDir, "bin", r.projectName.String()+".exe"), ln); err != nil {
			return fmt.Errorf("building and running generated Go code: %v", err)
		}
	}

	return nil
}

type routesCmd struct {
	projectDir string
}

func newRoutesCmd(args []string) *routesCmd {
	flags := flag.NewFlagSet("pushup routes", flag.ExitOnError)
	r := new(routesCmd)
	flags.Parse(args)
	if flags.NArg() == 1 {
		r.projectDir = flags.Arg(0)
	} else {
		r.projectDir = "."
	}
	return r
}

func (r *routesCmd) do() error {
	fmt.Fprintf(os.Stderr, "routes! TBD\n")
	return nil
}

var _ doer = (*routesCmd)(nil)

type cliCmd struct {
	name        string
	usage       string
	description string
	fn          func(args []string) doer
}

var cliCmds = []cliCmd{
	{name: "new", usage: "[path]", description: "create new Pushup project directory", fn: func(args []string) doer { return newNewCmd(args) }},
	{name: "build", usage: "", description: "compile Pushup project and build executable", fn: func(args []string) doer { return newBuildCmd(args) }},
	{name: "run", usage: "", description: "build and run Pushup project app", fn: func(args []string) doer { return newRunCmd(args) }},
	{name: "routes", usage: "", description: "print the routes in the Pushup project", fn: func(args []string) doer { return newRoutesCmd(args) }},
}

func printPushupHelp() {
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "Usage: pushup [command] [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "\t-version\t\tPrint the version number and exit")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	for _, c := range cliCmds {
		fmt.Fprintf(w, "\t%s %s\t\t%s\n", c.name, c.usage, c.description)
	}
	w.Flush()
}

//go:embed banner.txt
var bannerFile embed.FS
var banner, _ = bannerFile.ReadFile("banner.txt")

func printBanner() {
	fmt.Fprintf(os.Stdout, "\n%s\n", banner)
}

type pushupContext struct {
	*cancellationSource
	fileChangeCtx    context.Context
	fileChangeCancel context.CancelFunc
	sigNotifyCtx     context.Context
	sigStop          context.CancelFunc
}

func newPushupContext(parent context.Context) *pushupContext {
	c := new(pushupContext)
	c.fileChangeCtx, c.fileChangeCancel = context.WithCancel(parent)
	c.sigNotifyCtx, c.sigStop = signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	c.cancellationSource = newCancellationSource(
		contextSource{c.fileChangeCtx, cancelSourceFileChange},
		contextSource{c.sigNotifyCtx, cancelSourceSignal},
	)
	return c
}

// project file represents an .up file in a Pushup project context.
type projectFile struct {
	// path from cwd to the .up file
	path string
	// directory structure that may be part of the path of the .up file
	// like app/pages, app/layouts, or (empty string)
	projectFilesSubdir string
}

func (f *projectFile) relpath() string {
	path, err := filepath.Rel(f.projectFilesSubdir, f.path)
	if err != nil {
		panic("internal error: calling filepath.Rel(): " + err.Error())
	}
	return path
}

// projectFiles represents all the source files in a Pushup project.
type projectFiles struct {
	// list of .up page files
	pages []projectFile
	// list of .up layout files
	layouts []projectFile
	// paths to static files like JS, CSS, etc.
	static []projectFile
	// paths to user-contributed .go code
	gofiles []string // TODO(paulsmith): convert to projectFile
}

func (f *projectFiles) debug() {
	fmt.Println("pages:")
	for _, p := range f.pages {
		fmt.Printf("\t%v\n", p)
	}
	fmt.Println("layouts:")
	for _, p := range f.layouts {
		fmt.Printf("\t%v\n", p)
	}
	fmt.Println("static:")
	for _, p := range f.static {
		fmt.Printf("\t%v\n", p)
	}
	fmt.Println("gofiles:")
	for _, p := range f.gofiles {
		fmt.Printf("\t%s\n", p)
	}
}

func findProjectFiles(appDir string) (*projectFiles, error) {
	pf := new(projectFiles)

	layoutsDir := filepath.Join(appDir, "layouts")
	{
		entries, err := os.ReadDir(layoutsDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("invalid Pushup project directory structure: couldn't find `layouts` subdir")
			} else {
				return nil, fmt.Errorf("reading app layouts directory: %w", err)
			}
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), upFileExt) {
				path := filepath.Join(layoutsDir, entry.Name())
				pfile := projectFile{path: path, projectFilesSubdir: layoutsDir}
				pf.layouts = append(pf.layouts, pfile)
			}
		}
	}

	pagesDir := filepath.Join(appDir, "pages")
	{
		if err := fs.WalkDir(os.DirFS(pagesDir), ".", func(path string, d fs.DirEntry, _ error) error {
			if !d.IsDir() && filepath.Ext(path) == upFileExt {
				pfile := projectFile{path: filepath.Join(pagesDir, path), projectFilesSubdir: pagesDir}
				pf.pages = append(pf.pages, pfile)
			}
			return nil
		}); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("invalid Pushup project directory structure: couldn't find `pages` subdir")
			} else {
				return nil, err
			}
		}
	}

	pkgDir := filepath.Join(appDir, "pkg")
	{
		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("invalid Pushup project directory structure: couldn't find `pkg` subdir")
			} else {
				return nil, fmt.Errorf("reading app pkg directory: %w", err)
			}
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
				path := filepath.Join(pkgDir, entry.Name())
				pf.gofiles = append(pf.gofiles, path)
			}
		}
	}

	staticDir := filepath.Join(appDir, "static")
	{
		if err := fs.WalkDir(os.DirFS(staticDir), ".", func(path string, d fs.DirEntry, _ error) error {
			if !d.IsDir() {
				path = filepath.Join(staticDir, path)
				pf.static = append(pf.static, projectFile{path: path, projectFilesSubdir: staticDir})
			}
			return nil
		}); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("invalid Pushup project directory structure: couldn't find `static` dir")
			} else {
				return nil, fmt.Errorf("walking static dir: %w", err)
			}
		}
	}

	return pf, nil
}

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

	// "compile" user Go code
	for _, path := range c.files.gofiles {
		if err := copyFile(filepath.Join(c.outDir, filepath.Base(path)), path); err != nil {
			return fmt.Errorf("copying Go package file %s: %w", path, err)
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

type layout struct {
	imports []importDecl
	nodes   []node
}

func newLayoutFromTree(tree *syntaxTree) (*layout, error) {
	layout := &layout{}
	n := 0
	var f inspector = func(e node) bool {
		switch e := e.(type) {
		case *nodeImport:
			layout.imports = append(layout.imports, e.decl)
		default:
			layout.nodes = append(layout.nodes, e)
			n++
		}
		return false
	}
	inspect(nodeList(tree.nodes), f)
	layout.nodes = layout.nodes[:n]
	return layout, nil
}

type layoutCodeGen struct {
	layout  *layout
	pfile   projectFile
	imports map[importDecl]bool

	// source code of .up file, needed for mapping line numbers back to
	// original source in stack traces.
	source string

	// buffer for the package clauses and import declarations at the top of
	// a Go source file.
	outb bytes.Buffer

	// buffer for the main body of a Go source file, i.e., the top-level
	// declarations.
	bodyb bytes.Buffer

	ioWriterVar           string
	lineDirectivesEnabled bool
}

func newLayoutCodeGen(layout *layout, pfile projectFile, source string) *layoutCodeGen {
	l := &layoutCodeGen{
		layout:                layout,
		pfile:                 pfile,
		source:                source,
		imports:               make(map[importDecl]bool),
		ioWriterVar:           "w",
		lineDirectivesEnabled: true,
	}
	for _, im := range layout.imports {
		l.imports[im] = true
	}
	return l
}

func (c *layoutCodeGen) lineNo(s span) int {
	return lineCount(c.source[:s.start+1])
}

func (g *layoutCodeGen) outPrintf(format string, args ...any) {
	fmt.Fprintf(&g.outb, format, args...)
}

func (g *layoutCodeGen) bodyPrintf(format string, args ...any) {
	fmt.Fprintf(&g.bodyb, format, args...)
}

func (g *layoutCodeGen) used(path ...string) {
	for _, p := range path {
		g.imports[importDecl{path: strconv.Quote(p), pkgName: ""}] = true
	}
}

func (g *layoutCodeGen) nodeLineNo(e node) {
	if g.lineDirectivesEnabled {
		g.emitLineDirective(g.lineNo(e.Pos()))
	}
}

func (g *layoutCodeGen) emitLineDirective(n int) {
	g.bodyPrintf("//line %s:%d\n", g.pfile.relpath(), n)
}

func (g *layoutCodeGen) generate() {
	nodes := g.layout.nodes
	g.genNode(nodeList(nodes))
}

func (g *layoutCodeGen) genNode(n node) {
	var f inspector
	f = func(e node) bool {
		// TODO(paulsmith): these could be functions so they can be reused
		switch e := e.(type) {
		case *nodeLiteral:
			g.used("io")
			g.nodeLineNo(e)
			g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(e.str))
		case *nodeElement:
			g.used("io")
			g.nodeLineNo(e)
			f(nodeList(e.startTagNodes))
			f(nodeList(e.children))
			g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(e.tag.end()))
			return false
		case *nodeGoStrExpr:
			g.nodeLineNo(e)
			g.bodyPrintf("printEscaped(%s, %s)\n", g.ioWriterVar, e.expr)
		case *nodeGoCode:
			if e.context != inlineGoCode {
				panic(fmt.Sprintf("internal error: expected inlineGoCode, got %v", e.context))
			}
			srcLineNo := g.lineNo(e.Pos())
			lines := strings.Split(e.code, "\n")
			for _, line := range lines {
				// FIXME(paulsmith): leaky abstraction
				if g.lineDirectivesEnabled {
					g.emitLineDirective(srcLineNo)
				}
				g.bodyPrintf("%s\n", line)
				srcLineNo++
			}
		case *nodeIf:
			g.bodyPrintf("if %s {\n", e.cond.expr)
			f(e.then)
			if e.alt == nil {
				g.bodyPrintf("}\n")
			} else {
				g.bodyPrintf("} else {\n")
				f(e.alt)
				g.bodyPrintf("}\n")
			}
			return false
		case nodeList:
			for _, x := range e {
				f(x)
			}
			return false
		case *nodeFor:
			g.bodyPrintf("for %s {\n", e.clause.code)
			f(e.block)
			g.bodyPrintf("}\n")
			return false
		case *nodeBlock:
			f(nodeList(e.nodes))
			return false
		case *nodeSection:
			f(e.block)
			return false
		case *nodePartial:
			// FIXME(paulsmith): prune these out in newLayoutFromTree
			panic("partials are not allowed in layouts")
		case *nodeLayout:
			// nothing to do
		case *nodeImport:
			// nothing to do
		}
		return true
	}
	inspect(n, f)
}

const methodReceiverName = "up"

func genCodeLayout(g *layoutCodeGen) ([]byte, error) {
	// FIXME(paulsmith): need way to specify this as user
	packageName := "build"

	g.outPrintf("// this file is mechanically generated, do not edit!\n")
	g.outPrintf("// version: ")
	printVersion(&g.outb)
	g.outPrintf("\n")
	g.outPrintf("package %s\n\n", packageName)

	type field struct {
		name string
		typ  string
	}

	fields := []field{}

	typename := generatedTypename(g.pfile, upFileLayout)

	g.bodyPrintf("type %s struct {\n", typename)
	for _, field := range fields {
		g.bodyPrintf("%s %s\n", field.name, field.typ)
	}
	g.bodyPrintf("}\n")

	g.bodyPrintf("func (%s *%s) buildCliArgs() []string {\n", methodReceiverName, typename)
	g.bodyPrintf("  return %#v\n", os.Args)
	g.bodyPrintf("}\n\n")

	g.bodyPrintf("func init() {\n")
	g.bodyPrintf("  layouts[\"%s\"] = new(%s)\n", layoutName(g.pfile.relpath()), typename)
	g.bodyPrintf("}\n\n")

	g.used("net/http", "html/template")
	g.bodyPrintf("func (%s *%s) Respond(w http.ResponseWriter, req *http.Request, sections map[string]chan template.HTML) error {\n", methodReceiverName, typename)

	// sections support
	g.used("html/template")
	g.bodyPrintf(`
sectionDefined := func(name string) bool {
	_, ok := sections[name]
	return ok
}
_ = sectionDefined

outputSection := func(name string) template.HTML {
	return <-sections[name]
}
`)

	// Make a new scope for the user's code block and HTML. This will help (but not fully prevent)
	// name collisions with the surrounding code.
	g.bodyPrintf("\n// Begin user Go code and HTML\n")
	g.bodyPrintf("{\n")

	g.generate()

	// Close the scope we started for the user code and HTML.
	g.bodyPrintf("// End user Go code and HTML\n")
	g.bodyPrintf("}\n")

	g.bodyPrintf("return nil\n")
	g.bodyPrintf("}\n")

	g.outPrintf("import (\n")
	for decl, ok := range g.imports {
		if ok {
			if decl.pkgName != "" {
				g.outPrintf("%s ", decl.pkgName)
			}
			g.outPrintf("%s\n", decl.path)
		}
	}
	g.outPrintf(")\n\n")

	raw, err := io.ReadAll(io.MultiReader(&g.outb, &g.bodyb))
	if err != nil {
		return nil, fmt.Errorf("reading all buffers: %w", err)
	}

	formatted, err := format.Source(raw)
	if err != nil {
		return nil, fmt.Errorf("gofmt the generated code: %w", err)
	}

	return formatted, nil
}

// page represents a Pushup page that has been parsed and is ready for code
// generation.
type page struct {
	layout   string
	imports  []importDecl
	handler  *nodeGoCode
	nodes    []node
	sections map[string]*nodeBlock

	// partials is a list of all top-level inline partials in this page.
	partials []*partial
}

// partial represents an inline partial in a Pushup page.
type partial struct {
	node     node
	name     string
	parent   *partial
	children []*partial
}

// urlpath produces the URL path segment for the partial. this takes in to
// account its ancestor partials, so nested partials have the full path from
// their containing inline partials. note that the returned string is not
// prefixed with the host page's URL path.
func (p *partial) urlpath() string {
	segments := []string{p.name}
	for parent := p.parent; parent != nil; parent = parent.parent {
		segments = append([]string{parent.name}, segments...)
	}
	return strings.Join(segments, "/")
}

// newPageFromTree produces a page which is the main prepared object for code
// generation. this requires walking the syntax tree and reorganizing things
// somewhat to make them easier to access. some node types are encountered
// sequentially in the source file, but need to be reorganized for access in
// the code generator.
func newPageFromTree(tree *syntaxTree) (*page, error) {
	page := &page{
		layout:   "default",
		sections: make(map[string]*nodeBlock),
	}

	layoutSet := false
	n := 0
	var err error

	// this pass over the syntax tree nodes enforces invariants (only one
	// handler may be declared per page, layout may only be set once) and
	// aggregates imports and sections for easier access in the subsequent
	// code generation phase. as a result, some nodes are removed from the
	// tree.
	var f inspector
	f = func(e node) bool {
		switch e := e.(type) {
		case *nodeImport:
			page.imports = append(page.imports, e.decl)
		case *nodeLayout:
			if layoutSet {
				err = fmt.Errorf("layout already set as %q", page.layout)
				return false
			}
			if e.name == "!" {
				page.layout = ""
			} else {
				page.layout = e.name
			}
			layoutSet = true
		case *nodeGoCode:
			if e.context == handlerGoCode {
				if page.handler != nil {
					err = fmt.Errorf("only one handler per page can be defined")
					return false
				}
				page.handler = e
			} else {
				tree.nodes[n] = e
				n++
			}
		case nodeList:
			for _, x := range e {
				f(x)
			}
		case *nodeSection:
			page.sections[e.name] = e.block
		default:
			tree.nodes[n] = e
			n++
		}
		// don't recurse into child nodes
		return false
	}
	inspect(nodeList(tree.nodes), f)
	if err != nil {
		return nil, err
	}

	page.nodes = tree.nodes[:n]

	// this pass is for inline partials. it needs to be separate because the
	// traversal of the tree is slightly different than the pass above.
	{
		var currentPartial *partial
		var f inspector
		f = func(e node) bool {
			switch e := e.(type) {
			case *nodeLiteral:
			case *nodeElement:
				f(nodeList(e.startTagNodes))
				f(nodeList(e.children))
				return false
			case *nodeGoStrExpr:
			case *nodeGoCode:
			case *nodeIf:
				f(e.then)
				if e.alt != nil {
					f(e.alt)
				}
				return false
			case nodeList:
				for _, x := range e {
					f(x)
				}
				return false
			case *nodeFor:
				f(e.block)
				return false
			case *nodeBlock:
				f(nodeList(e.nodes))
				return false
			case *nodeSection:
				f(e.block)
				return false
			case *nodePartial:
				p := &partial{node: e, name: e.name, parent: currentPartial}
				if currentPartial != nil {
					currentPartial.children = append(currentPartial.children, p)
				}
				prevPartial := currentPartial
				currentPartial = p
				f(e.block)
				currentPartial = prevPartial
				page.partials = append(page.partials, p)
				return false
			case *nodeLayout:
				// nothing to do
			case *nodeImport:
				// nothing to do
			}
			return false
		}
		inspect(nodeList(page.nodes), f)
	}

	return page, nil
}

type pageCodeGen struct {
	page    *page
	pfile   projectFile
	source  string
	imports map[importDecl]bool

	// buffer for the package clauses and import declarations at the top of
	// a Go source file.
	outb bytes.Buffer

	// buffer for the main body of a Go source file, i.e., the top-level
	// declarations.
	bodyb bytes.Buffer

	ioWriterVar           string
	lineDirectivesEnabled bool
}

func newPageCodeGen(page *page, pfile projectFile, source string) *pageCodeGen {
	g := &pageCodeGen{
		page:                  page,
		pfile:                 pfile,
		source:                source,
		imports:               make(map[importDecl]bool),
		ioWriterVar:           "w",
		lineDirectivesEnabled: true,
	}
	for _, im := range page.imports {
		g.imports[im] = true
	}
	return g
}

func (g *pageCodeGen) used(path ...string) {
	for _, p := range path {
		g.imports[importDecl{path: strconv.Quote(p), pkgName: ""}] = true
	}
}

func (g *pageCodeGen) nodeLineNo(e node) {
	if g.lineDirectivesEnabled {
		g.emitLineDirective(g.lineNo(e.Pos()))
	}
}

func (c *pageCodeGen) lineNo(s span) int {
	return lineCount(c.source[:s.start+1])
}

func (g *pageCodeGen) emitLineDirective(n int) {
	g.bodyPrintf("//line %s:%d\n", g.pfile.relpath(), n)
}

func (g *pageCodeGen) outPrintf(format string, args ...any) {
	fmt.Fprintf(&g.outb, format, args...)
}

func (g *pageCodeGen) bodyPrintf(format string, args ...any) {
	fmt.Fprintf(&g.bodyb, format, args...)
}

func (g *pageCodeGen) generate() {
	nodes := g.page.nodes
	g.genNode(nodeList(nodes))
}

func (g *pageCodeGen) genNode(n node) {
	var f inspector
	f = func(e node) bool {
		switch e := e.(type) {
		case *nodeLiteral:
			g.used("io")
			g.nodeLineNo(e)
			g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(e.str))
		case *nodeElement:
			g.used("io")
			g.nodeLineNo(e)
			f(nodeList(e.startTagNodes))
			f(nodeList(e.children))
			g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(e.tag.end()))
			return false
		case *nodeGoStrExpr:
			g.nodeLineNo(e)
			g.bodyPrintf("printEscaped(%s, %s)\n", g.ioWriterVar, e.expr)
		case *nodeGoCode:
			if e.context != inlineGoCode {
				panic("internal error: expected inlineGoCode")
			}
			srcLineNo := g.lineNo(e.Pos())
			lines := strings.Split(e.code, "\n")
			for _, line := range lines {
				if g.lineDirectivesEnabled {
					g.emitLineDirective(srcLineNo)
				}
				g.bodyPrintf("%s\n", line)
				srcLineNo++
			}
		case *nodeIf:
			g.bodyPrintf("if %s {\n", e.cond.expr)
			f(e.then)
			if e.alt == nil {
				g.bodyPrintf("}\n")
			} else {
				g.bodyPrintf("} else {\n")
				f(e.alt)
				g.bodyPrintf("}\n")
			}
			return false
		case nodeList:
			for _, x := range e {
				f(x)
			}
			return false
		case *nodeFor:
			g.bodyPrintf("for %s {\n", e.clause.code)
			f(e.block)
			g.bodyPrintf("}\n")
			return false
		case *nodeBlock:
			f(nodeList(e.nodes))
			return false
		case *nodeSection:
			f(e.block)
			return false
		case *nodePartial:
			f(e.block)
			return false
		case *nodeLayout:
			// nothing to do
		case *nodeImport:
			// nothing to do
		}
		return true
	}
	inspect(n, f)
}

// NOTE(paulsmith): per DOM spec, "In tree order is preorder, depth-first traversal of a tree."

func (g *pageCodeGen) genNodePartial(n node, p *partial) {
	var f inspector
	var state int
	const (
		stateStart int = iota
		stateInPartialScope
	)
	state = stateStart
	f = func(n node) bool {
		if n != nil {
			switch n := n.(type) {
			case *nodeLiteral:
				if state == stateInPartialScope {
					g.used("io")
					g.nodeLineNo(n)
					g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(n.str))
				}
			case *nodeElement:
				if state == stateInPartialScope {
					g.used("io")
					g.nodeLineNo(n)
					f(nodeList(n.startTagNodes))
				}
				f(nodeList(n.children))
				if state == stateInPartialScope {
					g.bodyPrintf("io.WriteString(%s, %s)\n", g.ioWriterVar, strconv.Quote(n.tag.end()))
				}
				return false
			case *nodePartial:
				if n == p.node {
					state = stateInPartialScope
				}
				f(n.block)
				state = stateStart
				return false
			case *nodeGoStrExpr:
				if state == stateInPartialScope {
					g.nodeLineNo(n)
					g.bodyPrintf("printEscaped(%s, %s)\n", g.ioWriterVar, n.expr)
				}
			case *nodeFor:
				g.bodyPrintf("for %s {\n", n.clause.code)
				f(n.block)
				g.bodyPrintf("}\n")
				return false
			case *nodeIf:
				g.bodyPrintf("if %s {\n", n.cond.expr)
				f(n.then)
				if n.alt == nil {
					g.bodyPrintf("}\n")
				} else {
					g.bodyPrintf("} else {\n")
					f(n.alt)
					g.bodyPrintf("}\n")
				}
				return false
			case *nodeGoCode:
				if n.context != inlineGoCode {
					panic("internal error: expected inlineGoCode")
				}
				srcLineNo := g.lineNo(n.Pos())
				lines := strings.Split(n.code, "\n")
				for _, line := range lines {
					if g.lineDirectivesEnabled {
						g.emitLineDirective(srcLineNo)
					}
					g.bodyPrintf("%s\n", line)
					srcLineNo++
				}
			case nodeList:
				for _, x := range n {
					f(x)
				}
			case *nodeBlock:
				for _, x := range n.nodes {
					f(x)
				}
			default:
				panic(fmt.Sprintf("internal error: unhandled node type %T", n))
			}
		}
		return false
	}
	inspect(n, f)
}

func genCodePage(g *pageCodeGen) ([]byte, error) {
	type initRoute struct {
		typename string
		route    string
		role     string
	}
	var inits []initRoute

	// FIXME(paulsmith): need way to specify this as user
	packageName := "build"

	g.outPrintf("// this file is mechanically generated, do not edit!\n")
	g.outPrintf("// version: ")
	printVersion(&g.outb)
	g.outPrintf("\n")
	g.outPrintf("package %s\n\n", packageName)

	type field struct {
		name string
		typ  string
	}

	// main page
	{
		typename := generatedTypename(g.pfile, upFilePage)
		route := routeForPage(g.pfile.relpath())
		inits = append(inits, initRoute{typename: typename, route: route, role: "routePage"})

		g.bodyPrintf("type %s struct {\n", typename)
		fields := []field{}
		for _, field := range fields {
			g.bodyPrintf("%s %s\n", field.name, field.typ)
		}
		g.bodyPrintf("}\n")

		g.bodyPrintf("func (%s *%s) buildCliArgs() []string {\n", methodReceiverName, typename)
		g.bodyPrintf("  return %#v\n", os.Args)
		g.bodyPrintf("}\n\n")

		g.used("net/http")
		g.bodyPrintf("func (%s *%s) Respond(w http.ResponseWriter, req *http.Request) error {\n", methodReceiverName, typename)

		// NOTE(paulsmith): we might want to encapsulate this in its own
		// function/method, but would have to figure out the interplay between
		// user code and control flow, i.e., return an error if the handler
		// wants to skip rendering, redirect, etc.
		if h := g.page.handler; h != nil {
			srcLineNo := g.lineNo(h.Pos())
			lines := strings.Split(h.code, "\n")
			for _, line := range lines {
				if g.lineDirectivesEnabled {
					g.emitLineDirective(srcLineNo)
				}
				g.bodyPrintf("  %s\n", line)
				srcLineNo++
			}
		}

		g.used("html/template")
		g.bodyPrintf("// sections\n")
		g.bodyPrintf("sections := make(map[string]chan template.HTML)\n")
		g.bodyPrintf("sections[\"contents\"] = make(chan template.HTML)\n")
		for name := range g.page.sections {
			g.bodyPrintf("sections[%s] = make(chan template.HTML)\n", strconv.Quote(name))
		}

		// TODO(paulsmith): this is where a flag that could conditionally toggle the rendering
		// of the layout could go - maybe a special header in request object?
		g.used("log", "sync", "context", "time")
		g.bodyPrintf(
			`
			var wg sync.WaitGroup
			layout := getLayout("%s")
			ctx, cancel := context.WithTimeout(req.Context(), time.Second * 5)
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer cancel()
				if err := layout.Respond(w, req.WithContext(ctx), sections); err != nil {
					log.Printf("error responding with layout: %%v", err)
					panic(err)
				}
			}()
		`, g.page.layout)

		// Make a new scope for the user's code block and HTML. This will help (but not fully prevent)
		// name collisions with the surrounding code.
		g.bodyPrintf("\n// Begin user Go code and HTML\n")
		g.bodyPrintf("{\n")

		g.bodyPrintf("var panicked any\n")
		// render the main body contents
		// TODO(paulsmith) could do these as a incremental stream
		// so the receiving end is just pulling individual chunks off
		g.bodyPrintf("wg.Add(1)\n")
		g.bodyPrintf("go func() {\n")
		g.bodyPrintf("  defer wg.Done()\n")
		g.bodyPrintf("  defer func() {\n")
		g.bodyPrintf("    if r := recover(); r != nil {\n")
		g.bodyPrintf("      if panicked == nil {\n")
		g.bodyPrintf("	      cancel()\n")
		g.bodyPrintf("	      panicked = r\n")
		g.bodyPrintf("	    }\n")
		g.bodyPrintf("    }\n")
		g.bodyPrintf("  }()\n")
		g.used("bytes", "html/template")
		save := g.ioWriterVar
		g.ioWriterVar = "b"
		g.bodyPrintf("  %s := new(bytes.Buffer)\n", g.ioWriterVar)
		g.generate()
		g.bodyPrintf("  sections[\"contents\"] <- template.HTML(b.String())\n")
		g.bodyPrintf("}()\n\n")
		g.ioWriterVar = save

		for name, block := range g.page.sections {
			save := g.ioWriterVar
			g.ioWriterVar = "b"
			g.bodyPrintf("wg.Add(1)\n")
			g.bodyPrintf("go func() {\n")
			g.bodyPrintf("  defer wg.Done()\n")
			g.bodyPrintf("  defer func() {\n")
			g.bodyPrintf("    if r := recover(); r != nil {\n")
			g.bodyPrintf("      if panicked != nil {\n")
			g.bodyPrintf("	      cancel()\n")
			g.bodyPrintf("	      panicked = r\n")
			g.bodyPrintf("	    }\n")
			g.bodyPrintf("    }\n")
			g.bodyPrintf("  }()\n")
			g.bodyPrintf("  %s := new(bytes.Buffer)\n", g.ioWriterVar)
			g.genNode(block)
			g.bodyPrintf("  sections[%s] <- template.HTML(b.String())\n", strconv.Quote(name))
			g.bodyPrintf("}()\n")
			g.ioWriterVar = save
		}

		// Wait for layout to finish rendering
		g.bodyPrintf("wg.Wait()\n")

		// Check if any of the goroutines panicked
		g.bodyPrintf("if panicked != nil {\n")
		g.bodyPrintf("  close(sections[\"contents\"])\n")
		for name := range g.page.sections {
			g.bodyPrintf("  close(sections[%s])\n", strconv.Quote(name))
		}
		g.used("fmt")
		g.bodyPrintf("  return fmt.Errorf(\"goroutine panicked: %%v\", panicked)\n")
		g.bodyPrintf("}\n")

		// Close the scope we started for the user code and HTML.
		g.bodyPrintf("// End user Go code and HTML\n")
		g.bodyPrintf("}\n")

		// return from Respond()
		g.bodyPrintf("return nil\n")
		g.bodyPrintf("}\n")
	}

	for _, partial := range g.page.partials {
		typename := generatedTypenamePartial(partial, g.pfile)
		route := routeForPartial(g.pfile.relpath(), partial.urlpath())
		inits = append(inits, initRoute{typename: typename, route: route, role: "routePartial"})

		g.bodyPrintf("type %s struct {\n", typename)
		fields := []field{}
		for _, field := range fields {
			g.bodyPrintf("%s %s\n", field.name, field.typ)
		}
		g.bodyPrintf("}\n")

		g.used("net/http")
		g.bodyPrintf("func (%s *%s) Respond(w http.ResponseWriter, req *http.Request) error {\n", methodReceiverName, typename)

		// NOTE(paulsmith): we might want to encapsulate this in its own
		// function/method, but would have to figure out the interplay between
		// user code and control flow, i.e., return an error if the handler
		// wants to skip rendering, redirect, etc.
		if h := g.page.handler; h != nil {
			srcLineNo := g.lineNo(h.Pos())
			lines := strings.Split(h.code, "\n")
			for _, line := range lines {
				if g.lineDirectivesEnabled {
					g.emitLineDirective(srcLineNo)
				}
				g.bodyPrintf("  %s\n", line)
				srcLineNo++
			}
		}

		// Make a new scope for the user's code block and HTML. This will help (but not fully prevent)
		// name collisions with the surrounding code.
		g.bodyPrintf("// Begin user Go code and HTML\n")
		g.bodyPrintf("{\n")

		// FIXME(paulsmith): need to generate code for everything but emitting
		// top-level page values to the output
		g.genNodePartial(nodeList(g.page.nodes), partial)

		// Close the scope we started for the user code and HTML.
		g.bodyPrintf("// End user Go code and HTML\n")
		g.bodyPrintf("}\n")

		g.bodyPrintf("return nil\n")
		g.bodyPrintf("}\n")
	}

	g.bodyPrintf("\nfunc init() {\n")
	for _, initRoute := range inits {
		g.bodyPrintf("  routes.add(%s, new(%s), %s)\n", strconv.Quote(initRoute.route), initRoute.typename, initRoute.role)
	}
	g.bodyPrintf("}\n\n")

	// we write out imports at the end because we need to know what was
	// actually used by the body code
	g.outPrintf("import (\n")
	for decl, ok := range g.imports {
		if ok {
			if decl.pkgName != "" {
				g.outPrintf("%s ", decl.pkgName)
			}
			g.outPrintf("%s\n", decl.path)
		}
	}
	g.outPrintf(")\n\n")

	raw, err := io.ReadAll(io.MultiReader(&g.outb, &g.bodyb))
	if err != nil {
		return nil, fmt.Errorf("reading all buffers: %w", err)
	}

	formatted, err := format.Source(raw)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil, fmt.Errorf("gofmt the generated code: %w", err)
	}

	return formatted, nil
}

func watchForReload(ctx context.Context, cancel context.CancelFunc, root string, reload chan struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(fmt.Errorf("creating new fsnotify watcher: %v", err))
	}

	go debounceEvents(ctx, 125*time.Millisecond, watcher, func(event fsnotify.Event) {
		if !reloadableFilename(event.Name) {
			return
		}
		if isDir(event.Name) {
			if err := watchDirRecursively(watcher, event.Name); err != nil {
				panic(err)
			}
			return
		}
		log.Printf("change detected in project directory, reloading")
		cancel()
		stopWatching(watcher)
		reload <- struct{}{}
	})

	if err := watchDirRecursively(watcher, root); err != nil {
		panic(fmt.Errorf("adding dir to watch: %w", err))
	}
}

func stopWatching(watcher *fsnotify.Watcher) {
	for _, name := range watcher.WatchList() {
		watcher.Remove(name)
	}
}

// reloadableFilename tests whether the file is one we want to trigger a reload
// from if it is modified. it tries not to cause a lot of unnecessary reloads
// by ignoring temporary files from editors like vim and Emacs.
func reloadableFilename(path string) bool {
	ext := filepath.Ext(path)
	// ignore vim swap files: .swp, .swo, .swn, etc
	if len(ext) == 4 && strings.HasPrefix(ext, ".sw") {
		return false
	}
	// ignore vim and Emacs backup files
	if strings.HasSuffix(ext, "~") {
		return false
	}
	// ignore Emacs autosave files
	if strings.HasPrefix(ext, "#") && strings.HasSuffix(ext, "#") {
		return false
	}
	return true
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		log.Printf("error stat'ing path %s, skipping", path)
		return false
	}
	return fi.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

func watchDirRecursively(watcher *fsnotify.Watcher, root string) error {
	err := fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, _ error) error {
		if d.IsDir() {
			path = filepath.Join(root, path)
			if err := watcher.Add(path); err != nil {
				return fmt.Errorf("adding path %s to watch: %w", path, err)
			}
			log.Printf("adding %s to watch", path)
		}
		return nil
	})
	return err
}

func startReloadRevProxy(socketPath string, buildComplete *sync.Cond, port string) error {
	// FIXME(paulsmith): addr should be a command line flag or env var, here
	// and elsewhere
	addr := "0.0.0.0:" + port
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		return fmt.Errorf("listening to port: %w", err)
	}

	target, err := url.Parse("http://" + addr)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}
	proxy.ModifyResponse = modifyResponseAddDevReload

	reloadHandler := new(devReloader)
	reloadHandler.complete = buildComplete
	reloadHandler.verboseLogging = os.Getenv("VERBOSE") != ""

	mux := http.NewServeMux()
	mux.Handle("/", proxy)
	mux.Handle("/--dev-reload", reloadHandler)

	srv := http.Server{Handler: mux}
	// FIXME(paulsmith): shutdown
	go srv.Serve(ln)
	fmt.Fprintf(os.Stdout, "\x1b[1;36m↑↑ PUSHUP DEV RELOADER ON http://%s ↑↑\x1b[0m\n", addr)
	return nil
}

func modifyResponseAddDevReload(res *http.Response) error {
	mediatype, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if err != nil {
		return fmt.Errorf("parsing MIME type: %w", err)
	}

	// FIXME(paulsmith): we might not want to skip injecting in the case of a
	// hx-boost link
	if mediatype == "text/html" {
		if res.Header.Get("Pushup-Partial") == "true" || res.Header.Get("HX-Response") == "true" {
			return nil
		}
		doc, err := appendDevReloaderScript(res.Body)
		if err != nil {
			return fmt.Errorf("appending dev reloading script: %w", err)
		}
		if err := res.Body.Close(); err != nil {
			return fmt.Errorf("closing proxied response body: %w", err)
		}

		var buf bytes.Buffer
		if err := html.Render(&buf, doc); err != nil {
			return fmt.Errorf("rendering modified HTML doc: %w", err)
		}

		res.Body = io.NopCloser(&buf)
		res.ContentLength = int64(buf.Len())
		res.Header.Set("Content-Length", strconv.Itoa(buf.Len()))
	}

	return nil
}

type devReloader struct {
	complete       *sync.Cond
	verboseLogging bool
}

func (d *devReloader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		panic("can't flush response so SSE not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	built := make(chan struct{})
	done := make(chan struct{})

	// FIXME(paulsmith): this probably leaks goroutines
	go func() {
		d.complete.L.Lock()
		d.complete.Wait()
		d.complete.L.Unlock()
		select {
		case built <- struct{}{}:
		case <-done:
			return
		}
	}()

loop:
	for {
		select {
		case <-built:
			w.Write([]byte("event: reload\ndata: \n\n"))
		case <-r.Context().Done():
			if d.verboseLogging {
				log.Printf("client disconnected")
			}
			close(done)
			break loop
		case <-time.After(1 * time.Second):
			w.Write([]byte(":keepalive\n\n"))
			flusher.Flush()
		}
	}
}

var devReloaderScript = `
if (!window.EventSource) {
	throw "Server-sent events not supported by this browser, live reloading disabled";
}

var source = new EventSource("/--dev-reload");

source.onmessage = e => {
	console.log("message:", e.data);
}

source.addEventListener("reload", () => {
	console.log("%c↑↑ Pushup server changed, reloading page ↑↑", "color: green");
	location.reload(true);
}, false);

source.addEventListener("open", e => {
	console.log("%c↑↑ Connection to Pushup server for dev mode reloading established ↑↑", "color: green");
}, false);

source.onerror = err => {
	console.error("SSE error:", err);
};
`

func appendDevReloaderScript(r io.Reader) (*html.Node, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			text := &html.Node{
				Type: html.TextNode,
				Data: devReloaderScript,
			}
			script := &html.Node{
				Type:     html.ElementNode,
				Data:     "script",
				DataAtom: atom.Script,
				Attr: []html.Attribute{
					{Key: "type", Val: "text/javascript"},
				},
			}
			script.AppendChild(text)
			n.AppendChild(script)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return doc, nil
}

func debounceEvents(ctx context.Context, interval time.Duration, watcher *fsnotify.Watcher, fn func(event fsnotify.Event)) {
	var mu sync.Mutex
	timers := make(map[string]*time.Timer)

	has := func(ev fsnotify.Event, op fsnotify.Op) bool {
		return ev.Op&op == op
	}

	for {
		select {
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("file watch error: %v", err)
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}
			if !has(ev, fsnotify.Create) && !has(ev, fsnotify.Write) {
				continue
			}
			mu.Lock()
			t, ok := timers[ev.Name]
			mu.Unlock()
			if !ok {
				t = time.AfterFunc(math.MaxInt64, func() {
					fn(ev)
					mu.Lock()
					defer mu.Unlock()
					delete(timers, ev.Name)
				})
				t.Stop()

				mu.Lock()
				timers[ev.Name] = t
				mu.Unlock()
			}
			t.Reset(interval)
		case <-ctx.Done():
			return
		}
	}
}

// cancellationSource implements the context.Context interface and allows a
// caller to distinguish between one of two possible contexts for which one was
// responsible for cancellation, by testing for identity against the `final'
// struct member.
type cancellationSource struct {
	a     contextSource
	b     contextSource
	final contextSource
	done  chan struct{}
	err   error
}

type contextSource struct {
	context.Context
	source cancelSourceID
}

type cancelSourceID int

const (
	cancelSourceFileChange cancelSourceID = iota
	cancelSourceSignal
)

func newCancellationSource(a contextSource, b contextSource) *cancellationSource {
	s := new(cancellationSource)
	s.a = a
	s.b = b
	s.done = make(chan struct{})
	go s.run()
	return s
}

func (s *cancellationSource) run() {
	select {
	case <-s.a.Done():
		s.final = s.a
		s.err = s.final.Err()
	case <-s.b.Done():
		s.final = s.b
		s.err = s.final.Err()
	case <-s.done:
		return
	}
	close(s.done)
}

func (s *cancellationSource) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (s *cancellationSource) Done() <-chan struct{} {
	return s.done
}

func (s *cancellationSource) Err() error {
	return s.err
}

var _ context.Context = (*cancellationSource)(nil)

func (s *cancellationSource) Value(key any) any {
	panic("not implemented") // TODO: Implement
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

//go:embed _runtime/pushup_support.go _runtime/cmd/main.go
var runtimeFiles embed.FS

// copyFileFS copies a file from an fs.FS and writes it to a file location on
// the local filesystem. src is the name of the file object in the FS. it
// assumes the directory for dest already exists.
func copyFileFS(fsys fs.FS, dest string, src string) error {
	f, err := fsys.Open(src)
	if err != nil {
		return fmt.Errorf("opening file from FS %s: %w", src, err)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", src, err)
	}

	if err := os.WriteFile(dest, b, 0664); err != nil {
		return fmt.Errorf("writing file %s: %w", dest, err)
	}

	return nil
}

type buildParams struct {
	projectName string
	pkgName     string
	// path to directory with the compiled Pushup project code
	compiledOutputDir string
	buildDir          string
}

// buildProject builds the Go program made up of the user's compiled .up
// files and .go code, as well as Pushup's library APIs.
func buildProject(_ context.Context, b buildParams) error {
	mainExeDir := filepath.Join(b.compiledOutputDir, "cmd", b.projectName)
	if err := os.MkdirAll(mainExeDir, 0755); err != nil {
		return fmt.Errorf("making directory for command: %w", err)
	}

	t := template.Must(template.ParseFS(runtimeFiles, filepath.Join("_runtime", "cmd", "main.go")))
	f, err := os.Create(filepath.Join(mainExeDir, "main.go"))
	if err != nil {
		return fmt.Errorf("creating main.go: %w", err)
	}
	if err := t.Execute(f, map[string]any{"ProjectPkg": b.pkgName}); err != nil {
		return fmt.Errorf("executing main.go template: %w", err)
	}
	f.Close()

	exeName := b.projectName + ".exe"
	args := []string{"build", "-o", filepath.Join(b.buildDir, "bin", exeName), filepath.Join(b.pkgName, "cmd", b.projectName)}
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("building project main executable: %w", err)
	}

	return nil
}

// runProject runs the generated Pushup project executable, taking a listener
// from the caller for its server. this is meant to be used primarily during
// development with `pushup run`, as a production deployment can merely deploy
// the executable and run it directly.
func runProject(ctx context.Context, exePath string, ln net.Listener) error {
	var file *os.File
	var err error
	switch ln := ln.(type) {
	case *net.TCPListener:
		file, err = ln.File()
	case *net.UnixListener:
		file, err = ln.File()
	default:
		panic(fmt.Sprintf("unsupported net listener type %T", ln))
	}
	if err != nil {
		return fmt.Errorf("getting file from Unix socket listener: %w", err)
	}

	cmd := exec.Command(exePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	sysProcAttr(cmd)
	cmd.ExtraFiles = []*os.File{file}
	cmd.Env = append(os.Environ(), "PUSHUP_LISTENER_FD=3")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting project main executable: %w", err)
	}

	g := new(errgroup.Group)
	done := make(chan struct{})

	g.Go(func() error {
		select {
		case <-ctx.Done():
		case <-done:
			return nil
		}
		// FIXME(paulsmith): don't like this interface
		if ctx, ok := ctx.(*pushupContext); ok {
			if ctx.final.source == cancelSourceFileChange {
				log.Printf("\x1b[35mFILE CHANGED\x1b[0m")
			} else if ctx.final.source == cancelSourceSignal {
				log.Printf("\x1b[34mSIGNAL TRAPPED\x1b[0m")
			} else {
				panic("unknown source of cancellation")
			}
		}
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGINT); err != nil {
			return fmt.Errorf("syscall kill: %w", err)
		}
		if !errors.Is(ctx.Err(), context.Canceled) {
			return ctx.Err()
		}
		return nil
	})

	g.Go(func() error {
		defer close(done)
		// NOTE(paulsmith): we have to wait() the child process(es) in any case,
		// regardless of how they were exited. this is also why there is a
		// `done' channel in this function, to signal to the other goroutine
		// waiting for context cancellation.
		if err := cmd.Wait(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				// FIXME(paulsmith): don't like this interface
				if _, ok := ctx.(*pushupContext); !ok {
					return fmt.Errorf("wait: %w", ee)
				}
			} else {
				return fmt.Errorf("wait: %w", ee)
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("errgroup: %w", err)
	}

	return nil
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false
		}
		panic(err)
	}
	return fi.IsDir()
}

type upFileType int

const (
	upFilePage upFileType = iota
	upFileLayout
)

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

// generatedTypename returns the name of the type of the Go struct that
// holds the generated code for the Pushup page and related methods.
func generatedTypename(pfile projectFile, ftype upFileType) string {
	relpath := pfile.relpath()
	ext := filepath.Ext(relpath)
	relpath = relpath[:len(relpath)-len(ext)]
	typename := typenameFromPath(relpath)
	var suffix string
	switch ftype {
	case upFilePage:
		suffix = "Page"
	case upFileLayout:
		suffix = "Layout"
	default:
		panic("unhandled file type")
	}
	result := typename + suffix
	return result
}

func generatedTypenamePartial(partial *partial, pfile projectFile) string {
	relpath := pfile.relpath()
	ext := filepath.Ext(relpath)
	relpath = relpath[:len(relpath)-len(ext)]
	typename := typenameFromPath(strings.Join([]string{relpath, partial.urlpath()}, "/"))
	result := typename + "Partial"
	return result
}

// routeForPage produces the URL path route from the name of the Pushup page.
// relpath is the path to the Pushup file, relative to its containing app
// directory in the Pushup project (so that part should not be part of the
// path).
func routeForPage(relpath string) string {
	var dirs []string
	dir := filepath.Dir(relpath)
	if dir != "." {
		dirs = strings.Split(dir, string([]rune{os.PathSeparator}))
	}
	file := filepath.Base(relpath)
	base := strings.TrimSuffix(file, filepath.Ext(file))
	var route string
	if base != "index" {
		dirs = append(dirs, base)
	}
	for i := range dirs {
		if strings.HasPrefix(dirs[i], "$") {
			dirs[i] = ":" + dirs[i][1:]
		}
	}
	route = "/" + strings.Join(dirs, "/")
	if base == "index" && route[len(route)-1] != '/' {
		// indexes always have a trailing slash
		route += "/"
	}
	return route
}

func routeForPartial(relpath string, partialUrlpath string) string {
	prefix := strings.TrimSuffix(relpath, filepath.Ext(relpath))
	if filepath.Base(prefix) == "index" {
		prefix = filepath.Dir(prefix)
	}
	route := routeForPage(prefix + "/" + partialUrlpath)
	return route
}

func layoutName(relpath string) string {
	ext := filepath.Ext(relpath)
	if ext != upFileExt {
		panic("internal error: unexpected file extension " + ext)
	}
	return strings.TrimSuffix(relpath, ext)
}

/* ------------------ start of syntax nodes -------------------------*/

// node represents a portion of the Pushup syntax, like a chunk of HTML,
// or a Go expression to be evaluated, or a control flow construct like `if'
// or `for'.
type node interface {
	Pos() span
}

type nodeList []node

func (n nodeList) Pos() span { return n[0].Pos() }

type visitor interface {
	visit(node) visitor
}

type inspector func(node) bool

func (f inspector) visit(n node) visitor {
	if f(n) {
		return f
	}
	return nil
}

func inspect(n node, f func(node) bool) {
	walk(inspector(f), n)
}

func walkNodeList(v visitor, list []node) {
	for _, n := range list {
		walk(v, n)
	}
}

func walk(v visitor, n node) {
	if v = v.visit(n); v == nil {
		return
	}

	switch n := n.(type) {
	case *nodeElement:
		walkNodeList(v, n.startTagNodes)
		walkNodeList(v, n.children)
	case *nodeLiteral:
		// no children
	case *nodeGoStrExpr:
		// no children
	case *nodeGoCode:
		// no children
	case *nodeIf:
		walk(v, n.cond)
		walk(v, n.then)
		if n.alt != nil {
			walk(v, n.alt)
		}
	case *nodeFor:
		walk(v, n.clause)
		walk(v, n.block)
	case *nodeBlock:
		walkNodeList(v, n.nodes)
	case *nodeSection:
		walk(v, n.block)
	case *nodeImport:
		// no children
	case *nodeLayout:
		// no children
	case nodeList:
		walkNodeList(v, n)
	case *nodePartial:
		walk(v, n.block)
	default:
		panic(fmt.Sprintf("unhandled type %T", n))
	}
	v.visit(nil)
}

type nodeLiteral struct {
	str string
	pos span
}

func (e nodeLiteral) Pos() span { return e.pos }

var _ node = (*nodeLiteral)(nil)

type nodeGoStrExpr struct {
	expr string
	pos  span
}

func (e nodeGoStrExpr) Pos() span { return e.pos }

var _ node = (*nodeGoStrExpr)(nil)

type goCodeContext int

const (
	inlineGoCode goCodeContext = iota
	handlerGoCode
)

type nodeGoCode struct {
	context goCodeContext
	code    string
	pos     span
}

func (e nodeGoCode) Pos() span { return e.pos }

var _ node = (*nodeGoCode)(nil)

type nodeIf struct {
	cond *nodeGoStrExpr
	then *nodeBlock
	alt  node
}

func (e nodeIf) Pos() span { return e.cond.pos }

var _ node = (*nodeIf)(nil)

type nodeFor struct {
	clause *nodeGoCode
	block  *nodeBlock
}

func (e nodeFor) Pos() span { return e.clause.pos }

type nodeSection struct {
	name  string
	pos   span
	block *nodeBlock
}

func (e nodeSection) Pos() span { return e.pos }

var _ node = (*nodeSection)(nil)

// nodePartial is a syntax tree node representing an inline partial in a Pushup
// page.
type nodePartial struct {
	name  string
	pos   span
	block *nodeBlock
}

func (e nodePartial) Pos() span { return e.pos }

var _ node = (*nodePartial)(nil)

// nodeBlock represents a block of nodes, i.e., a sequence of nodes that
// appear in order in the source syntax.
type nodeBlock struct {
	nodes []node
}

func (e *nodeBlock) Pos() span {
	// FIXME(paulsmith): span end all exprs
	return e.nodes[0].Pos()
}

var _ node = (*nodeBlock)(nil)

// nodeElement represents an HTML element, with a start tag, optional
// attributes, optional children, and an end tag.
type nodeElement struct {
	tag           tag
	startTagNodes []node
	children      []node
	pos           span
}

func (e nodeElement) Pos() span { return e.pos }

var _ node = (*nodeElement)(nil)

type nodeImport struct {
	decl importDecl
	pos  span
}

func (e nodeImport) Pos() span { return e.pos }

var _ node = (*nodeImport)(nil)

type nodeLayout struct {
	name string
	pos  span
}

func (e nodeLayout) Pos() span { return e.pos }

var _ node = (*nodeLayout)(nil)

/* ------------------ end of syntax nodes -------------------------*/

type span struct {
	start int
	end   int
}

func optimize(tree *syntaxTree) *syntaxTree {
	tree.nodes = coalesceLiterals(tree.nodes)
	return tree
}

// coalesceLiterals is an optimization that coalesces consecutive HTML literal
// nodes together by concatenating their strings together in a single node.
// TODO(paulsmith): further optimization could be had by descending in to child
// nodes, refactor this using inspect().
func coalesceLiterals(nodes []node) []node {
	// before := len(nodes)
	if len(nodes) > 0 {
		n := 0
		for range nodes[:len(nodes)-1] {
			this, thisOk := nodes[n].(*nodeLiteral)
			next, nextOk := nodes[n+1].(*nodeLiteral)
			if thisOk && nextOk && len(this.str) < 512 {
				this.str += next.str
				this.pos.end = next.pos.end
				nodes = append(nodes[:n+1], nodes[n+2:]...)
			} else {
				n++
			}
		}
		nodes = nodes[:n+1]
	}
	return nodes
}

func lineCount(s string) int {
	return strings.Count(s, "\n") + 1
}

func typenameFromPath(path string) string {
	path = strings.ReplaceAll(path, "$", "DollarSign_")
	buf := make([]rune, len(path))
	i := 0
	wordBoundary := true
	for _, r := range path {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if wordBoundary {
				wordBoundary = false
				buf[i] = unicode.ToUpper(r)
			} else {
				buf[i] = r
			}
			i++
		} else {
			wordBoundary = true
		}
	}
	return string(buf[:i])
}

type importDecl struct {
	pkgName string
	path    string
}

type syntaxTree struct {
	nodes []node
}

func init() {
	if atom.Lookup([]byte("text")) != 0 {
		panic("expected <text> to not be a common HTML tag")
	}
}

func parse(source string) (tree *syntaxTree, err error) {
	p := newParser(source)
	defer func() {
		if e := recover(); e != nil {
			if se, ok := e.(syntaxError); ok {
				tree = nil
				err = se
			} else {
				panic(e)
			}
		}
	}()
	tree = p.htmlParser.parseDocument()
	return
}

// parser is the main Pushup parser. it is comprised of an HTML parser and a Go
// parser, and handles Pushup template language syntax, too. it starts in HTML
// mode, and switches to parsing Go code when it encounters the transition
// symbol.
type parser struct {
	src string
	// byte offset into src representing the maximum position read
	offset int

	htmlParser *htmlParser
	codeParser *codeParser
}

func newParser(source string) *parser {
	p := new(parser)
	p.src = source
	p.offset = 0
	p.htmlParser = &htmlParser{parser: p}
	p.codeParser = &codeParser{parser: p}
	return p
}

// remainingSource returns the source code starting from the internal byte
// offset all the way to the end.
func (p *parser) remainingSource() string {
	return p.sourceFrom(p.offset)
}

// sourceFrom returns the source code starting from the given byte offset. it
// returns the empty string if the offset is greater than the source code's
// length.
func (p *parser) sourceFrom(offset int) string {
	if len(p.src) >= offset {
		return p.src[offset:]
	}
	return ""
}

// advanceOffset advances the internal byte offset position by delta amount.
func (p *parser) advanceOffset(delta int) {
	p.offset += delta
}

// syntaxError represents a synax error in the Pushup template language.
type syntaxError struct {
	// err is the underlying error that caused this syntax error
	err error
	// lineNo and column are the positions in the source code where the
	// error occurred
	lineNo int
	column int
}

func (e syntaxError) Error() string {
	// TODO(paulsmith): add source file name
	return fmt.Sprintf("%d:%d: %s", e.lineNo, e.column, e.err.Error())
}

// errorf signals that a syntax error in the Pushup template language has been
// detected. The Pushup parser uses panic mode error handling, so a function
// calling the parser higher up in the call stack can recover from the panic
// and test for a syntax error (syntaxError type).
func (p *parser) errorf(format string, args ...any) {
	offset := p.offset
	if offset >= len(p.src) {
		offset = len(p.src) - 1
	}
	upToErr := p.src[:offset]
	lineNo := strings.Count(upToErr, "\n") + 1
	lastNL := strings.LastIndex(upToErr, "\n")
	column := p.offset + 1
	if lastNL > -1 {
		column = p.offset - lastNL
	}
	panic(syntaxError{fmt.Errorf(format, args...), lineNo, column})
}

// htmlParser is the Pushup HTML parser. It wraps the golang.org/x/net/html
// tokenizer, which is an HTML 5 specification-compliant parser. It changes
// control to the Go code parser (codeParser type) if it encounters the
// transition symbol in the course of tokenizing HTML documents.
type htmlParser struct {
	// a pointer to the main Pushup parser
	parser *parser

	// current token
	toktyp  html.TokenType
	tagname []byte
	err     error
	raw     string
	attrs   []*attr

	// the global parser offset at the beginning of a new token
	start int
}

func (p *htmlParser) errorf(format string, args ...any) {
	p.parser.errorf(format, args...)
}

func (p *htmlParser) advance() {
	// NOTE(paulsmith): we're re-creating a tokenizer each time through
	// the loop, with the starting point of the source text moved up by the
	// length of the previous token, in order to synchronize the position
	// in the source between the code parser and the HTML parser. this is
	// probably inefficient and could be done "better" and more efficiently
	// by reusing the tokenizer, as for sure it generates more garbage. but
	// would need to profile to see if this is actually a big problem to
	// end users, and in any case, it's only during compilation, so doesn't
	// impact the runtime web application.
	tokenizer := html.NewTokenizer(strings.NewReader(p.parser.remainingSource()))
	tokenizer.SetMaxBuf(0) // unlimited buffer size
	p.toktyp = tokenizer.Next()
	p.err = tokenizer.Err()
	p.raw = string(tokenizer.Raw())
	p.attrs = nil
	var hasAttr bool
	p.tagname, hasAttr = tokenizer.TagName()
	if hasAttr && p.err == nil {
		p.attrs, p.err = scanAttrs(p.raw)
	}
	p.start = p.parser.offset
	p.parser.advanceOffset(len(p.raw))
}

func isAllWhitespace(s string) bool {
	for s != "" {
		r, size := utf8.DecodeRuneInString(s)
		if !unicode.IsSpace(r) {
			return false
		}
		s = s[size:]
	}
	return true
}

func (p *htmlParser) skipWhitespace() []*nodeLiteral {
	var result []*nodeLiteral
	for p.toktyp == html.TextToken && isAllWhitespace(p.raw) {
		n := nodeLiteral{str: p.raw, pos: span{start: p.start, end: p.parser.offset}}
		result = append(result, &n)
		p.advance()
	}
	return result
}

// transition character: transitions the parser from HTML markup to Go code: ^
const (
	transSym    = '^'
	transSymStr = string(transSym)
	transSymEsc = transSymStr + transSymStr
)

func (p *htmlParser) parseAttributeNameOrValue(nameOrValue string, nameOrValueStartPos, nameOrValueEndPos int, pos int) ([]node, int) {
	var nodes []node
	if strings.ContainsRune(nameOrValue, transSym) {
		for pos < nameOrValueEndPos && strings.ContainsRune(nameOrValue, transSym) {
			if idx := strings.IndexRune(nameOrValue, transSym); idx > 0 {
				nodes = append(nodes, p.emitLiteralFromRange(pos, pos+idx))
				pos += idx
				nameOrValue = nameOrValue[idx:]
			}
			if strings.HasPrefix(nameOrValue, transSymStr+transSymStr) {
				nodes = append(nodes, p.emitLiteralFromRange(pos, pos+1))
				pos += 2
				nameOrValue = nameOrValue[2:]
			} else {
				pos++
				saveParser := p.parser
				p.parser = newParser(nameOrValue[1:])
				nodes = append(nodes, p.transition())
				bytesRead := p.parser.offset
				pos += bytesRead
				p.parser = saveParser
				nameOrValue = nameOrValue[bytesRead:]
			}
		}
	} else {
		nodes = append(nodes, p.emitLiteralFromRange(nameOrValueStartPos, nameOrValueEndPos))
		pos = nameOrValueEndPos
	}
	return nodes, pos
}

func (p *htmlParser) emitLiteralFromRange(start, end int) node {
	e := new(nodeLiteral)
	e.str = p.raw[start:end]
	e.pos.start = p.start + start
	e.pos.end = p.start + end
	return e
}

func (p *htmlParser) parseStartTag() []node {
	var nodes []node

	if len(p.attrs) == 0 {
		nodes = append(nodes, p.emitLiteralFromRange(0, len(p.raw)))
	} else {
		// bytesRead keeps track of how far we've parsed into this p.raw string
		bytesRead := 0

		for _, attr := range p.attrs {
			name := attr.name.string
			value := attr.value.string
			nameStartPos := int(attr.name.start)
			valStartPos := int(attr.value.start)
			nameEndPos := nameStartPos + len(name)
			valEndPos := valStartPos + len(value)

			// emit raw chars between tag name or last attribute and this
			// attribute
			if n := nameStartPos - bytesRead; n > 0 {
				nodes = append(nodes, p.emitLiteralFromRange(bytesRead, bytesRead+n))
				bytesRead += n
			}

			// emit attribute name
			nameNodes, newPos := p.parseAttributeNameOrValue(name, nameStartPos, nameEndPos, bytesRead)
			nodes = append(nodes, nameNodes...)
			bytesRead = newPos

			if valStartPos > bytesRead {
				// emit any chars, including equals and quotes, between
				// attribute name and attribute value, if any
				nodes = append(nodes, p.emitLiteralFromRange(bytesRead, valStartPos))
				bytesRead = valStartPos

				// emit attribute value
				valNodes, newPos := p.parseAttributeNameOrValue(value, valStartPos, valEndPos, bytesRead)
				nodes = append(nodes, valNodes...)
				bytesRead = newPos
			}
		}

		// emit anything from the last attribute to the close of the tag
		nodes = append(nodes, p.emitLiteralFromRange(bytesRead, len(p.raw)))
	}

	return nodes
}

func (p *htmlParser) emitLiteral() node {
	e := new(nodeLiteral)
	e.pos.start = p.start
	e.pos.end = p.parser.offset
	e.str = p.raw
	return e
}

func (p *htmlParser) parseDocument() *syntaxTree {
	tree := new(syntaxTree)

tokenLoop:
	for {
		p.advance()
		if p.toktyp == html.ErrorToken {
			if p.err == io.EOF {
				break tokenLoop
			} else {
				p.errorf("HTML tokenizer: %w", p.err)
			}
		}
		switch p.toktyp {
		case html.StartTagToken, html.SelfClosingTagToken:
			tree.nodes = append(tree.nodes, p.parseStartTag()...)
		case html.EndTagToken, html.DoctypeToken, html.CommentToken:
			tree.nodes = append(tree.nodes, p.emitLiteral())
		case html.TextToken:
			if idx := strings.IndexRune(p.raw, transSym); idx >= 0 {
				if escaped := strings.Index(p.raw, transSymEsc); escaped >= 0 {
					// it's an escaped transition symbol
					if escaped > 0 {
						// emit the leading text before the doubled escape
						e := new(nodeLiteral)
						e.pos.start = p.start
						e.pos.end = p.start + escaped
						e.str = p.raw[:escaped]
						tree.nodes = append(tree.nodes, e)
					}
					e := new(nodeLiteral)
					e.pos.start = p.start + escaped
					e.pos.end = p.start + escaped + 2
					e.str = transSymStr
					tree.nodes = append(tree.nodes, e)
					p.parser.offset = p.start + escaped + 2
				} else {
					// FIXME(paulsmith): clean this up!
					if strings.HasPrefix(p.raw[idx+1:], "layout") {
						s := p.raw[idx+1+len("layout"):]
						n := 0
						if len(s) < 1 || s[0] != ' ' {
							p.errorf(transSymStr + "layout must be followed by a space")
						}
						s = s[1:]
						n++
						e := new(nodeLayout)
						if len(s) > 0 && s[0] == '!' {
							e.name = "!"
							n++
						} else {
							var name []rune
							for {
								r, size := utf8.DecodeRuneInString(s)
								if r == 0 {
									break
								}
								if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '-' || r == '.' {
									name = append(name, r)
									s = s[size:]
									n += size
								} else {
									break
								}
							}
							e.name = string(name)
						}
						e.pos.start = p.start + idx + 1
						newOffset := e.pos.start + len("layout") + n
						e.pos.end = newOffset
						p.parser.offset = newOffset
						tree.nodes = append(tree.nodes, e)
					} else {
						newOffset := p.start + idx + 1
						p.parser.offset = newOffset
						leading := p.raw[:idx]
						if idx > 0 {
							var htmlNode nodeLiteral
							htmlNode.pos.start = p.start
							htmlNode.pos.end = p.start + len(leading)
							htmlNode.str = leading
							tree.nodes = append(tree.nodes, &htmlNode)
						}
						e := p.transition()
						// NOTE(paulsmith): this bubbles up nil due to parseImportKeyword,
						// the result of which we don't treat as a node in the syntax tree
						if e != nil {
							tree.nodes = append(tree.nodes, e)
						}
					}
				}
			} else {
				tree.nodes = append(tree.nodes, p.emitLiteral())
			}
		default:
			panic("")
		}
	}

	return tree
}

func (p *htmlParser) transition() node {
	codeParser := p.parser.codeParser
	codeParser.reset()
	e := codeParser.parseCode()
	return e
}

type tag struct {
	name  string
	attrs []*attr
}

func (t tag) String() string {
	if len(t.attrs) == 0 {
		return t.name
	}
	buf := bytes.NewBufferString(t.name)
	for _, a := range t.attrs {
		buf.WriteByte(' ')
		buf.WriteString(a.name.string)
		buf.WriteString(`="`)
		buf.WriteString(html.EscapeString(a.value.string))
		buf.WriteByte('"')
	}
	return buf.String()
}

func (t tag) start() string {
	return "<" + t.String() + ">"
}

func (t tag) end() string {
	return "</" + t.name + ">"
}

func newTag(tagname []byte, attrs []*attr) tag {
	return tag{name: string(tagname), attrs: attrs}
}

func (p *htmlParser) match(typ html.TokenType) bool {
	return p.toktyp == typ
}

func (p *htmlParser) parseElement() node {
	var result *nodeElement

	// FIXME(paulsmith): handle self-closing elements
	if !p.match(html.StartTagToken) {
		p.errorf("expected an HTML element start tag, got %s", p.toktyp)
	}

	result = new(nodeElement)
	result.tag = newTag(p.tagname, p.attrs)
	result.pos.start = p.parser.offset - len(p.raw)
	result.pos.end = p.parser.offset
	result.startTagNodes = p.parseStartTag()
	p.advance()

	result.children = p.parseChildren()

	if !p.match(html.EndTagToken) {
		p.errorf("expected an HTML element end tag, got %q", p.toktyp)
	}

	if result.tag.name != string(p.tagname) {
		p.errorf("expected </%s> end tag, got </%s>", result.tag.name, p.tagname)
	}

	// <text></text> elements are just for parsing
	if string(p.tagname) == "text" {
		return &nodeBlock{nodes: result.children}
	}

	return result
}

func (p *htmlParser) parseChildren() []node {
	var result []node // either *nodeElement or *nodeLiteral
	var elemStack []*nodeElement
loop:
	for {
		switch p.toktyp {
		case html.ErrorToken:
			if p.err == io.EOF {
				break loop
			} else {
				p.errorf("HTML tokenizer: %w", p.err)
			}
		case html.SelfClosingTagToken:
			elem := new(nodeElement)
			elem.tag = newTag(p.tagname, p.attrs)
			elem.pos.start = p.parser.offset - len(p.raw)
			elem.pos.end = p.parser.offset
			elem.startTagNodes = p.parseStartTag()
			p.advance()
			result = append(result, elem)
		case html.StartTagToken:
			elem := new(nodeElement)
			elem.tag = newTag(p.tagname, p.attrs)
			elem.pos.start = p.parser.offset - len(p.raw)
			elem.pos.end = p.parser.offset
			elem.startTagNodes = p.parseStartTag()
			p.advance()
			elem.children = p.parseChildren()
			result = append(result, elem)
			elemStack = append(elemStack, elem)
		case html.EndTagToken:
			if len(elemStack) == 0 {
				return result
			}
			elem := elemStack[len(elemStack)-1]
			if elem.tag.name == string(p.tagname) {
				elemStack = elemStack[:len(elemStack)-1]
				p.advance()
			} else {
				p.errorf("mismatch end tag, expected </%s>, got </%s>", elem.tag.name, p.tagname)
			}
		case html.TextToken:
			// TODO(paulsmith): de-dupe this logic
			if idx := strings.IndexRune(p.raw, transSym); idx >= 0 {
				if idx < len(p.raw)-1 && p.raw[idx+1] == transSym {
					// it's an escaped transition sym
					// TODO(paulsmith): emit transSym literal text expression
				} else {
					newOffset := p.start + idx + 1
					p.parser.offset = newOffset
					leading := p.raw[:idx]
					if idx > 0 {
						var htmlNode nodeLiteral
						htmlNode.pos.start = p.start
						htmlNode.pos.end = p.start + len(leading)
						htmlNode.str = leading
						result = append(result, &htmlNode)
					}
					e := p.transition()
					result = append(result, e)
				}
			} else {
				result = append(result, p.emitLiteral())
			}
			p.advance()
		case html.CommentToken:
			p.advance()
		case html.DoctypeToken:
			p.errorf("doctype token may not be a child of an element")
		default:
			panic(fmt.Sprintf("unexpected HTML token type %v", p.toktyp))
		}
	}

	return result
}

type Optional[T any] struct {
	value *T
}

func None[T any]() Optional[T] {
	return Optional[T]{}
}

func Some[T any](val T) Optional[T] {
	return Optional[T]{value: &val}
}

func Value[T any](o Optional[T]) (T, bool) {
	if o.value != nil {
		return *o.value, true
	} else {
		var zero T
		return zero, false
	}
}

type codeParser struct {
	parser         *parser
	baseOffset     int
	file           *token.File
	scanner        *scanner.Scanner
	bufferedToken  Optional[goToken]
	acceptedToken  Optional[goToken]
	lookaheadToken Optional[goToken]
}

func (p *codeParser) reset() {
	p.baseOffset = p.parser.offset
	fset := token.NewFileSet()
	source := p.parser.remainingSource()
	p.file = fset.AddFile("", fset.Base(), len(source))
	p.scanner = new(scanner.Scanner)
	p.scanner.Init(p.file, []byte(source), nil, scanner.ScanComments)
	p.bufferedToken = None[goToken]()
	p.acceptedToken = None[goToken]()
	p.lookaheadToken = None[goToken]()
}

func (p *codeParser) errorf(format string, args ...any) {
	p.parser.errorf(format, args...)
}

func (p *codeParser) sourceFrom(pos token.Pos) string {
	return p.parser.sourceFrom(p.baseOffset + p.file.Offset(pos))
}

func (p *codeParser) sourceRange(start, end int) string {
	return p.parser.src[start:end]
}

func (p *codeParser) lookahead() goToken {
	if tok, ok := Value(p.bufferedToken); ok {
		p.bufferedToken = None[goToken]()
		return tok
	}
	var t goToken
	var lit string
	t.pos, t.tok, lit = p.scanner.Scan()
	// from go/scanner docs:
	// If the returned token is a literal (token.IDENT, token.INT, token.FLOAT,
	// token.IMAG, token.CHAR, token.STRING) or token.COMMENT, the literal string
	// has the corresponding value.
	//
	// If the returned token is a keyword, the literal string is the keyword.
	//
	// If the returned token is token.SEMICOLON, the corresponding
	// literal string is ";" if the semicolon was present in the source,
	// and "\n" if the semicolon was inserted because of a newline or
	// at EOF.
	//
	// If the returned token is token.ILLEGAL, the literal string is the
	// offending character.
	//
	// In all other cases, Scan returns an empty literal string.
	if t.tok.IsLiteral() || t.tok.IsKeyword() || t.tok == token.SEMICOLON || t.tok == token.COMMENT || t.tok == token.ILLEGAL {
		t.lit = lit
	} else {
		t.lit = t.tok.String()
	}
	return t
}

type goToken struct {
	pos token.Pos
	tok token.Token
	lit string
}

func (t goToken) String() string {
	return t.lit
}

func (p *codeParser) peek() goToken {
	if tok, ok := Value(p.lookaheadToken); ok {
		return tok
	}
	tok := p.lookahead()
	p.lookaheadToken = Some(tok)
	return tok
}

// charAt() returns the byte at the offset in the input source string. because
// the Go tokenizer discards white space, we need this method in order to
// check for, for example, a space after an identifier in parsing an implicit
// expression, because that would denote the end of that simple expression in
// Pushup syntax.
func (p *codeParser) charAt(offset int) byte {
	if len(p.parser.src) > offset {
		return p.parser.src[offset]
	}
	return 0
}

func (p *codeParser) prev() goToken {
	if tok, ok := Value(p.acceptedToken); ok {
		return tok
	}
	panic("internal error: expected some accepted token, got none")
}

// sync synchronizes the global offset position in the main Pushup parser with
// the Go code scanner.
func (p *codeParser) sync() goToken {
	t := p.peek()
	// the Go scanner skips over whitespace so we need to be careful about the
	// logic for advancing the main parser internal source offset.
	p.parser.offset = p.tokenOffset(t) + len(t.String())
	return t
}

// advance consumes the lookahead token (which should be accessed via p.peek())
func (p *codeParser) advance() {
	p.acceptedToken = Some(p.sync())
	p.lookaheadToken = Some(p.lookahead())
}

// backup undoes a call to p.advance(). may only be called once between calls
// to p.advance(). must have called p.advance() at least once prior.
func (p *codeParser) backup() {
	if _, ok := Value(p.bufferedToken); ok {
		panic("internal error: p.backup() called more than once before p.advance()")
	}
	if _, ok := Value(p.lookaheadToken); !ok {
		panic("internal error: p.backup() called before p.advance()")
	}
	if tok, ok := Value(p.acceptedToken); ok {
		p.parser.offset = p.tokenOffset(tok)
	} else {
		panic("internal error: expected some accepted token, got none")
	}
	p.bufferedToken = p.lookaheadToken
	p.lookaheadToken = p.acceptedToken
}

func (p *codeParser) transition() *nodeBlock {
	htmlParser := p.parser.htmlParser
	htmlParser.advance()
	var stmtBlock nodeBlock
	ws := htmlParser.skipWhitespace()
	for _, n := range ws {
		stmtBlock.nodes = append(stmtBlock.nodes, n)
	}
	elem := htmlParser.parseElement()
	stmtBlock.nodes = append(stmtBlock.nodes, elem)
	p.reset()
	return &stmtBlock
}

func (p *codeParser) parseCode() node {
	// starting at the token just past the transSym indicating a transition from HTML
	// parsing to Go code parsing
	var e node
	tok := p.peek().tok
	lit := p.peek().lit
	if tok == token.IF {
		p.advance()
		e = p.parseIfStmt()
	} else if tok == token.IDENT && lit == "handler" {
		p.advance()
		e = p.parseHandlerKeyword()
		// NOTE(paulsmith): there is a tricky bit here where an implicit
		// expression in the form of an identifier token is next and we would
		// not be able to distinguish it from a keyword. this is also a problem
		// for name collisions because a user could create a variable named the
		// same as a keyword and then later try to use it in an implicit
		// expression, but it would be parsed with the keyword parsing flow
		// (which probably would lead to an infinite loop because it wouldn't
		// terminate and the user would be left with an unresponsive Pushup
		// compiler). a fix could be to have a notion of allowed contexts in
		// which a keyword block or an implicit expression could be used in the
		// surrounding markup, and only parse for either depending on which
		// context is current.
	} else if tok == token.IDENT && lit == "section" {
		p.advance()
		e = p.parseSectionKeyword()
	} else if tok == token.IDENT && lit == "partial" {
		p.advance()
		e = p.parsePartialKeyword()
	} else if tok == token.LBRACE {
		e = p.parseCodeBlock()
	} else if tok == token.IMPORT {
		p.advance()
		e = p.parseImportKeyword()
	} else if tok == token.FOR {
		p.advance()
		e = p.parseForStmt()
	} else if tok == token.LPAREN {
		p.advance()
		e = p.parseExplicitExpression()
	} else if tok == token.IDENT {
		e = p.parseImplicitExpression()
	} else if tok == token.INT || tok == token.FLOAT || tok == token.STRING {
		p.errorf("Go integer, float, and string literals must be grouped by parens")
	} else if tok == token.EOF {
		p.errorf("unexpected EOF in code parser")
	} else if tok == token.NOT || tok == token.REM || tok == token.AND || tok == token.CHAR {
		p.errorf("invalid '%s' Go token while parsing code", tok.String())
	} else {
		p.errorf("expected Pushup keyword or expression, got %q", tok.String())
	}
	return e
}

func (p *codeParser) parseIfStmt() *nodeIf {
	var stmt nodeIf
	start := p.peek().pos
	maxread := start
	lastlit := p.peek().String()
loop:
	for {
		switch p.peek().tok {
		case token.EOF:
			p.errorf("premature end of conditional in IF statement")
		case token.LBRACE:
			// conditional expression has been scanned
			break loop
			// TODO(paulsmith): add cases for tokens that are illegal in an expression
		}
		maxread = p.peek().pos
		lastlit = p.peek().String()
		p.advance()
	}
	n := (p.file.Offset(maxread) - p.file.Offset(start)) + len(lastlit)
	offset := p.baseOffset + p.file.Offset(start)
	stmt.cond = new(nodeGoStrExpr)
	stmt.cond.pos.start = offset
	stmt.cond.pos.end = offset + n
	stmt.cond.expr = p.sourceFrom(start)[:n]
	if _, err := goparser.ParseExpr(stmt.cond.expr); err != nil {
		p.errorf("parsing Go expression in IF conditional: %w", err)
	}
	stmt.then = p.parseStmtBlock()
	// parse ^else clause
	if p.peek().tok == token.XOR {
		p.advance()
		if p.peek().tok == token.ELSE {
			p.advance()
			if p.peek().tok == token.XOR {
				p.advance()
				if p.peek().tok == token.IF {
					p.advance()
					stmt.alt = p.parseIfStmt()
				} else {
					p.errorf("expected `if' after transition character, got %v", p.peek().String())
				}
			} else {
				stmt.alt = p.parseStmtBlock()
			}
		}
	}
	return &stmt
}

func (p *codeParser) parseForStmt() *nodeFor {
	var stmt nodeFor
	start := p.peek().pos
loop:
	for {
		switch p.peek().tok {
		case token.EOF:
			p.errorf("premature end of clause in FOR statement")
		case token.LBRACE:
			break loop
		default:
			p.advance()
		}
	}
	n := (p.file.Offset(p.prev().pos) - p.file.Offset(start)) + len(p.prev().String())
	offset := p.baseOffset + p.file.Offset(start)
	stmt.clause = new(nodeGoCode)
	stmt.clause.pos.start = offset
	stmt.clause.pos.end = offset + n
	stmt.clause.code = p.sourceFrom(start)[:n]
	stmt.block = p.parseStmtBlock()
	return &stmt
}

func (p *codeParser) parseStmtBlock() *nodeBlock {
	// we are sitting on the opening '{' token here
	if p.peek().tok != token.LBRACE {
		p.errorf("expected '{', got '%s'", p.peek().String())
	}
	p.advance()
	var block *nodeBlock
	switch p.peek().tok {
	// check for a transition, i.e., stay in code parser
	case token.XOR:
		p.advance()
		code := p.parseCode()
		if p.peek().tok == token.SEMICOLON {
			p.advance()
		}
		block = &nodeBlock{nodes: []node{code}}
	case token.EOF:
		p.errorf("premature end of block in IF statement")
	default:
		block = p.transition()
	}
	// we should be at the closing '}' token here
	if p.peek().tok != token.RBRACE {
		if p.peek().tok == token.LSS {
			p.errorf("there must be a single HTML element inside a Go code block, try wrapping them in a <text></text> pseudo-element")
		} else {
			p.errorf("expected closing '}', got %v", p.peek())
		}
	}
	p.advance()
	return block
}

// TODO(paulsmith): extract a common function with parseCodeKeyword
func (p *codeParser) parseHandlerKeyword() *nodeGoCode {
	result := &nodeGoCode{context: handlerGoCode}
	// we are one token past the 'handler' keyword
	if p.peek().tok != token.LBRACE {
		p.errorf("expected '{', got '%s'", p.peek().tok)
	}
	depth := 1
	p.advance()
	result.pos.start = p.parser.offset
	start := p.peek().pos
loop:
	for {
		switch p.peek().tok {
		case token.LBRACE:
			depth++
		case token.RBRACE:
			depth--
			if depth == 0 {
				break loop
			}
		}
		p.advance()
	}
	n := (p.file.Offset(p.prev().pos) - p.file.Offset(start)) + len(p.prev().String())
	if p.peek().tok != token.RBRACE {
		panic("")
	}
	p.advance()
	result.code = p.sourceFrom(start)[:n]
	result.pos.end = result.pos.start + n
	return result
}

func (p *codeParser) parseSectionKeyword() *nodeSection {
	// enter function one past the "section" IDENT token
	// FIXME(paulsmith): we are currently requiring that the name of the
	// partial be a valid Go identifier, but there is no reason that need be
	// the case. perhaps a string is better here.
	if p.peek().tok != token.IDENT {
		p.errorf("expected IDENT, got %s", p.peek().tok.String())
	}
	result := &nodeSection{name: p.peek().lit}
	result.pos.start = p.parser.offset
	p.advance()
	result.pos.end = p.parser.offset
	result.block = p.parseStmtBlock()
	return result
}

func (p *codeParser) parsePartialKeyword() *nodePartial {
	// enter function one past the "partial" IDENT token
	// FIXME(paulsmith): we are currently requiring that the name of the
	// partial be a valid Go identifier, but there is no reason that need be
	// the case. authors may want to, for example, have a name that is contains
	// dashes or other punctuation (which would need to be URL-escaped for the
	// routing of partials). perhaps a string is better here.
	if p.peek().tok != token.IDENT {
		p.errorf("expected IDENT, got %s", p.peek().tok.String())
	}
	result := &nodePartial{name: p.peek().lit}
	result.pos.start = p.parser.offset
	p.advance()
	result.pos.end = p.parser.offset
	result.block = p.parseStmtBlock()
	return result
}

func (p *codeParser) parseCodeBlock() *nodeGoCode {
	result := &nodeGoCode{context: inlineGoCode}
	if p.peek().tok != token.LBRACE {
		p.errorf("expected '{', got '%s'", p.peek().tok)
	}
	depth := 1
	p.advance()
	result.pos.start = p.parser.offset
	start := p.peek().pos
	maxread := start
	lastlit := p.peek().String()
loop:
	for {
		switch p.peek().tok {
		case token.LBRACE:
			depth++
		case token.RBRACE:
			depth--
			if depth == 0 {
				break loop
			}
		case token.EOF:
			p.errorf("unexpected EOF parsing code block")
		}
		maxread = p.peek().pos
		lastlit = p.peek().String()
		p.advance()
	}
	n := (p.file.Offset(maxread) - p.file.Offset(start)) + len(lastlit)
	if p.peek().tok != token.RBRACE {
		panic("")
	}
	p.advance()
	result.code = p.sourceFrom(start)[:n]
	result.pos.end = result.pos.start + n
	return result
}

func (p *codeParser) parseImportKeyword() *nodeImport {
	/*
		examples:
		TRANS_SYMimport   "lib/math"         math.Sin
		TRANS_SYMimport m "lib/math"         m.Sin
		TRANS_SYMimport . "lib/math"         Sin
	*/
	e := new(nodeImport)
	// we are one token past the 'import' keyword
	switch p.peek().tok {
	case token.STRING:
		e.decl.path = p.peek().lit
		p.advance()
	case token.IDENT:
		e.decl.pkgName = p.peek().lit
		p.advance()
		if p.peek().tok != token.STRING {
			p.errorf("expected string, got %s", p.peek().tok)
		}
		e.decl.path = p.peek().lit
	case token.PERIOD:
		e.decl.pkgName = "."
		p.advance()
		if p.peek().tok != token.STRING {
			p.errorf("expected string, got %s", p.peek().tok)
		}
		e.decl.path = p.peek().lit
	default:
		p.errorf("unexpected token type after "+transSymStr+"import: %s", p.peek().tok)
	}
	return e
}

func (p *codeParser) parseExplicitExpression() *nodeGoStrExpr {
	// one token past the opening '('
	result := new(nodeGoStrExpr)
	result.pos.start = p.parser.offset
	start := p.peek().pos
	maxread := start
	lastlit := p.peek().String()
	depth := 1
loop:
	for {
		switch p.peek().tok {
		case token.LPAREN:
			depth++
		case token.RPAREN:
			depth--
			if depth == 0 {
				break loop
			}
		case token.ILLEGAL:
			p.errorf("illegal Go token %q", p.peek().String())
		case token.EOF:
			p.errorf("unterminated explicit expression, expected closing ')'")
		}
		maxread = p.peek().pos
		lastlit = p.peek().String()
		p.advance()
	}
	n := (p.file.Offset(maxread) - p.file.Offset(start)) + len(lastlit)
	if p.peek().tok != token.RPAREN {
		panic(fmt.Sprintf("internal error: expected ')', got '%s'", p.peek().String()))
	}
	_ = p.sync()
	result.expr = p.sourceFrom(start)[:n]
	result.pos.end = result.pos.start + n
	if _, err := goparser.ParseExpr(result.expr); err != nil {
		p.errorf("illegal Go expression: %w", err)
	}
	return result
}

// offset is the current global offset into the original source code of the Pushup file.
func (p *codeParser) offset() int {
	return p.parser.offset
}

// tokenOffset is the global offset into the original source code for this token.
func (p *codeParser) tokenOffset(tok goToken) int {
	return p.baseOffset + p.file.Offset(tok.pos)
}

func (p *codeParser) parseImplicitExpression() *nodeGoStrExpr {
	if p.peek().tok != token.IDENT {
		panic("internal error: expected Go identifier start implicit expression")
	}
	result := new(nodeGoStrExpr)
	end := p.tokenOffset(p.peek())
	result.pos.start = end
	identLen := len(p.peek().String())
	end += identLen
	p.advance()
	if !unicode.IsSpace(rune(p.charAt(result.pos.start + identLen))) {
	Loop:
		for {
			if p.peek().tok == token.LPAREN {
				nested := 1
				end++
				p.advance()
				for {
					if p.peek().tok == token.RPAREN {
						end++
						p.advance()
						nested--
						if nested == 0 {
							goto Loop
						}
					} else if p.peek().tok == token.ILLEGAL {
						p.errorf("illegal Go token %q", p.peek().String())
					} else if p.peek().tok == token.EOF {
						p.errorf("unexpected EOF, want ')'")
					}
					end = p.tokenOffset(p.peek()) + len(p.peek().String())
					p.advance()
				}
			} else if p.peek().tok == token.LBRACK { // '['
				nested := 1
				end++
				p.advance()
				for {
					if p.peek().tok == token.RBRACK {
						end++
						p.advance()
						nested--
						if nested == 0 {
							goto Loop
						}
					} else if p.peek().tok == token.ILLEGAL {
						p.errorf("illegal Go token %q", p.peek().String())
					} else if p.peek().tok == token.EOF {
						p.errorf("unexpected EOF, want ')'")
					}
					end = p.tokenOffset(p.peek()) + len(p.peek().String())
					p.advance()
				}
			} else if p.peek().tok == token.PERIOD {
				last := p.peek().pos
				p.advance()
				end++
				// if space between period and next token, regardless of what
				// it is, need to break. the period needs to be pushed back on
				// to the stream to be parsed.
				if p.peek().pos-last > 1 || p.peek().tok != token.IDENT {
					p.backup()
					end--
					break
				}
				adv := len(p.peek().String())
				end += adv
				if unicode.IsSpace(rune(p.charAt(end))) {
					// done
					p.advance()
					break
				}
				p.advance()
			} else {
				break
			}
		}
	}
	result.expr = p.sourceRange(result.pos.start, end)
	result.pos.end = end
	if _, err := goparser.ParseExpr(result.expr); err != nil {
		p.errorf("illegal Go expression %q: %w", result.expr, err)
	}
	return result
}

const padding = " "

func prettyPrintTree(t *syntaxTree) {
	depth := -1
	var w io.Writer = os.Stdout
	pad := func() { w.Write([]byte(strings.Repeat(padding, depth))) }
	var f inspector
	f = func(n node) bool {
		depth++
		defer func() {
			depth--
		}()
		pad()
		switch n := n.(type) {
		case *nodeLiteral:
			if !isAllWhitespace(n.str) {
				str := n.str
				if len(str) > 20 {
					str = str[:20] + "..."
				}
				fmt.Fprintf(w, "\x1b[32m%q\x1b[0m\n", str)
			}
		case *nodeGoStrExpr:
			fmt.Fprintf(w, "\x1b[33m%s\x1b[0m\n", n.expr)
		case *nodeGoCode:
			fmt.Fprintf(w, "\x1b[34m%s\x1b[0m\n", n.code)
		case *nodeIf:
			fmt.Fprintf(w, "\x1b[35mIF\x1b[0m")
			f(n.cond)
			pad()
			fmt.Fprintf(w, "\x1b[35mTHEN\x1b[0m\n")
			f(n.then)
			if n.alt != nil {
				pad()
				fmt.Fprintf(w, "\x1b[1;35mELSE\x1b[0m\n")
				f(n.alt)
			}
			return false
		case *nodeFor:
			fmt.Fprintf(w, "\x1b[36mFOR\x1b[0m")
			f(n.clause)
			f(n.block)
			return false
		case *nodeElement:
			fmt.Fprintf(w, "\x1b[31m%s\x1b[0m\n", n.tag.start())
			f(nodeList(n.children))
			fmt.Fprintf(w, "\x1b[31m%s\x1b[0m\n", n.tag.end())
			return false
		case *nodeSection:
			fmt.Fprintf(w, "SECTION %s\n", n.name)
			f(n.block)
			return false
		case *nodePartial:
			fmt.Fprintf(w, "PARTIAL %s\n", n.name)
			f(n.block)
			return false
		case *nodeBlock:
			f(nodeList(n.nodes))
			return false
		case *nodeImport:
			fmt.Fprintf(w, "IMPORT ")
			if n.decl.pkgName != "" {
				fmt.Fprintf(w, "%s", n.decl.pkgName)
			}
			fmt.Fprintf(w, "%s\n", n.decl.path)
		case *nodeLayout:
			fmt.Fprintf(w, "LAYOUT %s\n", n.name)
		case nodeList:
			for _, x := range n {
				f(x)
			}
			return false
		}
		return true
	}
	inspect(nodeList(t.nodes), f)
}

// implement the HTML5 spec lexing algorithm for open tags. this is necessary
// because in order to switch safely between HTML and Go code parsing in
// the Pushup parser, we need to precisely track the read character position
// internally to start (or self-closing) tags, because the transition character
// may appear inside HTML attributes. the golang.org/x/net/html tokenizer
// that forms the basis of the Pushup HTML parser, while it precisely tracks
// character position for token types indirectly via its Raw() method, does
// not help us inside a start (or self-closing) tag, including attributes. So,
// yes, we're doing extra work, re-tokenizing the tag. But it's not expensive
// work (just open and self-closing tags, not the whole doc) and there's not an
// alternative with golang.org/x/net/html.
//
// we start in the data state
//
// https://html.spec.whatwg.org/multipage/parsing.html#tag-open-state

func scanAttrs(openTag string) (attrs []*attr, err error) {
	// maintain some invariants, we are not a general-purpose HTML
	// tokenizer/parser, we are just parsing open tags.
	if len(openTag) == 0 {
		return []*attr{}, nil
	}
	if ch := openTag[0]; ch != '<' {
		return nil, openTagScanError(fmt.Sprintf("expected '<', got '%c'", ch))
	}

	l := newOpenTagLexer(openTag)
	// panic mode error handling
	defer func() {
		if e := recover(); e != nil {
			if le, ok := e.(openTagScanError); ok {
				attrs = nil
				err = le
			} else {
				panic(e)
			}
		}
	}()
	attrs = l.scan()
	return
}

type openTagScanError string

func (e openTagScanError) Error() string {
	return string(e)
}

type openTagLexer struct {
	raw         string
	pos         int
	state       openTagLexState
	returnState openTagLexState
	charRefBuf  bytes.Buffer

	nextInputChar        int
	provideCurrInputChar bool

	attrs    []*attrBuilder
	currAttr *attrBuilder
}

func newOpenTagLexer(source string) *openTagLexer {
	l := new(openTagLexer)
	l.raw = source
	l.state = openTagLexData
	return l
}

func (l *openTagLexer) consumeNextInputChar() int {
	var result int
	if l.provideCurrInputChar {
		l.provideCurrInputChar = false
		result = l.nextInputChar
	} else {
		if len(l.raw) > 0 {
			l.decodeNextInputChar()
		} else {
			l.nextInputChar = eof
		}
		result = l.nextInputChar
	}
	return result
}

func (l *openTagLexer) decodeNextInputChar() {
	l.nextInputChar = int(l.raw[0])
	l.raw = l.raw[1:]
	l.pos++
}

type attrBuilder struct {
	name  bufPos
	value bufPos
}

type bufPos struct {
	*bytes.Buffer
	start pos
}

type attr struct {
	name  stringPos
	value stringPos
}

type stringPos struct {
	string
	start pos
}

type pos int

type openTagLexState int

// NOTE(paulsmith): we only consider a subset of the HTML5 tokenization states,
// because we rely on the golang.org/x/net/html tokenizer to produce a valid
// start tag token that we scan here for attributes. so certain states are not
// considered, or are considered assertion errors if they would ordinarily be
// entered into.
const (
	openTagLexData openTagLexState = iota
	openTagLexTagOpen
	openTagLexTagName
	openTagLexBeforeAttrName
	openTagLexAttrName
	openTagLexAfterAttrName
	openTagLexBeforeAttrVal
	openTagLexAttrValDoubleQuote
	openTagLexAttrValSingleQuote
	openTagLexAttrValUnquoted
	openTagLexAfterAttrValQuoted
	openTagLexCharRef
	openTagLexNamedCharRef
	openTagLexNumericCharRef
	openTagLexSelfClosingStartTag
	openTagLexAmbiguousAmpersand
)

func (s openTagLexState) String() string {
	switch s {
	case openTagLexData:
		return "Data"
	case openTagLexTagOpen:
		return "TagOpen"
	case openTagLexTagName:
		return "TagName"
	case openTagLexBeforeAttrName:
		return "BeforeAttrName"
	case openTagLexAttrName:
		return "AttrName"
	case openTagLexAfterAttrName:
		return "AfterAttrName"
	case openTagLexBeforeAttrVal:
		return "BeforeAttrVal"
	case openTagLexAttrValDoubleQuote:
		return "AttrValDoubleQuote"
	case openTagLexAttrValSingleQuote:
		return "AttrValSingleQuote"
	case openTagLexAttrValUnquoted:
		return "AttrValUnquoted"
	case openTagLexAfterAttrValQuoted:
		return "AfterAttrValQuoted"
	case openTagLexCharRef:
		return "CharRef"
	case openTagLexNamedCharRef:
		return "NamedCharRef"
	case openTagLexNumericCharRef:
		return "NumericCharRef"
	case openTagLexSelfClosingStartTag:
		return "SelfClosingStartTag"
	case openTagLexAmbiguousAmpersand:
		return "AmbiguousAmpersand"
	default:
		panic("unexpected tag state")
	}
}

const eof = -1

func (l *openTagLexer) scan() []*attr {
loop:
	for {
		switch l.state {
		// 13.2.5.1 Data state
		// https://html.spec.whatwg.org/multipage/parsing.html#data-state
		case openTagLexData:
			ch := l.consumeNextInputChar()
			switch ch {
			case '&':
				l.returnState = openTagLexData
				l.switchState(openTagLexCharRef)
			case '<':
				l.switchState(openTagLexTagOpen)
			case 0:
				l.specParseError("unexpected-null-character")
			default:
				l.errorf("found '%c' in data state, expected '<'", ch)
			}
		// 13.2.5.6 Tag open state
		// https://html.spec.whatwg.org/multipage/parsing.html#tag-open-state
		case openTagLexTagOpen:
			ch := l.consumeNextInputChar()
			switch {
			case ch == '!':
				l.errorf("input '%c' switch to markup declaration open state", ch)
			case ch == '/':
				l.errorf("input '%c' switch to end tag open state", ch)
			case isASCIIAlpha(ch):
				l.reconsumeIn(openTagLexTagName)
			case ch == '?':
				l.errorf("input '%c' parse error", ch)
			case ch == eof:
				l.errorf("eof before tag name parse error")
			default:
				l.errorf("found '%c' in tag open state", ch)
			}
		// 13.2.5.8 Tag name state
		// https://html.spec.whatwg.org/multipage/parsing.html#tag-name-state
		case openTagLexTagName:
			ch := l.consumeNextInputChar()
			switch {
			case ch == '\t' || ch == '\n' || ch == '\f' || ch == ' ':
				l.switchState(openTagLexBeforeAttrName)
			case ch == '/':
				l.switchState(openTagLexSelfClosingStartTag)
			case ch == '>':
				break loop
			case isASCIIUpper(ch):
				// append lowercase version of current input char to current tag token's tag name
				// not needed, we know the tag name from the golang.org/x/net/html tokenizer
			case ch == 0:
				l.errorf("found null in tag name state")
			case ch == eof:
				l.errorf("found eof in tag name state")
			default:
				// append current input char to current tag token's tag name
			}
		// 13.2.5.32 Before attribute name state
		// https://html.spec.whatwg.org/multipage/parsing.html#before-attribute-name-state
		case openTagLexBeforeAttrName:
			ch := l.consumeNextInputChar()
			switch ch {
			case '\t', '\n', '\f', ' ':
				// ignore
			case '/', '>', eof:
				l.reconsumeIn(openTagLexAfterAttrName)
			case '=':
				l.errorf("found '%c' in before attribute name state", ch)
			default:
				l.newAttr()
				l.reconsumeIn(openTagLexAttrName)
			}
		// 13.2.5.33 Attribute name state
		// https://html.spec.whatwg.org/multipage/parsing.html#attribute-name-state
		case openTagLexAttrName:
			ch := l.consumeNextInputChar()
			switch {
			case ch == '\t' || ch == '\n' || ch == '\f' || ch == ' ' || ch == '/' || ch == '>' || ch == eof:
				l.reconsumeIn(openTagLexAfterAttrName)
				l.cmpAttrName()
			case ch == '=':
				l.switchState(openTagLexBeforeAttrVal)
				l.cmpAttrName()
			case isASCIIUpper(ch):
				// append lowercase version of current input character to current attr's name
				l.appendCurrName(int(unicode.ToLower(rune(ch))))
			case ch == 0:
				l.specParseError("unexpected-null-character")
			case ch == '"' || ch == '\'' || ch == '<':
				l.specParseError("unexpected-character-in-attribute-name")
				// append current input character to current attribute's name
				l.appendCurrName(ch)
			default:
				// append current input character to current attribute's name
				l.appendCurrName(ch)
			}
		// 13.2.5.34 After attribute name state
		// https://html.spec.whatwg.org/multipage/parsing.html#after-attribute-name-state
		case openTagLexAfterAttrName:
			ch := l.consumeNextInputChar()
			switch ch {
			case '\t', '\n', '\f', ' ':
				// ignore
			case '/':
				l.switchState(openTagLexSelfClosingStartTag)
			case '=':
				l.switchState(openTagLexBeforeAttrVal)
			case '>':
				break loop
			case eof:
				l.specParseError("eof-in-tag")
			default:
				l.newAttr()
				l.reconsumeIn(openTagLexAttrName)
			}
		// 13.2.5.35 Before attribute value state
		// https://html.spec.whatwg.org/multipage/parsing.html#before-attribute-value-state
		case openTagLexBeforeAttrVal:
			ch := l.consumeNextInputChar()
			switch ch {
			case '\t', '\n', '\f', ' ':
				// ignore
			case '"':
				l.switchState(openTagLexAttrValDoubleQuote)
			case '\'':
				l.switchState(openTagLexAttrValSingleQuote)
			case '>':
				l.specParseError("missing-attribute-value")
				break loop
			default:
				l.reconsumeIn(openTagLexAttrValUnquoted)
			}
		// 13.2.5.36 Attribute value (double-quoted) state
		// https://html.spec.whatwg.org/multipage/parsing.html#attribute-value-(double-quoted)-state
		case openTagLexAttrValDoubleQuote:
			ch := l.consumeNextInputChar()
			switch ch {
			case '"':
				l.switchState(openTagLexAfterAttrValQuoted)
			case '&':
				l.returnState = openTagLexAttrValDoubleQuote
				l.switchState(openTagLexCharRef)
			case 0:
				l.errorf("found null in attribute value (double-quoted) state")
			case eof:
				l.errorf("found EOF in tag")
			default:
				l.appendCurrVal(ch)
			}
		// 13.2.5.37 Attribute value (single-quoted) state
		// https://html.spec.whatwg.org/multipage/parsing.html#attribute-value-(single-quoted)-state
		case openTagLexAttrValSingleQuote:
			ch := l.consumeNextInputChar()
			switch ch {
			case '"':
				l.switchState(openTagLexAfterAttrValQuoted)
			case '&':
				l.returnState = openTagLexAttrValSingleQuote
				l.switchState(openTagLexCharRef)
			case 0:
				l.errorf("found null in attribute value (single-quoted) state")
			case eof:
				l.errorf("found EOF in tag")
			default:
				l.appendCurrVal(ch)
			}
		// 13.2.5.38 Attribute value (unquoted) state
		// https://html.spec.whatwg.org/multipage/parsing.html#attribute-value-(unquoted)-state
		case openTagLexAttrValUnquoted:
			ch := l.consumeNextInputChar()
			switch ch {
			case '\t', '\n', '\f', ' ':
				l.switchState(openTagLexBeforeAttrName)
			case '&':
				l.returnState = openTagLexAttrValUnquoted
				l.switchState(openTagLexCharRef)
			case '>':
				break loop
			case 0:
				l.errorf("found null in attribute value (unquoted) state")
			case '"', '\'', '<', '=', '`':
				l.specParseError("unexpected-null-character")
				l.appendCurrVal(ch)
			case eof:
				l.errorf("found EOF in tag")
			default:
				l.appendCurrVal(ch)
			}
		// 13.2.5.39 After attribute value (quoted) state
		// https://html.spec.whatwg.org/multipage/parsing.html#after-attribute-value-(quoted)-state
		case openTagLexAfterAttrValQuoted:
			ch := l.consumeNextInputChar()
			switch ch {
			case '\t', '\n', '\f', ' ':
				l.switchState(openTagLexBeforeAttrName)
			case '/':
				l.switchState(openTagLexSelfClosingStartTag)
			case '>':
				break loop
			case eof:
				l.errorf("found EOF in tag")
			default:
				l.specParseError("missing-whitespace-between-attributes")
				l.reconsumeIn(openTagLexBeforeAttrName)
			}
		// 13.2.5.72 Character reference state
		// https://html.spec.whatwg.org/multipage/parsing.html#character-reference-state
		case openTagLexCharRef:
			l.charRefBuf = bytes.Buffer{}
			l.charRefBuf.WriteByte('&')
			ch := l.consumeNextInputChar()
			switch {
			case isASCIIAlphanum(ch):
				l.reconsumeIn(openTagLexNamedCharRef)
			case ch == '#':
				l.charRefBuf.WriteByte(byte(ch))
				l.switchState(openTagLexNumericCharRef)
			default:
				l.flushCharRef()
				l.reconsumeIn(l.returnState)
			}
		// 13.2.5.40 Self-closing start tag state
		// https://html.spec.whatwg.org/multipage/parsing.html#self-closing-start-tag-state
		case openTagLexSelfClosingStartTag:
			ch := l.consumeNextInputChar()
			switch ch {
			case '>':
				break loop
			case eof:
				l.errorf("found EOF in tag")
			default:
				l.specParseError("unexpected-solidus-in-tag")
				l.reconsumeIn(openTagLexBeforeAttrName)
			}
		// 13.2.5.73 Named character reference state
		// https://html.spec.whatwg.org/multipage/parsing.html#named-character-reference-state
		case openTagLexNamedCharRef:
			var ch int
			for ch = l.consumeNextInputChar(); isASCIIAlphanum(ch); ch = l.consumeNextInputChar() {
				l.charRefBuf.WriteByte(byte(ch))
			}
			ident := l.charRefBuf.String()
			ref, ok := namedCharRefs[ident]
			if ok {
				lastCharMatched := ident[len(ident)-1]
				if l.consumedAsPartOfAttr() && lastCharMatched != ';' && (ch == '=' || isASCIIAlphanum(ch)) {
					l.flushCharRef()
					l.switchState(l.returnState)
				} else {
					if ch != ';' {
						l.specParseError("missing-semicolon-after-character-reference")
					}
					l.charRefBuf.Reset()
					l.charRefBuf.WriteString(ref)
					l.flushCharRef()
					l.switchState(l.returnState)
				}
			} else {
				l.flushCharRef()
				l.switchState(openTagLexAmbiguousAmpersand)
			}

		default:
			l.errorf("unimplemented parse state " + l.state.String())
		}
	}

	result := make([]*attr, len(l.attrs))
	for i := range l.attrs {
		builder := l.attrs[i]
		attr := &attr{
			name: stringPos{
				builder.name.String(),
				builder.name.start,
			},
			value: stringPos{
				builder.value.String(),
				builder.value.start,
			},
		}
		result[i] = attr
	}
	return result
}

func (l *openTagLexer) consumedAsPartOfAttr() bool {
	if l.returnState == openTagLexAttrValDoubleQuote ||
		l.returnState == openTagLexAttrValSingleQuote ||
		l.returnState == openTagLexAttrValUnquoted {
		return true
	} else {
		return false
	}
}

func (l *openTagLexer) flushCharRef() {
	if l.currAttr.value.start == 0 {
		l.currAttr.value.start = pos(l.pos - 1)
	}
	l.currAttr.value.Write(l.charRefBuf.Bytes())
}

func (l *openTagLexer) newAttr() {
	a := &attrBuilder{
		name: bufPos{
			Buffer: new(bytes.Buffer),
		},
		value: bufPos{
			Buffer: new(bytes.Buffer),
		},
	}
	l.attrs = append(l.attrs, a)
	l.currAttr = a
}

func (l *openTagLexer) appendCurrName(ch int) {
	if l.currAttr.name.start == 0 {
		l.currAttr.name.start = pos(l.pos - 1)
	}
	l.currAttr.name.WriteByte(byte(ch))
}

func (l *openTagLexer) appendCurrVal(ch int) {
	if l.currAttr.value.start == 0 {
		l.currAttr.value.start = pos(l.pos - 1)
	}
	l.currAttr.value.WriteByte(byte(ch))
}

func (l *openTagLexer) errorf(format string, args ...any) {
	panic(openTagScanError(fmt.Sprintf(format, args...)))
}

// specParseError panics with a openTagScanError type as the panic value but
// is specifically meant for signaling the known parse errors per the HTML5
// parsing specification we expect to encounter with this limited version
// that just focuses on tokenizing open tags.
func (l *openTagLexer) specParseError(code string) {
	switch code {
	case "eof-in-tag":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-eof-in-tag
		// This error occurs if the parser encounters the end of the input
		// stream in a start tag or an end tag (e.g., <div id=). Such a tag is
		// ignored.
	case "unexpected-character-in-attribute-name":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-unexpected-character-in-attribute-name
	case "duplicate-attribute":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-duplicate-attribute
		// This error occurs if the parser encounters an attribute in a tag that
		// already has an attribute with the same name. The parser ignores all such
		// duplicate occurrences of the attribute.
	case "missing-attribute-value":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-missing-attribute-value
		// This error occurs if the parser encounters a U+003E (>) code point where
		// an attribute value is expected (e.g., <div id=>). The parser treats the
		// attribute as having an empty value.
	case "missing-whitespace-between-attributes":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-missing-whitespace-between-attributes
	case "unexpected-solidus-in-tag":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-unexpected-solidus-in-tag
		// This error occurs if the parser encounters a U+002F (/) code point
		// that is not a part of a quoted attribute value and not immediately
		// followed by a U+003E (>) code point in a tag (e.g., <div / id="foo">).
		// In this case the parser behaves as if it encountered ASCII whitespace.
	case "unexpected-null-character":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-unexpected-null-character
		// This error occurs if the parser encounters a U+0000 NULL code point
		// in the input stream in certain positions. In general, such code
		// points are either ignored or, for security reasons, replaced with a
		// U+FFFD REPLACEMENT CHARACTER.
	case "missing-semicolon-after-character-reference":
		// https://html.spec.whatwg.org/multipage/parsing.html#parse-error-missing-semicolon-after-character-reference
		// This error occurs if the parser encounters a character reference
		// that is not terminated by a U+003B (;) code point. Usually the
		// parser behaves as if character reference is terminated by the U+003B
		// (;) code point; however, there are some ambiguous cases in which the
		// parser includes subsequent code points in the character reference.
	default:
		panic("unexpected parse error code " + code)
	}
	panic(openTagScanError(strings.ReplaceAll(code, "-", " ")))
}

func (l *openTagLexer) reconsumeIn(state openTagLexState) {
	l.provideCurrInputChar = true
	l.switchState(state)
}

func (l *openTagLexer) exitingState(state openTagLexState) {
}

func (l *openTagLexer) enteringState(state openTagLexState) {
}

func (l *openTagLexer) switchState(state openTagLexState) {
	l.exitingState(l.state)
	l.enteringState(state)
	l.state = state
}

func (l *openTagLexer) cmpAttrName() {
	for i := range l.attrs[:len(l.attrs)-1] {
		if l.currAttr.name == l.attrs[i].name {
			l.specParseError("duplicate-attribute")
			// we're supposed to ignore this per the spec but the
			// golang.org/x/net/html tokenizer doesn't, so we follow that
			// TODO(paulsmith): open issue with ^^
		}
	}
}

func isASCIIUpper(ch int) bool {
	if ch >= 'A' && ch <= 'Z' {
		return true
	}
	return false
}

func isASCIIAlpha(ch int) bool {
	if isASCIIUpper(ch) || (ch >= 'a' && ch <= 'z') {
		return true
	}
	return false
}

func isASCIIAlphanum(ch int) bool {
	if isASCIIAlpha(ch) || (ch >= '0' && ch <= '9') {
		return true
	}
	return false
}
