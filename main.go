// Pushup web framework
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"

	"golang.org/x/mod/modfile"
	"golang.org/x/net/html/atom"
	"golang.org/x/sync/errgroup"
)

const (
	upFileExt       = ".up"
	compiledFileExt = upFileExt + ".go"
)

func main() {
	var version bool
	var cpuprofile = flag.String("cpuprofile", "", "")
	var memprofile = flag.String("memprofile", "", "")

	flag.Usage = printPushupHelp
	flag.BoolVar(&version, "version", false, "Print the version number and exit")

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

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

	var found bool
	for _, c := range cliCmds {
		if c.name == cmdName {
			found = true
			cmd := c.fn(args)
			if err := cmd.do(); err != nil {
				log.Fatalf("%s command: %v", c.name, err)
			}
		}
	}

	if !found {
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmdName)
		flag.Usage()
		os.Exit(1)
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
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
	modulePath string
}

func newNewCmd(arguments []string) *newCmd {
	flags := flag.NewFlagSet("pushup new", flag.ExitOnError)
	modulePathFlag := newRegexString(`^\w[\w-]*$`, "")
	flags.Var(modulePathFlag, "module", "Go module path of the new Pushup app")
	//nolint:errcheck
	flags.Parse(arguments)
	if flags.NArg() > 1 {
		log.Fatalf("extra unprocessed argument(s)")
	}
	projectDir := "."
	modPath := modulePathFlag.String()
	if flags.NArg() == 1 {
		projectDir = flags.Arg(0)
		if modPath == "" {
			modPath = filepath.Clean(flags.Arg(0))
		}
	}
	return &newCmd{projectDir: projectDir, modulePath: modPath}
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
	for _, name := range []string{"pages", "static"} {
		path := filepath.Join(n.projectDir, name)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("creating project directory %s: %w", path, err)
		}
	}

	scaffoldFiles := []string{
		"pages/index.up",
		"pages/index.go",
		"static/style.css",
		"static/htmx.min.js",
	}
	for _, name := range scaffoldFiles {
		dest := filepath.Join(n.projectDir, name)
		src := filepath.Join("scaffold", name)
		if err := copyFileFS(scaffold, dest, src); err != nil {
			return fmt.Errorf("copying scaffold file to project dir %w", err)
		}
	}

	if err := createGoModFile(n.projectDir, n.modulePath); err != nil {
		return err
	}

	if err := initVcs(n.projectDir, vcsGit); err != nil {
		return err
	}

	return nil
}

func createGoModFile(destDir string, moduleName string) error {
	// create the go.mod file
	{
		cmd := exec.Command("go", "mod", "init", moduleName)
		cmd.Dir = destDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("creating new go.mod file: %w", err)
		}
	}
	// add Pushup as an API dependency
	{
		cmd := exec.Command("go", "get", pushupModulePath+"@latest")
		cmd.Dir = destDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go get'ing Pushup: %w", err)
		}
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
			log.Printf("[WARN] git not found in $PATH")
			return nil
		}

		cmd := exec.Command(path, "init")
		cmd.Dir = projectDir
		if err = cmd.Run(); err != nil {
			return fmt.Errorf("git init: %w", err)
		}
	default:
		panic("internal error: unimplemented VCS")
	}

	return nil
}

type buildCmd struct {
	projectDir         string
	applyOptimizations bool
	parseOnly          bool
	codeGenOnly        bool
	compileOnly        bool
	output             string
	embedSource        bool
	pages              stringSlice
	verbose            bool

	files *projectFiles
}

func setBuildFlags(flags *flag.FlagSet, b *buildCmd) {
	flags.BoolVar(&b.applyOptimizations, "O", false, "apply simple optimizations to the parse tree")
	flags.BoolVar(&b.parseOnly, "parse-only", false, "exit after dumping parse result")
	flags.BoolVar(&b.codeGenOnly, "codegen-only", false, "codegen only, don't compile")
	flags.BoolVar(&b.compileOnly, "compile-only", false, "compile only, don't start web server after")
	flags.StringVar(&b.output, "o", "", "path to output executable")
	flags.BoolVar(&b.embedSource, "embed-source", true, "embed the source .up files in executable")
	flags.Var(&b.pages, "page", "path to a Pushup page. mulitple can be given")
	flags.BoolVar(&b.verbose, "verbose", false, "output verbose information")
}

func newBuildCmd(arguments []string) *buildCmd {
	flags := flag.NewFlagSet("pushup build", flag.ExitOnError)
	b := new(buildCmd)
	setBuildFlags(flags, b)
	//nolint:errcheck
	flags.Parse(arguments)
	if flags.NArg() == 1 {
		b.projectDir = flags.Arg(0)
	} else {
		b.projectDir = "."
	}
	if b.output == "" {
		modPath, err := projectModulePath(b.projectDir)
		if err != nil {
			panic(fmt.Sprintf("getting module path: %v", err))
		}
		b.output = b.projectDir + string([]byte{os.PathSeparator}) + filepath.Base(modPath)
	}
	log.Printf("OUTPUT: %v", b.output)
	return b
}

func (b *buildCmd) rescanProjectFiles() error {
	if len(b.pages) == 0 {
		var err error
		b.files, err = findProjectFiles(b.projectDir)
		if err != nil {
			return err
		}
	} else {
		pfiles := &projectFiles{}
		for _, page := range b.pages {
			pfiles.pages = append(pfiles.pages, projectFile{path: page})
		}
		b.files = pfiles
	}
	return nil
}

func (b *buildCmd) do() error {
	// FIXME(paulsmith): remove singleFile (and -single flag) and replace with
	// configurable project root, leading path strip, and optional file paths.
	if err := b.rescanProjectFiles(); err != nil {
		return err
	}

	modPath, err := projectModulePath(b.projectDir)
	if err != nil {
		return fmt.Errorf("getting module path: %w", err)
	}

	// FIXME(paulsmith): dedupe this with runCmd.do()
	cparams := &compileProjectParams{
		modPath:            modPath,
		parseOnly:          b.parseOnly,
		files:              b.files,
		applyOptimizations: b.applyOptimizations,
		embedSource:        b.embedSource,
	}

	output, err := compileProject(cparams)
	if err != nil {
		return fmt.Errorf("parsing and compiling: %w", err)
	}

	if b.parseOnly || b.codeGenOnly {
		return nil
	}

	lparams := &linkerParams{
		compiledOutput: output,
		modPath:        modPath,
		projectDir:     b.projectDir,
		exePath:        b.output,
	}
	if err := linkProject(context.TODO(), lparams); err != nil {
		return fmt.Errorf("linking project: %w", err)
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

	//nolint:errcheck
	flags.Parse(arguments)
	// FIXME this logic is duplicated with newBuildCmd
	if flags.NArg() == 1 {
		b.projectDir = flags.Arg(0)
	} else {
		b.projectDir = "."
	}
	if b.output == "" {
		modPath, err := projectModulePath(b.projectDir)
		if err != nil {
			panic(fmt.Sprintf("getting module path: %v", err))
		}
		b.output = b.projectDir + string([]byte{os.PathSeparator}) + filepath.Base(modPath)
	}
	return &runCmd{
		buildCmd:   b,
		host:       *host,
		port:       *port,
		unixSocket: *unixSocket,
		devReload:  *devReload,
	}
}

var errFileChanged = fmt.Errorf("file change detected")
var errSignalCaught = fmt.Errorf("signal caught")

func (r *runCmd) do() error {
	if err := r.buildCmd.do(); err != nil {
		return fmt.Errorf("build command: %w", err)
	}

	if r.compileOnly {
		return nil
	}

	// TODO(paulsmith): add a linkOnly flag (or a releaseMode flag, alternatively?)

	if r.devReload {
		var mu sync.Mutex
		linkComplete := sync.NewCond(&mu)

		tmpdir, err := os.MkdirTemp("", "pushupdev")
		if err != nil {
			return fmt.Errorf("creating temp dir: %v", err)
		}
		defer os.RemoveAll(tmpdir)
		socketPath := filepath.Join(tmpdir, "pushup-"+strconv.Itoa(os.Getpid())+".sock")
		if err = startReloadRevProxy(socketPath, linkComplete, r.port); err != nil {
			return fmt.Errorf("starting reverse proxy: %v", err)
		}

		ln, err := net.Listen("unix", socketPath)
		if err != nil {
			return fmt.Errorf("listening on Unix socket: %v", err)
		}

		for {
			ctx, cancel := context.WithCancelCause(context.Background())
			signals := make(chan os.Signal, 1)
			signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
			reload := make(chan struct{})

			go func() {
				select {
				case <-reload:
					cancel(errFileChanged)
				case <-signals:
					cancel(errSignalCaught)
				case <-ctx.Done():
				}
			}()

			if err := r.rescanProjectFiles(); err != nil {
				return fmt.Errorf("scanning for project files: %v", err)
			}

			modPath, err := projectModulePath(r.projectDir)
			if err != nil {
				return fmt.Errorf("getting module path: %w", err)
			}

			cparams := &compileProjectParams{
				modPath:            modPath,
				parseOnly:          r.parseOnly,
				files:              r.files,
				applyOptimizations: r.applyOptimizations,
				embedSource:        r.embedSource,
			}
			compiledOutput, err := compileProject(cparams)
			if err != nil {
				return fmt.Errorf("parsing and compiling: %v", err)
			}

			lparams := &linkerParams{
				compiledOutput: compiledOutput,
				modPath:        modPath,
				projectDir:     r.projectDir,
				exePath:        r.output,
			}
			if err := linkProject(context.TODO(), lparams); err != nil {
				return fmt.Errorf("building Pushup project: %v", err)
			}
			linkComplete.Broadcast()

			watchForReload(ctx, r.projectDir, reload)
			if err := runProject(ctx, lparams.exePath, ln); err != nil {
				return fmt.Errorf("building and running generated Go code: %v", err)
			}

			signal.Stop(signals)

			if err := context.Cause(ctx); errors.Is(err, errSignalCaught) {
				return nil
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
		if err := runProject(context.TODO(), r.buildCmd.output, ln); err != nil {
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
	//nolint:errcheck
	flags.Parse(args)
	if flags.NArg() == 1 {
		r.projectDir = flags.Arg(0)
	} else {
		r.projectDir = "."
	}
	return r
}

func (r *routesCmd) do() error {
	files, err := findProjectFiles(r.projectDir)
	if err != nil {
		return err
	}
	// TODO(paulsmith): sort by route match specificity
	// TODO(paulsmith): colorize the dynamic path segments
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 1, ' ', 0)
	for _, page := range files.pages {
		route := page.route()
		fmt.Fprintln(w, route+"\t"+page.relpath())
	}
	w.Flush()
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
	fmt.Fprintln(w, "Usage: pushup [flags] [command] [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "\t-version\t\tPrint the version number and exit")
	fmt.Fprintln(w, "\t-cpuprofile\t\tWrite CPU profile to `file`")
	fmt.Fprintln(w, "\t-memprofile\t\tWrite memory profile to `file`")
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

// project file represents an .up file in a Pushup project context.
type projectFile struct {
	// path from cwd to the .up file
	path string
}

func (f *projectFile) relpath() string {
	return f.path
}

//nolint:unused
type router interface {
	route() string
}

func (f *projectFile) route() string {
	return routeForPage(f.relpath())
}

// projectFiles represents all the source files in a Pushup project.
type projectFiles struct {
	// list of .up page files
	pages []projectFile
	// paths to static files like JS, CSS, etc.
	static []projectFile
}

//nolint:unused
func (f *projectFiles) debug() {
	fmt.Println("pages:")
	for _, p := range f.pages {
		fmt.Printf("\t%v\n", p)
	}
	fmt.Println("static:")
	for _, p := range f.static {
		fmt.Printf("\t%v\n", p)
	}
}

func findProjectFiles(root string) (*projectFiles, error) {
	pf := new(projectFiles)

	pagesDir := filepath.Join(root, "pages")
	{
		if err := fs.WalkDir(os.DirFS(pagesDir), ".", func(path string, d fs.DirEntry, _ error) error {
			if !d.IsDir() && filepath.Ext(path) == upFileExt {
				pfile := projectFile{path: filepath.Join(pagesDir, path)}
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

	staticDir := filepath.Join(root, "static")
	if dirExists(staticDir) {
		if err := fs.WalkDir(os.DirFS(staticDir), ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				path = filepath.Join(staticDir, path)
				pf.static = append(pf.static, projectFile{path: path})
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
	// path to directory with the compiled Pushup project code
	compiledOutputDir string
	buildDir          string
	output            string
	verbose           bool
}

func projectModulePath(projectDir string) (string, error) {
	goModContents, err := os.ReadFile(filepath.Join(projectDir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("could not read go.mod: %w", err)
	}
	f, err := modfile.Parse("go.mod", goModContents, nil)
	if err != nil {
		return "", fmt.Errorf("parsing go.mod file: %w", err)
	}
	return f.Module.Mod.Path, nil
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

	cmd := exec.CommandContext(ctx, exePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{file}
	cmd.Env = append(os.Environ(), "PUSHUP_LISTENER_FD=3")

	ctx, cancel := context.WithCancel(ctx)

	g := new(errgroup.Group)

	g.Go(func() error {
		<-ctx.Done()
		if errors.Is(context.Cause(ctx), errFileChanged) {
			log.Printf("[PUSHUP RELOADER] file changed, reloading")
		} else if errors.Is(context.Cause(ctx), errSignalCaught) {
			log.Printf("[PUSHUP] got signal, shutting down")
		}
		return nil
	})

	g.Go(func() error {
		// NOTE(paulsmith): intentionally ignoring *ExitError because the child
		// process will be signal killed here as a matter of course
		//nolint:errcheck
		if err := cmd.Run(); err != nil {
			log.Printf("[WARN] error running command: %v", err)
		}
		cancel()
		return nil
	})

	// NOTE(paulsmith): intentionally ignoring *ExitError for same reason as
	// above
	//nolint:errcheck
	g.Wait()

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

func init() {
	if atom.Lookup([]byte("text")) != 0 {
		panic("expected <text> to not be a common HTML tag")
	}
}
