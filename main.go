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
	"io/ioutil"
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

func main() {
	var version bool

	flag.Usage = printPushupHelp
	flag.BoolVar(&version, "version", false, "Print the version number and exit")

	flag.Parse()

	if version {
		printVersion()
		os.Exit(0)
	}

	if flag.NArg() == 0 {
		printPushupHelp()
		os.Exit(1)
	}

	command := flag.Arg(0)

	for _, cmd := range commands {
		if cmd.name == command {
			switch command {
			case "new":
				flags := flag.NewFlagSet("pushup "+command, flag.ExitOnError)
				moduleNameFlag := newRegexString(`^\w[\w-]*$`, "example/myproject")
				flags.Var(moduleNameFlag, "module", "name of Go module of the new Pushup app")
				flags.Parse(flag.Args()[1:])
				if flags.NArg() > 1 {
					log.Fatalf("extra unprocessed argument(s)")
				}
				projectDir := "."
				if flags.NArg() == 1 {
					projectDir = flags.Arg(0)
				}
				newc := newCmdFromFlags(projectDir, moduleNameFlag.String())
				if err := newc.do(); err != nil {
					log.Fatalf("'new' command error: %v", err)
				}
			case "build":
				panic("unimplemented 'build' command")
			case "run":
				flags := flag.NewFlagSet("pushup "+command, flag.ExitOnError)
				// build flags
				projectName := newRegexString(`^\w+`, "myproject")
				buildPkg := flags.String("build-pkg", "example/myproject/build", "name of package of compiled Pushup app")
				flags.Var(projectName, "project", "name of Pushup project")
				singleFlag := flags.String("single", "", "path to a single Pushup file")
				applyOptimizations := flags.Bool("O", false, "apply simple optimizations to the parse tree")
				parseOnly := flags.Bool("parse-only", false, "exit after dumping parse result")
				compileOnly := flags.Bool("compile-only", false, "compile only, don't start web server after")
				outDir := flags.String("out-dir", "./build", "path to output build directory")
				// run flags
				host := flags.String("host", "0.0.0.0", "host to listen on")
				port := flags.String("port", "8080", "port to listen on with TCP IPv4")
				unixSocket := flags.String("unix-socket", "", "path to listen on with Unix socket")
				devReload := flags.Bool("dev", false, "compile and run the Pushup app and reload on changes")
				flags.Parse(flag.Args()[1:])
				run := runCmdFromFlags(host, port, unixSocket, devReload)
				run.buildCmd = buildCmdFromFlags(projectName.String(), *buildPkg, singleFlag, applyOptimizations, parseOnly, compileOnly, outDir)
				if err := run.do(); err != nil {
					log.Fatalf("'run' command error: %v", err)
				}
			}
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

type newCmd struct {
	projectDir string
	moduleName string
}

func newCmdFromFlags(projectDir string, moduleName string) *newCmd {
	return &newCmd{projectDir: projectDir, moduleName: moduleName}
}

//go:embed scaffold/layouts/*.pushup scaffold/pages/*.pushup
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
	for _, name := range []string{"pages", "layouts", "pkg"} {
		path := filepath.Join(n.projectDir, "app", name)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("creating project directory %s: %w", path, err)
		}
	}

	for _, name := range []string{"layouts/default.pushup", "pages/index.pushup"} {
		dest := filepath.Join(n.projectDir, "app", name)
		src := filepath.Join("scaffold", name)
		if err := copyFileFS(scaffold, dest, src); err != nil {
			return fmt.Errorf("copying scaffold file to project dir %w", err)
		}
	}

	if err := createGoModFile(n.projectDir, n.moduleName); err != nil {
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

type buildCmd struct {
	projectName        string
	buildPkg           string
	singleFile         string
	applyOptimizations bool
	parseOnly          bool
	compileOnly        bool
	outDir             string
}

func buildCmdFromFlags(projectName string, buildPkg string, singleFile *string, applyOptimizations *bool, parseOnly *bool, compileOnly *bool, outDir *string) *buildCmd {
	return &buildCmd{
		projectName:        projectName,
		buildPkg:           buildPkg,
		singleFile:         *singleFile,
		applyOptimizations: *applyOptimizations,
		parseOnly:          *parseOnly,
		compileOnly:        *compileOnly,
		outDir:             *outDir,
	}
}

func (b *buildCmd) do() error {
	return nil
}

type runCmd struct {
	*buildCmd
	port       string
	host       string
	unixSocket string
	devReload  bool
}

func runCmdFromFlags(host, port, unixSocket *string, devReload *bool) *runCmd {
	return &runCmd{
		host:       *host,
		port:       *port,
		unixSocket: *unixSocket,
		devReload:  *devReload,
	}
}

func (r *runCmd) do() error {
	appDir := "app"

	if err := parseAndCompile(appDir, r.outDir, r.parseOnly, r.singleFile, r.applyOptimizations); err != nil {
		return fmt.Errorf("parsing and compiling: %w", err)
	}

	// TODO(paulsmith): add a linkOnly flag and separate build step from
	// buildAndRun. (or a releaseMode flag, alternatively?)
	if !r.compileOnly {
		var mu sync.Mutex
		buildComplete := sync.NewCond(&mu)

		ctx := newPushupContext(context.Background())

		if r.devReload {
			reload := make(chan struct{})
			tmpdir, err := ioutil.TempDir("", "pushupdev")
			if err != nil {
				return fmt.Errorf("creating temp dir: %v", err)
			}
			defer os.RemoveAll(tmpdir)
			socketPath := filepath.Join(tmpdir, "pushup-"+strconv.Itoa(os.Getpid())+".sock")
			if err := startReloadRevProxy(socketPath, buildComplete, r.port); err != nil {
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
				if err := parseAndCompile(appDir, r.outDir, r.parseOnly, r.singleFile, r.applyOptimizations); err != nil {
					return fmt.Errorf("parsing and compiling: %v", err)
				}
				ctx = newPushupContext(context.Background())
				go func() {
					watchForReload(ctx, ctx.fileChangeCancel, appDir, reload)
				}()
				if err := buildAndRun(ctx, r.projectName, r.buildPkg, r.outDir, ln, buildComplete); err != nil {
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

			if err := buildAndRun(ctx, r.projectName, r.buildPkg, r.outDir, ln, buildComplete); err != nil {
				return fmt.Errorf("building and running generated Go code: %v", err)
			}
		}
	}

	return nil
}

type command struct {
	name        string
	usage       string
	description string
}

// TODO(paulsmith): link these with their command struct counterparts
var commands = []command{
	{name: "new", usage: "[path]", description: "create new Pushup project directory"},
	{name: "build", usage: "", description: "compile Pushup project and build executable"},
	{name: "run", usage: "", description: "build and run Pushup project app"},
}

func printPushupHelp() {
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, "Usage: pushup [command] [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "\t-version\t\tPrint the version number and exit")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	for _, cmd := range commands {
		fmt.Fprintf(w, "\t%s %s\t\t%s\n", cmd.name, cmd.usage, cmd.description)
	}
	w.Flush()
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

func parseAndCompile(root string, outDir string, parseOnly bool, singleFile string, applyOptimizations bool) error {
	var layoutsDir string
	var pagesDir string
	var pkgDir string

	var layoutFiles []string
	var pushupFiles []string
	var pkgFiles []string

	os.RemoveAll(outDir)

	if singleFile != "" {
		pushupFiles = []string{singleFile}
	} else {
		layoutsDir = filepath.Join(root, "layouts")
		{
			if !dirExists(layoutsDir) {
				return fmt.Errorf("invalid Pushup project directory structure: couldn't find `layouts` subdir")
			}

			entries, err := os.ReadDir(layoutsDir)
			if err != nil {
				return fmt.Errorf("reading app layouts directory: %w", err)
			}

			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pushup") {
					path := filepath.Join(layoutsDir, entry.Name())
					layoutFiles = append(layoutFiles, path)
				}
			}
		}

		pagesDir = filepath.Join(root, "pages")
		{
			if !dirExists(pagesDir) {
				return fmt.Errorf("invalid Pushup project directory structure: couldn't find `pages` subdir")
			}

			pushupFiles = getPushupPagePaths(pagesDir)
		}

		pkgDir = filepath.Join(root, "pkg")
		{
			if !dirExists(pkgDir) {
				return fmt.Errorf("invalid Pushup project directory structure: couldn't find `pkg` subdir")
			}

			entries, err := os.ReadDir(pkgDir)
			if err != nil {
				return fmt.Errorf("reading app pkg directory: %w", err)
			}

			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
					path := filepath.Join(pkgDir, entry.Name())
					pkgFiles = append(pkgFiles, path)
				}
			}
		}
	}

	if parseOnly {
		for _, path := range pushupFiles {
			b, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading file %s: %w", path, err)
			}
			tree, err := parse(string(b))
			if err != nil {
				return fmt.Errorf("parsing file %s: %w", path, err)
			}
			(&debugPrettyPrinter{w: os.Stdout}).visitNodes(tree.nodes)
		}
		os.Exit(0)
	}

	for _, path := range layoutFiles {
		if err := compilePushup(path, layoutsDir, compileLayout, outDir, applyOptimizations, singleFile); err != nil {
			return fmt.Errorf("compiling layout file %s: %w", path, err)
		}
	}

	for _, path := range pushupFiles {
		if err := compilePushup(path, pagesDir, compilePushupPage, outDir, applyOptimizations, singleFile); err != nil {
			return fmt.Errorf("compiling pushup file %s: %w", path, err)
		}
	}

	for _, path := range pkgFiles {
		if err := copyFile(filepath.Join(outDir, filepath.Base(path)), path); err != nil {
			return fmt.Errorf("copying Go package file %s: %w", path, err)
		}
	}

	if err := copyFileFS(runtimeFiles, filepath.Join(outDir, "pushup_support.go"), filepath.Join("_runtime", "pushup_support.go")); err != nil {
		return fmt.Errorf("copying runtime file: %w", err)
	}

	return nil
}

func watchForReload(ctx context.Context, cancel context.CancelFunc, root string, reload chan struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(fmt.Errorf("creating new fsnotify watcher: %v", err))
	}

	go debounceEvents(ctx, 125*time.Millisecond, watcher, func(event fsnotify.Event) {
		// log.Printf("name: %s\top: %s", event.Name, event.Op)
		if event.Op == fsnotify.Create && isDir(event.Name) {
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

func isDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		panic(err)
	}
	return fi.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

func watchDirRecursively(watcher *fsnotify.Watcher, root string) error {
	err := fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			path := filepath.Join(root, path)
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

	if mediatype == "text/html" && res.Header.Get("HX-Response") != "true" {
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
	complete *sync.Cond
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
			log.Printf("client disconnected")
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
			// log.Printf("GOT EVENT: %s %d", ev.String(), ev.Op)
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

func (s *cancellationSource) Value(key any) any {
	panic("not implemented") // TODO: Implement
}

func getPushupPagePaths(root string) []string {
	var paths []string
	err := fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && filepath.Ext(path) == ".pushup" {
			paths = append(paths, filepath.Join(root, path))
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return paths
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

// assumes directory for dest already exists
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

func buildAndRun(ctx context.Context, projectName string, buildPkg string, srcDir string, ln net.Listener, buildComplete *sync.Cond) error {
	mainExeDir := filepath.Join(srcDir, "cmd", projectName)
	if err := os.MkdirAll(mainExeDir, 0755); err != nil {
		return fmt.Errorf("making directory for command: %w", err)
	}

	{
		t := template.Must(template.ParseFS(runtimeFiles, filepath.Join("_runtime", "cmd", "main.go")))
		f, err := os.Create(filepath.Join(mainExeDir, "main.go"))
		if err != nil {
			return fmt.Errorf("creating main.go: %w", err)
		}
		if err := t.Execute(f, map[string]string{"BuildPkg": buildPkg}); err != nil {
			return fmt.Errorf("executing main.go template: %w", err)
		}
		f.Close()
	}

	var file *os.File
	var err error
	switch ln := ln.(type) {
	case *net.TCPListener:
		file, err = ln.File()
	case *net.UnixListener:
		file, err = ln.File()
	default:
		panic("")
	}
	if err != nil {
		return fmt.Errorf("getting file from Unix socket listener: %w", err)
	}

	args := []string{"run", "./build/cmd/" + projectName}
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	sysProcAttr(cmd)
	cmd.ExtraFiles = []*os.File{file}
	cmd.Env = append(os.Environ(), "PUSHUP_LISTENER_FD=3")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting project main executable: %w", err)
	}

	// FIXME(paulsmith): actually detect when the build is complete
	buildComplete.Broadcast()

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
		// regardless of how they were exited. this is also way there is a
		// `done' channel in this function, to signal to the other goroutine
		// waiting for context cancellation.
		err := cmd.Wait()
		if err != nil {
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

type compilationStrategy int

const (
	compilePushupPage compilationStrategy = iota
	compileLayout
)

func compilePushup(sourcePath string, rootDir string, strategy compilationStrategy, targetDir string,
	applyOptimizations bool, singleFile string,
) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", targetDir, err)
	}

	b, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	source := string(b)

	tree, err := parse(source)
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic while parsing %s: %v", sourcePath, err)
			panic(err)
		}
	}()

	// apply some simple optimizations
	if applyOptimizations {
		tree = optimize(tree)
	}

	var c codeGenUnit
	switch strategy {
	case compilePushupPage:
		page, err := postProcessTree(tree)
		if err != nil {
			return fmt.Errorf("post-processing tree: %w", err)
		}
		layoutName := page.layout
		if singleFile != "" {
			layoutName = ""
		}
		route := routeFromPath(sourcePath, rootDir)
		c = &pageCodeGen{path: trimCommonPrefix(sourcePath, rootDir), source: source, layout: layoutName, page: page, route: route}
	case compileLayout:
		c = &layoutCodeGen{path: trimCommonPrefix(sourcePath, rootDir), source: source, tree: tree}
	default:
		panic("")
	}

	outputFilename := generatedFilename(sourcePath, rootDir, strategy)
	outputPath := filepath.Join(targetDir, outputFilename)
	basename := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))

	if err := generateCodeToFile(c, basename, outputPath, strategy); err != nil {
		return fmt.Errorf("generating Go code from parse result: %w", err)
	}

	return nil
}

// generatedFilename returns the filename for the .go file containing the
// generated code for the Pushup page.
func generatedFilename(path string, root string, strategy compilationStrategy) string {
	path = trimCommonPrefix(path, root)
	var dirs []string
	dir := filepath.Dir(path)
	if dir != "." {
		dirs = strings.Split(dir, string([]rune{os.PathSeparator}))
	}
	file := filepath.Base(path)
	base := strings.TrimSuffix(file, filepath.Ext(file))
	prefix := strings.Join(dirs, "__")
	var result string
	var suffix string
	switch strategy {
	case compileLayout:
		suffix = "__layout-gen"
	case compilePushupPage:
		suffix = "__gen"
	default:
		panic("")
	}
	if prefix != "" {
		result = prefix + "__" + base + suffix + ".go"
	} else {
		result = base + suffix + ".go"
	}
	return result
}

// routeFromPath produces the URL path route from the name of the Pushup page.
// path is the path to the Pushup file. root is the path of the root of the
// Pushup project.
func routeFromPath(path string, root string) string {
	path = trimCommonPrefix(path, root)
	var dirs []string
	dir := filepath.Dir(path)
	if dir != "." {
		dirs = strings.Split(dir, string([]rune{os.PathSeparator}))
	}
	file := filepath.Base(path)
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
	return route
}

func trimCommonPrefix(path string, prefix string) string {
	path = filepath.Clean(path)
	prefix = filepath.Clean(prefix)
	stripped := strings.TrimPrefix(path, prefix)
	stripped = strings.TrimPrefix(stripped, "/")
	return stripped
}

// node represents a portion of the Pushup syntax, like a chunk of HTML,
// or a Go expression to be evaluated, or a control flow construct like `if'
// or `for'.
type node interface {
	Pos() span
	nodeAcceptor
}

type nodeAcceptor interface {
	accept(nodeVisitor)
}

type nodeList []node

func (n nodeList) Pos() span            { return n[0].Pos() }
func (n nodeList) accept(v nodeVisitor) { v.visitNodes(n) }

type nodeVisitor interface {
	visitElement(*nodeElement)
	visitLiteral(*nodeLiteral)
	visitGoStrExpr(*nodeGoStrExpr)
	visitGoCode(*nodeGoCode)
	visitIf(*nodeIf)
	visitFor(*nodeFor)
	visitStmtBlock(*nodeBlock)
	visitNodes([]node)
	visitImport(*nodeImport)
	visitLayout(*nodeLayout)
}

type nodeLiteral struct {
	str string
	pos span
}

func (e nodeLiteral) Pos() span             { return e.pos }
func (e *nodeLiteral) accept(v nodeVisitor) { v.visitLiteral(e) }

var _ node = (*nodeLiteral)(nil)

type nodeGoStrExpr struct {
	expr string
	pos  span
}

func (e nodeGoStrExpr) Pos() span             { return e.pos }
func (e *nodeGoStrExpr) accept(v nodeVisitor) { v.visitGoStrExpr(e) }

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

func (e nodeGoCode) Pos() span             { return e.pos }
func (e *nodeGoCode) accept(v nodeVisitor) { v.visitGoCode(e) }

var _ node = (*nodeGoCode)(nil)

type nodeIf struct {
	cond *nodeGoStrExpr
	then *nodeBlock
	alt  *nodeBlock
}

func (e nodeIf) Pos() span             { return e.cond.pos }
func (e *nodeIf) accept(v nodeVisitor) { v.visitIf(e) }

var _ node = (*nodeIf)(nil)

type nodeFor struct {
	clause *nodeGoCode
	block  *nodeBlock
}

func (e nodeFor) Pos() span             { return e.clause.pos }
func (e *nodeFor) accept(v nodeVisitor) { v.visitFor(e) }

// A nodeBlock represents a block of nodes, i.e., a sequence of nodes that
// appear in order in the source syntax.
type nodeBlock struct {
	nodes []node
}

func (e *nodeBlock) Pos() span {
	// FIXME(paulsmith): span end all exprs
	return e.nodes[0].Pos()
}
func (e *nodeBlock) accept(v nodeVisitor) { v.visitStmtBlock(e) }

var _ node = (*nodeBlock)(nil)

// nodeElement represents an HTML element, with a start tag, optional
// attributes, optional children, and an end tag.
type nodeElement struct {
	tag           tag
	startTagNodes []node
	children      []node
	pos           span
}

func (e nodeElement) Pos() span             { return e.pos }
func (e *nodeElement) accept(v nodeVisitor) { v.visitElement(e) }

var _ node = (*nodeElement)(nil)

type nodeImport struct {
	decl importDecl
	pos  span
}

func (e nodeImport) Pos() span             { return e.pos }
func (e *nodeImport) accept(v nodeVisitor) { v.visitImport(e) }

var _ node = (*nodeImport)(nil)

type nodeLayout struct {
	name string
	pos  span
}

func (e nodeLayout) Pos() span             { return e.pos }
func (e *nodeLayout) accept(v nodeVisitor) { v.visitLayout(e) }

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
	// log.Printf("SAVED %d NODES", before-len(nodes))
	return nodes
}

type page struct {
	layout  string
	imports []importDecl
	handler *nodeGoCode
	nodes   []node
}

func postProcessTree(tree *syntaxTree) (*page, error) {
	// FIXME(paulsmith): recurse down into child nodes
	layoutSet := false
	page := &page{layout: "default"}
	n := 0
	for _, e := range tree.nodes {
		switch e := e.(type) {
		case *nodeImport:
			page.imports = append(page.imports, e.decl)
		case *nodeLayout:
			if layoutSet {
				return nil, fmt.Errorf("layout already set as %q", page.layout)
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
					return nil, fmt.Errorf("only one handler per page can be defined")
				}
				page.handler = e
			} else {
				tree.nodes[n] = e
				n++
			}
		default:
			tree.nodes[n] = e
			n++
		}
	}
	page.nodes = tree.nodes[:n]
	return page, nil
}

func generateCodeToFile(c codeGenUnit, basename string, outputPath string, strategy compilationStrategy) error {
	code, err := genCode(c, basename, strategy)
	if err != nil {
		return fmt.Errorf("code gen: %w", err)
	}

	if err := os.WriteFile(outputPath, code, 0644); err != nil {
		return fmt.Errorf("writing out generated code to file: %w", err)
	}

	return nil
}

// TODO(paulsmith): probably can unify the two implementations of this and just
// use the strategy type for discrimintating
type codeGenUnit interface {
	filePath() string
	nodes() []node
	lineNo(span) int
}

type pageCodeGen struct {
	path   string
	source string
	layout string
	page   *page
	route  string
}

func (c *pageCodeGen) filePath() string {
	return c.path
}

func (c *pageCodeGen) nodes() []node {
	return c.page.nodes
}

func (c *pageCodeGen) lineNo(s span) int {
	return lineCount(c.source[:s.start+1])
}

func lineCount(s string) int {
	return strings.Count(s, "\n") + 1
}

type layoutCodeGen struct {
	path   string
	source string
	tree   *syntaxTree
}

func (c *layoutCodeGen) filePath() string {
	return c.path
}

func (c *layoutCodeGen) nodes() []node {
	return c.tree.nodes
}

func (c *layoutCodeGen) lineNo(s span) int {
	return lineCount(c.source[:s.start+1])
}

type codeGenerator struct {
	c        codeGenUnit
	strategy compilationStrategy
	basename string
	imports  map[importDecl]bool
	outb     bytes.Buffer
	bodyb    bytes.Buffer
}

func newCodeGenerator(c codeGenUnit, basename string, strategy compilationStrategy) *codeGenerator {
	var g codeGenerator
	g.c = c
	g.strategy = strategy
	g.basename = basename
	g.imports = make(map[importDecl]bool)
	if p, ok := c.(*pageCodeGen); ok {
		for _, decl := range p.page.imports {
			g.imports[decl] = true
		}
	}
	return &g
}

func (g *codeGenerator) used(path string) {
	g.imports[importDecl{path: strconv.Quote(path), pkgName: ""}] = true
}

func (g *codeGenerator) nodeLineNo(e node) {
	g.lineNo(g.c.lineNo(e.Pos()))
}

func (g *codeGenerator) lineNo(n int) {
	g.bodyPrintf("//line %s:%d\n", g.basename+".pushup", n)
}

func (g *codeGenerator) outPrintf(format string, args ...any) {
	fmt.Fprintf(&g.outb, format, args...)
}

func (g *codeGenerator) bodyPrintf(format string, args ...any) {
	fmt.Fprintf(&g.bodyb, format, args...)
}

func (g *codeGenerator) generate() {
	g.visitNodes(g.c.nodes())
}

func (g *codeGenerator) visitLiteral(n *nodeLiteral) {
	g.used("io")
	g.nodeLineNo(n)
	g.bodyPrintf("io.WriteString(w, %s)\n", strconv.Quote(n.str))
}

func (g *codeGenerator) visitElement(n *nodeElement) {
	g.used("io")
	g.nodeLineNo(n)
	for _, e := range n.startTagNodes {
		e.accept(g)
	}
	nodeList(n.children).accept(g)
	g.bodyPrintf("io.WriteString(w, %s)\n", strconv.Quote(n.tag.end()))
}

func (g *codeGenerator) visitGoStrExpr(n *nodeGoStrExpr) {
	if g.strategy == compileLayout && n.expr == "contents" {
		// NOTE(paulsmith): this is acting sort of like a coroutine, yielding back to the
		// component that is being rendered with this layout
		g.bodyPrintf(`if fl, ok := w.(http.Flusher); ok {
			fl.Flush()
		}
		`)
		g.bodyPrintf("yield <- struct{}{}\n")
		g.bodyPrintf("<-yield\n")
	} else {
		g.nodeLineNo(n)
		g.bodyPrintf("printEscaped(w, %s)\n", n.expr)
	}
}

func (g *codeGenerator) visitGoCode(n *nodeGoCode) {
	if n.context != inlineGoCode {
		panic(fmt.Sprintf("assertion failure: expected inlineGoCode, got %v", n.context))
	}
	srcLineNo := g.c.lineNo(n.Pos())
	lines := strings.Split(n.code, "\n")
	for _, line := range lines {
		g.lineNo(srcLineNo)
		g.bodyPrintf("%s\n", line)
		srcLineNo++
	}
}

func (g *codeGenerator) visitIf(n *nodeIf) {
	g.bodyPrintf("if %s {\n", n.cond.expr)
	n.then.accept(g)
	if n.alt == nil {
		g.bodyPrintf("}\n")
	} else {
		g.bodyPrintf("} else {\n")
		n.alt.accept(g)
		g.bodyPrintf("}\n")
	}
}

func (g *codeGenerator) visitFor(n *nodeFor) {
	g.bodyPrintf("for %s {\n", n.clause.code)
	n.block.accept(g)
	g.bodyPrintf("}\n")
}

func (g *codeGenerator) visitStmtBlock(n *nodeBlock) {
	for _, e := range n.nodes {
		e.accept(g)
	}
}

func (g *codeGenerator) visitNodes(n []node) {
	for _, e := range n {
		e.accept(g)
	}
}

func (g *codeGenerator) visitLayout(n *nodeLayout) {
	// no-op
}

func (g *codeGenerator) visitImport(n *nodeImport) {
	// no-op
}

var _ nodeVisitor = (*codeGenerator)(nil)

func genCode(c codeGenUnit, basename string, strategy compilationStrategy) ([]byte, error) {
	g := newCodeGenerator(c, basename, strategy)

	// FIXME(paulsmith): need way to specify this as user
	packageName := "build"

	g.outPrintf("// this file is mechanically generated, do not edit!\n")
	g.outPrintf("package %s\n\n", packageName)

	typeName := genStructName(basename, strategy)

	type field struct {
		name string
		typ  string
	}

	fields := []field{
		{name: "pushupFilePath", typ: "string"},
	}

	g.bodyPrintf("type %s struct {\n", typeName)
	for _, field := range fields {
		g.bodyPrintf("%s %s\n", field.name, field.typ)
	}
	g.bodyPrintf("}\n")

	g.bodyPrintf("func (t *%s) buildCliArgs() []string {\n", typeName)
	g.bodyPrintf("  return %#v\n", os.Args)
	g.bodyPrintf("}\n\n")

	switch strategy {
	case compilePushupPage:
		p := c.(*pageCodeGen)
		g.bodyPrintf("func (t *%s) register() {\n", typeName)
		g.bodyPrintf("  routes.add(\"%s\", t)\n", p.route)
		g.bodyPrintf("}\n\n")

		g.bodyPrintf("func init() {\n")
		g.bodyPrintf("  page := new(%s)\n", typeName)
		g.bodyPrintf("  page.pushupFilePath = %s\n", strconv.Quote(c.filePath()))
		g.bodyPrintf("  page.register()\n")
		g.bodyPrintf("}\n\n")
	case compileLayout:
		g.bodyPrintf("func init() {\n")
		g.bodyPrintf("  layout := new(%s)\n", typeName)
		g.bodyPrintf("  layout.pushupFilePath = %s\n", strconv.Quote(c.filePath()))
		g.bodyPrintf("  layouts[\"%s\"] = layout\n", basename)
		g.bodyPrintf("}\n\n")
	}

	// FIXME(paulsmith): feels a bit hacky to have this method in the page interface
	g.bodyPrintf("func (t *%s) filePath() string {\n", typeName)
	g.bodyPrintf("  return t.pushupFilePath\n")
	g.bodyPrintf("}\n\n")

	g.used("io")
	g.used("net/http")
	switch strategy {
	case compilePushupPage:
		g.bodyPrintf("func (t *%s) Respond(w http.ResponseWriter, req *http.Request) error {\n", typeName)
	case compileLayout:
		g.bodyPrintf("func (t *%s) Respond(yield chan struct{}, w http.ResponseWriter, req *http.Request) error {\n", typeName)
	default:
		panic("")
	}

	if strategy == compilePushupPage {
		p := c.(*pageCodeGen)
		if p.layout != "" {
			g.bodyPrintf("  renderLayout := true\n")
		}

		// NOTE(paulsmith): we might want to encapsulate this in its own
		// function/method, but would have to figure out the interplay between
		// user code and control flow, i.e., return an error if the handler
		// wants to skip rendering, redirect, etc.
		if h := p.page.handler; h != nil {
			srcLineNo := g.c.lineNo(h.Pos())
			lines := strings.Split(h.code, "\n")
			for _, line := range lines {
				g.lineNo(srcLineNo)
				g.bodyPrintf("  %s\n", line)
				srcLineNo++
			}
		}

		if p.layout != "" {
			// TODO(paulsmith): this is where a flag that could conditionally toggle the rendering
			// of the layout could go - maybe a special header in request object?
			g.used("sync")
			g.used("log")
			g.bodyPrintf(
				`
				yield := make(chan struct{})
				var wg sync.WaitGroup
				if renderLayout {
					layout := getLayout("%s")
					wg.Add(1)
					go func() {
						if err := layout.Respond(yield, w, req); err != nil {
							log.Printf("error responding with layout: %%v", err)
							panic(err)
						}
						wg.Done()
					}()
					// Let layout render run until its `+transSymStr+`contents is encountered
					<-yield
				}
			`, p.layout)
		}
	}

	// Make a new scope for the user's code block and HTML. This will help (but not fully prevent)
	// name collisions with the surrounding code.
	g.bodyPrintf("// Begin user Go code and HTML\n")
	g.bodyPrintf("{\n")

	g.generate()

	if strategy == compilePushupPage {
		p := c.(*pageCodeGen)
		if p.layout != "" {
			g.bodyPrintf(
				`
				if renderLayout {
					yield <- struct{}{}
					wg.Wait()
				}
			`)
		}
	}

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

	// fmt.Fprintf(os.Stderr, "\x1b[36m%s\x1b[0m", string(raw))

	formatted, err := format.Source(raw)
	if err != nil {
		return nil, fmt.Errorf("gofmt the generated code: %w", err)
	}

	return formatted, nil
}

var structNameIdx int

func safeGoIdentFromFilename(filename string) string {
	// FIXME(paulsmith): need to be more rigorous in mapping safely from
	// filenames to legal Go identifiers
	filename = strings.ReplaceAll(filename, ".", "")
	filename = strings.ReplaceAll(filename, "-", "_")
	filename = strings.ReplaceAll(filename, "$", "DollarSign_")
	return filename
}

func genStructName(basename string, strategy compilationStrategy) string {
	structNameIdx++
	basename = safeGoIdentFromFilename(basename)
	if strategy == compileLayout {
		basename += "_layout"
	}
	return "Pushup__" + basename + "__" + strconv.Itoa(structNameIdx)
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

func parse(source string) (*syntaxTree, error) {
	p := newParser(source)
	tree := p.htmlParser.parseDocument()
	if len(p.errs) > 0 {
		return nil, p.errs[0]
	}
	return tree, nil
}

type parser struct {
	src        string
	offset     int
	errs       []error
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

func (p *parser) source() string {
	return p.sourceFrom(p.offset)
}

func (p *parser) sourceFrom(offset int) string {
	return p.src[offset:]
}

func (p *parser) errorf(format string, args ...any) {
	p.errs = append(p.errs, fmt.Errorf(format, args...))
	log.Printf("\x1b[0;31mERROR: %v\x1b[0m", p.errs[len(p.errs)-1])
}

type htmlParser struct {
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
	tokenizer := html.NewTokenizer(strings.NewReader(p.parser.source()))
	tokenizer.SetMaxBuf(0) // unlimited buffer size
	p.toktyp = tokenizer.Next()
	p.err = tokenizer.Err()
	p.raw = string(tokenizer.Raw())
	p.attrs = nil
	var hasAttr bool
	p.tagname, hasAttr = tokenizer.TagName()
	if hasAttr {
		p.attrs = scanAttrs(p.raw)
	}
	p.start = p.parser.offset
	p.parser.offset += len(p.raw)
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
	for {
		if p.toktyp == html.TextToken && isAllWhitespace(p.raw) {
			n := nodeLiteral{str: p.raw, pos: span{start: p.start, end: p.parser.offset}}
			result = append(result, &n)
			p.advance()
		} else {
			break
		}
	}
	return result
}

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
				nodes = append(nodes, p.parseRawSpan(pos, pos+idx))
				pos += idx
				nameOrValue = nameOrValue[idx:]
			}
			if strings.HasPrefix(nameOrValue, transSymStr+transSymStr) {
				nodes = append(nodes, p.parseRawSpan(pos, pos+1))
				pos += 2
				nameOrValue = nameOrValue[2:]
			} else {
				pos++
				saveParser := p.parser
				p.parser = newParser(nameOrValue[1:])
				nodes = append(nodes, p.transition())
				n := p.parser.offset
				pos += n
				p.parser = saveParser
				nameOrValue = nameOrValue[n:]
			}
		}
	} else {
		nodes = append(nodes, p.parseRawSpan(nameOrValueStartPos, nameOrValueEndPos))
		pos = nameOrValueEndPos
	}
	return nodes, pos
}

func (p *htmlParser) parseRawSpan(start, end int) node {
	e := new(nodeLiteral)
	e.str = p.raw[start:end]
	e.pos.start = p.start + start
	e.pos.end = p.start + end
	return e
}

func (p *htmlParser) parseStartTag() []node {
	var nodes []node

	if len(p.attrs) == 0 {
		nodes = append(nodes, p.parseRawSpan(0, len(p.raw)))
	} else {
		// pos keeps track of how far we've parsed into this p.raw string
		pos := 0

		for _, attr := range p.attrs {
			name := attr.name.string
			value := attr.value.string
			nameStartPos := int(attr.name.start)
			valStartPos := int(attr.value.start)
			nameEndPos := nameStartPos + len(name)
			valEndPos := valStartPos + len(value)

			// emit raw chars between tag name or last attribute and this
			// attribute
			if n := nameStartPos - pos; n > 0 {
				nodes = append(nodes, p.parseRawSpan(pos, pos+n))
				pos += n
			}

			// emit attribute name
			nameNodes, newPos := p.parseAttributeNameOrValue(name, nameStartPos, nameEndPos, pos)
			nodes = append(nodes, nameNodes...)
			pos = newPos

			if valStartPos > pos {
				// emit any chars, including equals and quotes, between
				// attribute name and attribute value, if any
				nodes = append(nodes, p.parseRawSpan(pos, valStartPos))
				pos = valStartPos

				// emit attribute value
				valNodes, newPos := p.parseAttributeNameOrValue(value, valStartPos, valEndPos, pos)
				nodes = append(nodes, valNodes...)
				pos = newPos
			}
		}

		// emit anything from the last attribute to the close of the tag
		nodes = append(nodes, p.parseRawSpan(pos, len(p.raw)))
	}

	return nodes
}

func (p *htmlParser) parseRawLiteral() node {
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
				p.parser.errorf("HTML tokenizer: %w", p.err)
			}
		}
		switch p.toktyp {
		case html.StartTagToken, html.SelfClosingTagToken:
			tree.nodes = append(tree.nodes, p.parseStartTag()...)
		case html.EndTagToken, html.DoctypeToken, html.CommentToken:
			tree.nodes = append(tree.nodes, p.parseRawLiteral())
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
							p.parser.errorf(transSymStr + "layout must be followed by a space")
							break tokenLoop
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
				tree.nodes = append(tree.nodes, p.parseRawLiteral())
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
	return "</" + t.String() + ">"
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
		p.parser.errorf("expected an HTML element start tag, got %s", p.toktyp)
		return result
	}

	result = new(nodeElement)
	result.tag = newTag(p.tagname, p.attrs)
	result.pos.start = p.parser.offset - len(p.raw)
	result.pos.end = p.parser.offset
	result.startTagNodes = p.parseStartTag()
	p.advance()

	result.children = p.parseChildren()

	if !p.match(html.EndTagToken) {
		p.parser.errorf("expected an HTML element end tag, got %q", p.toktyp)
		return result
	}

	if result.tag.name != string(p.tagname) {
		p.parser.errorf("expected </%s> end tag, got </%s>", result.tag.name, p.tagname)
	}

	// <text></text> elements are just for parsing
	if string(p.tagname) == "text" {
		return nodeList(result.children)
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
				p.parser.errorf("HTML tokenizer: %w", p.err)
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
				p.parser.errorf("mismatch end tag, expected </%s>, got </%s>", elem.tag.name, p.tagname)
				return result
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
				result = append(result, p.parseRawLiteral())
			}
			p.advance()
		case html.CommentToken:
			// ???
		case html.DoctypeToken:
			// ???
		default:
			panic("")
		}
	}

	return result
}

type codeParser struct {
	parser         *parser
	baseOffset     int
	file           *token.File
	scanner        *scanner.Scanner
	acceptedToken  goToken
	lookaheadToken goToken
}

func (p *codeParser) reset() {
	p.baseOffset = p.parser.offset
	fset := token.NewFileSet()
	source := p.parser.source()
	// log.Printf("SOURCE: %q", source)
	p.file = fset.AddFile("", fset.Base(), len(source))
	p.scanner = new(scanner.Scanner)
	p.scanner.Init(p.file, []byte(source), p.handleGoScanErr, scanner.ScanComments)
	p.acceptedToken = goToken{}
	p.lookaheadToken = goToken{}
}

func (p *codeParser) sourceFrom(pos token.Pos) string {
	return p.parser.sourceFrom(p.baseOffset + p.file.Offset(pos))
}

func (p *codeParser) lookahead() (t goToken) {
	t.pos, t.tok, t.lit = p.scanner.Scan()
	// log.Printf("POS: %d\tTOK: %v\tLIT: %q", t.pos, t.tok, t.lit)
	return t
}

func (p *codeParser) handleGoScanErr(pos token.Position, msg string) {
	p.parser.errorf("Go scanning error: pos: %v msg: %s", pos, msg)
}

type goToken struct {
	pos token.Pos
	tok token.Token
	lit string
}

func (t goToken) String() string {
	if t.tok.IsLiteral() || t.tok == token.IDENT {
		return t.lit
	}
	return t.tok.String()
}

func (p *codeParser) peek() goToken {
	if p.lookaheadToken.pos == 0 {
		p.lookaheadToken = p.lookahead()
	}
	return p.lookaheadToken
}

func (p *codeParser) prev() goToken {
	return p.acceptedToken
}

func (p *codeParser) sync() goToken {
	t := p.peek()
	p.parser.offset = p.baseOffset + p.file.Offset(t.pos) + len(t.String())
	return t
}

func (p *codeParser) advance() {
	// the Go scanner skips over whitespace so we need to be careful about the
	// logic for advancing the main parser internal source offset.
	t := p.sync()
	p.acceptedToken = t
	p.lookaheadToken = p.lookahead()
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
	if p.peek().tok == token.IF {
		p.advance()
		e = p.parseIfStmt()
	} else if p.peek().tok == token.IDENT && p.peek().lit == "handler" {
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
	} else if p.peek().tok == token.LBRACE {
		e = p.parseCodeBlock()
	} else if p.peek().tok == token.IMPORT {
		p.advance()
		e = p.parseImportKeyword()
	} else if p.peek().tok == token.FOR {
		p.advance()
		e = p.parseForStmt()
	} else if p.peek().tok == token.LPAREN {
		p.advance()
		e = p.parseExplicitExpression()
	} else if p.peek().tok == token.IDENT {
		e = p.parseImplicitExpression()
	} else if p.peek().tok == token.EOF {
		p.parser.errorf("unexpected EOF")
	} else {
		panic("unexpected token type: " + p.peek().tok.String())
	}
	return e
}

func (p *codeParser) parseIfStmt() *nodeIf {
	var stmt nodeIf
	start := p.peek().pos
loop:
	for {
		switch p.peek().tok {
		case token.EOF:
			p.parser.errorf("premature end of conditional in IF statement")
			break loop
		case token.LBRACE:
			// conditional expression has been scanned
			break loop
		// TODO(paulsmith): add cases for tokens that are illegal in an expression
		default:
			p.advance()
		}
	}
	n := (p.file.Offset(p.prev().pos) - p.file.Offset(start)) + len(p.prev().String())
	offset := p.baseOffset + p.file.Offset(start)
	stmt.cond = new(nodeGoStrExpr)
	stmt.cond.pos.start = offset
	stmt.cond.pos.end = offset + n
	stmt.cond.expr = p.sourceFrom(start)[:n]
	if _, err := goparser.ParseExpr(stmt.cond.expr); err != nil {
		p.parser.errorf("parsing Go expression in IF conditional: %w", err)
	}
	stmt.then = p.parseStmtBlock()
	if p.peek().tok == token.ELSE {
		p.advance()
		elseBlock := p.parseStmtBlock()
		stmt.alt = elseBlock
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
			p.parser.errorf("premature end of clause in FOR statement")
			break loop
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
		panic("")
	}
	p.advance()
	var block *nodeBlock
	// it is likely non-Go code (i.e., HTML, or HTML and a transition)
	switch p.peek().tok {
	case token.ILLEGAL:
		if p.peek().lit == transSymStr {
			p.scanner.ErrorCount--
			p.advance()
			// we can just stay in the code parser
			code := p.parseCode()
			block = &nodeBlock{nodes: []node{code}}
		}
	case token.EOF:
		p.parser.errorf("premature end of block in IF statement")
	default:
		block = p.transition()
	}
	// we should be at the closing '}' token here
	if p.peek().tok != token.RBRACE {
		if p.peek().tok == token.LSS {
			p.parser.errorf("there must be a single HTML element inside a Go code block, try wrapping them")
		} else {
			p.parser.errorf("expected closing '}', got %v", p.peek())
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
		p.parser.errorf("expected '{', got '%s'", p.peek().tok)
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

func (p *codeParser) parseCodeBlock() *nodeGoCode {
	result := &nodeGoCode{context: inlineGoCode}
	if p.peek().tok != token.LBRACE {
		p.parser.errorf("expected '{', got '%s'", p.peek().tok)
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
			p.parser.errorf("expected string, got %s", p.peek().tok)
			return e
		}
		e.decl.path = p.peek().lit
	case token.PERIOD:
		e.decl.pkgName = "."
		p.advance()
		if p.peek().tok != token.STRING {
			p.parser.errorf("expected string, got %s", p.peek().tok)
			return e
		}
		e.decl.path = p.peek().lit
	default:
		p.parser.errorf("unexpected token type after "+transSymStr+"import: %s", p.peek().tok)
	}
	return e
}

func (p *codeParser) parseExplicitExpression() *nodeGoStrExpr {
	// one token past the opening '('
	result := new(nodeGoStrExpr)
	result.pos.start = p.parser.offset
	start := p.peek().pos
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
		default:
		}
		p.advance()
	}
	n := (p.file.Offset(p.prev().pos) - p.file.Offset(start)) + len(p.prev().String())
	if p.peek().tok != token.RPAREN {
		panic("")
	}
	_ = p.sync()
	result.expr = p.sourceFrom(start)[:n]
	result.pos.end = result.pos.start + n
	if _, err := goparser.ParseExpr(result.expr); err != nil {
		p.parser.errorf("illegal Go expression: %w", err)
	}
	return result
}

func (p *codeParser) parseImplicitExpression() *nodeGoStrExpr {
	if p.peek().tok != token.IDENT {
		panic("")
	}
	result := new(nodeGoStrExpr)
	result.pos.start = p.parser.offset
	start := p.peek().pos
	n := len(p.peek().String())
	p.advance()
	for {
		if p.peek().tok == token.PERIOD {
			p.advance()
			if p.peek().tok == token.IDENT {
				n += 1 + len(p.peek().String())
				p.advance()
			} else {
				p.parser.errorf("illegal selector expression")
				break
			}
		} else {
			break
		}
	}
	result.expr = p.sourceFrom(start)[:n]
	result.pos.end = result.pos.start + n
	if _, err := goparser.ParseExpr(result.expr); err != nil {
		p.parser.errorf("illegal Go expression: %w", err)
	}
	return result
}

type debugPrettyPrinter struct {
	w     io.Writer
	depth int
}

var _ nodeVisitor = (*debugPrettyPrinter)(nil)

const pad = "    "

func acceptAndIndent(n node, p *debugPrettyPrinter) {
	p.depth++
	n.accept(p)
	p.depth--
}

func (p *debugPrettyPrinter) visitLiteral(n *nodeLiteral) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[32m%q\x1b[0m", n.str)
	fmt.Fprintln(p.w, "")
}

func (p *debugPrettyPrinter) visitGoStrExpr(n *nodeGoStrExpr) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[33m%s\x1b[0m", n.expr)
	fmt.Fprintln(p.w, "")
}

func (p *debugPrettyPrinter) visitGoCode(n *nodeGoCode) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[34m%s\x1b[0m", n.code)
	fmt.Fprintln(p.w, "")
}

func (p *debugPrettyPrinter) visitIf(n *nodeIf) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[35mIF\x1b[0m\n")
	acceptAndIndent(n.cond, p)
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[35mTHEN\x1b[0m\n")
	acceptAndIndent(n.then, p)
	if n.alt != nil {
		p.w.Write([]byte(strings.Repeat(pad, p.depth)))
		fmt.Fprintf(p.w, "\x1b[1;35mELSE\x1b[0m\n")
		acceptAndIndent(n.alt, p)
	}
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[35mEND IF\x1b[0m\n")
}

func (p *debugPrettyPrinter) visitFor(n *nodeFor) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[36mFOR\x1b[0m\n")
	acceptAndIndent(n.clause, p)
	acceptAndIndent(n.block, p)
}

func (p *debugPrettyPrinter) visitElement(n *nodeElement) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[31m%s\x1b[0m\n", n.tag.start())
	for _, e := range n.children {
		acceptAndIndent(e, p)
	}
	fmt.Fprintf(p.w, "\x1b[31m%s\x1b[0m\n", n.tag.end())
}

func (p *debugPrettyPrinter) visitStmtBlock(n *nodeBlock) {
	nodeList(n.nodes).accept(p)
}

func (p *debugPrettyPrinter) visitNodes(nodes []node) {
	for _, n := range nodes {
		acceptAndIndent(n, p)
	}
}

func (p *debugPrettyPrinter) visitImport(n *nodeImport) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "IMPORT ")
	if n.decl.pkgName != "" {
		fmt.Fprintf(p.w, "%s", n.decl.pkgName)
	}
	fmt.Fprintf(p.w, "%s\n", n.decl.path)
}

func (p *debugPrettyPrinter) visitLayout(n *nodeLayout) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "LAYOUT %s\n", n.name)
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

func scanAttrs(openTag string) []*attr {
	l := newOpenTagLexer(openTag)
	result := l.scan()
	return result
}

type openTagLexer struct {
	raw         string
	pos         int
	state       openTagLexState
	returnState openTagLexState
	charRefBuf  bytes.Buffer
	attrs       []*attr
	currAttr    *attr
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

func newOpenTagLexer(source string) *openTagLexer {
	l := new(openTagLexer)
	l.raw = source
	l.state = openTagLexData
	return l
}

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
	default:
		panic("")
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
			ch := l.consumeNextChar()
			switch {
			case ch == '<':
				l.switchState(openTagLexTagOpen)
			default:
				l.assertionFailure("found '%c' in data state, expected '<'", ch)
			}
		// 13.2.5.6 Tag open state
		// https://html.spec.whatwg.org/multipage/parsing.html#tag-open-state
		case openTagLexTagOpen:
			ch := l.consumeNextChar()
			switch {
			case ch == '!':
				l.assertionFailure("input '%c' switch to markup declaration open state", ch)
			case ch == '/':
				l.assertionFailure("input '%c' switch to end tag open state", ch)
			case isASCIIAlpha(ch):
				l.reconsumeIn(openTagLexTagName)
			case ch == '?':
				l.assertionFailure("input '%c' parse error", ch)
			case ch == eof:
				l.assertionFailure("eof before tag name parse error")
			default:
				l.assertionFailure("found '%c' in tag open state", ch)
			}
		// 13.2.5.8 Tag name state
		// https://html.spec.whatwg.org/multipage/parsing.html#tag-name-state
		case openTagLexTagName:
			ch := l.consumeNextChar()
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
				l.assertionFailure("found null in tag name state")
			case ch == eof:
				l.assertionFailure("found eof in tag name state")
			default:
				// append current input char to current tag token's tag name
			}
		// 13.2.5.32 Before attribute name state
		// https://html.spec.whatwg.org/multipage/parsing.html#before-attribute-name-state
		case openTagLexBeforeAttrName:
			ch := l.consumeNextChar()
			switch {
			case ch == '\t' || ch == '\n' || ch == '\f' || ch == ' ':
				// ignore
			case ch == '/' || ch == '>' || ch == eof:
				l.reconsumeIn(openTagLexAfterAttrName)
			case ch == '=':
				l.assertionFailure("found '%c' in before attribute name state", ch)
			default:
				l.newAttr()
				l.reconsumeIn(openTagLexAttrName)
			}
		// 13.2.5.33 Attribute name state
		// https://html.spec.whatwg.org/multipage/parsing.html#attribute-name-state
		case openTagLexAttrName:
			ch := l.consumeNextChar()
			switch {
			case ch == '\t' || ch == '\n' || ch == '\f' || ch == ' ' || ch == '/' || ch == '>' || ch == eof:
				defer l.cmpAttrName()
				l.reconsumeIn(openTagLexAfterAttrName)
			case ch == '=':
				defer l.cmpAttrName()
				l.switchState(openTagLexBeforeAttrVal)
			case isASCIIUpper(ch):
				// append lowercase version (add 0x20) of current input character to current attr's name
				l.appendCurrName(byte(ch + 0x20))
			case ch == 0:
				l.assertionFailure("found null in attribute name state")
			case ch == '"' || ch == '\'' || ch == '<':
				l.parseError("unexpected-character-in-attribute-name")
				// append current input character to current attribute's name
				l.appendCurrName(byte(ch))
			default:
				// append current input character to current attribute's name
				l.appendCurrName(byte(ch))
			}
		// 13.2.5.34 After attribute name state
		// https://html.spec.whatwg.org/multipage/parsing.html#after-attribute-name-state
		case openTagLexAfterAttrName:
			ch := l.consumeNextChar()
			switch {
			case ch == '\t' || ch == '\n' || ch == '\f' || ch == ' ':
				// ignore
			case ch == '/':
				l.switchState(openTagLexSelfClosingStartTag)
			case ch == '=':
				l.switchState(openTagLexBeforeAttrVal)
			case ch == '>':
				break loop
			case ch == eof:
				l.assertionFailure("found EOF in after attribute name state")
			default:
				l.newAttr()
				l.reconsumeIn(openTagLexAttrName)
			}
		// 13.2.5.35 Before attribute value state
		// https://html.spec.whatwg.org/multipage/parsing.html#before-attribute-value-state
		case openTagLexBeforeAttrVal:
			ch := l.consumeNextChar()
			switch {
			case ch == '\t' || ch == '\n' || ch == '\f' || ch == ' ':
				// ignore
			case ch == '"':
				l.switchState(openTagLexAttrValDoubleQuote)
			case ch == '\'':
				l.switchState(openTagLexAttrValSingleQuote)
			case ch == '>':
				l.parseError("missing-attribute-value")
				break loop
			default:
				l.reconsumeIn(openTagLexAttrValUnquoted)
			}
		// 13.2.5.36 Attribute value (double-quoted) state
		// https://html.spec.whatwg.org/multipage/parsing.html#attribute-value-(double-quoted)-state
		case openTagLexAttrValDoubleQuote:
			ch := l.consumeNextChar()
			switch {
			case ch == '"':
				l.switchState(openTagLexAfterAttrValQuoted)
			case ch == '&':
				l.returnState = openTagLexAttrValDoubleQuote
				l.switchState(openTagLexCharRef)
			case ch == 0:
				l.assertionFailure("found null in attribute value (double-quoted) state")
			case ch == eof:
				l.assertionFailure("found EOF in tag")
			default:
				l.appendCurrVal(byte(ch))
			}
		// 13.2.5.37 Attribute value (single-quoted) state
		// https://html.spec.whatwg.org/multipage/parsing.html#attribute-value-(single-quoted)-state
		case openTagLexAttrValSingleQuote:
			ch := l.consumeNextChar()
			switch {
			case ch == '"':
				l.switchState(openTagLexAfterAttrValQuoted)
			case ch == '&':
				l.returnState = openTagLexAttrValSingleQuote
				l.switchState(openTagLexCharRef)
			case ch == 0:
				l.assertionFailure("found null in attribute value (single-quoted) state")
			case ch == eof:
				l.assertionFailure("found EOF in tag")
			default:
				l.appendCurrVal(byte(ch))
			}
		// 13.2.5.38 Attribute value (unquoted) state
		// https://html.spec.whatwg.org/multipage/parsing.html#attribute-value-(unquoted)-state
		case openTagLexAttrValUnquoted:
			ch := l.consumeNextChar()
			switch {
			case ch == '\t' || ch == '\n' || ch == '\f' || ch == ' ':
				l.switchState(openTagLexBeforeAttrName)
			case ch == '&':
				l.returnState = openTagLexAttrValUnquoted
				l.switchState(openTagLexCharRef)
			case ch == '>':
				break loop
			case ch == 0:
				l.assertionFailure("found null in attribute value (unquoted) state")
			case ch == '"' || ch == '\'' || ch == '<' || ch == '=' || ch == '`':
				l.parseError("unexpected-null-character")
				l.appendCurrVal(byte(ch))
			case ch == eof:
				l.assertionFailure("found EOF in tag")
			default:
				l.appendCurrVal(byte(ch))
			}
		// 13.2.5.39 After attribute value (quoted) state
		// https://html.spec.whatwg.org/multipage/parsing.html#after-attribute-value-(quoted)-state
		case openTagLexAfterAttrValQuoted:
			ch := l.consumeNextChar()
			switch {
			case ch == '\t' || ch == '\n' || ch == '\f' || ch == ' ':
				l.switchState(openTagLexBeforeAttrName)
			case ch == '/':
				l.switchState(openTagLexSelfClosingStartTag)
			case ch == '>':
				break loop
			case ch == eof:
				l.assertionFailure("found EOF in tag")
			default:
				l.parseError("missing-whitespace-between-attributes")
				l.reconsumeIn(openTagLexBeforeAttrName)
			}
		// 13.2.5.72 Character reference state
		// https://html.spec.whatwg.org/multipage/parsing.html#character-reference-state
		case openTagLexCharRef:
			l.charRefBuf = bytes.Buffer{}
			l.charRefBuf.WriteByte('&')
			ch := l.consumeNextChar()
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
			ch := l.consumeNextChar()
			switch {
			case ch == '>':
				break loop
			case ch == eof:
				l.assertionFailure("found EOF in tag")
			default:
				l.parseError("unexpected-solidus-in-tag")
				l.reconsumeIn(openTagLexBeforeAttrName)
			}
		default:
			panic("open tag lex state " + l.state.String())
		}
	}

	return l.attrs
}

func (l *openTagLexer) consumeNextChar() int {
	var ch int
	if l.pos < len(l.raw) {
		ch = int(l.raw[l.pos])
		l.pos++
	} else {
		ch = eof
	}
	return ch
}

func (l *openTagLexer) flushCharRef() {
	b := l.charRefBuf.Bytes()
	for _, bb := range b {
		l.appendCurrVal(bb)
	}
}

func (l *openTagLexer) newAttr() {
	a := new(attr)
	l.attrs = append(l.attrs, a)
	l.currAttr = a
}

func (l *openTagLexer) appendCurrName(ch byte) {
	if l.currAttr.name.start == 0 {
		l.currAttr.name.start = pos(l.pos - 1)
	}
	l.currAttr.name.string += string(ch)
}

func (l *openTagLexer) appendCurrVal(ch byte) {
	if l.currAttr.value.start == 0 {
		l.currAttr.value.start = pos(l.pos - 1)
	}
	l.currAttr.value.string += string(ch)
}

func (l *openTagLexer) assertionFailure(format string, args ...any) {
	err := fmt.Errorf(format, args...)
	// FIXME(paulsmith): handle in regular control flow
	panic(err)
}

func (l *openTagLexer) parseError(name string) {
	switch name {
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
	default:
		log.Printf("parse error: %s", name)
	}
}

func (l *openTagLexer) reconsumeIn(state openTagLexState) {
	l.backup()
	l.switchState(state)
}

func (l *openTagLexer) backup() {
	if l.pos > 1 {
		l.pos--
	} else {
		panic("underflowed")
	}
}

func (l *openTagLexer) exitingState(state openTagLexState) {
	// log.Printf("<- %s", state)
}

func (l *openTagLexer) enteringState(state openTagLexState) {
	// log.Printf("-> %s", state)
}

func (l *openTagLexer) switchState(state openTagLexState) {
	l.exitingState(l.state)
	l.enteringState(state)
	l.state = state
}

func (l *openTagLexer) cmpAttrName() {
	for i := range l.attrs {
		if l.currAttr.name == l.attrs[i].name {
			l.parseError("duplicate-attribute")
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
