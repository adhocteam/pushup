package devserver

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/net/context"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type DevServer struct {
	root           string
	port           int
	watcher        *fileWatcher
	reloader       *reloader
	childCmd       *exec.Cmd
	socketListener net.Listener
}

func New(root string, port int) *DevServer {
	return &DevServer{
		root:     root,
		port:     port,
		reloader: newReloader(),
	}
}

func (s *DevServer) debounceEvents(ctx context.Context, interval time.Duration, fn func(fsnotify.Event)) {
	timer := time.NewTimer(0)
	timer.Stop()

	var pendingEvent fsnotify.Event
	var hasPending bool

	for {
		select {
		case event := <-s.watcher.watcher.Events:
			slog.Debug("file watcher event", "event", event)
			if !s.filterEvent(event) {
				continue
			}

			pendingEvent = event
			hasPending = true

			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(interval)

		case <-timer.C:
			if hasPending {
				fn(pendingEvent)
				hasPending = false
			}

		case err := <-s.watcher.watcher.Errors:
			slog.Error("file watcher error", "error", err)
			return

		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (s *DevServer) filterEvent(event fsnotify.Event) bool {
	if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
		return false
	}

	ext := filepath.Ext(event.Name)

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

const wsDevReloadURLPath = "/@pushup/reload"

func (s *DevServer) Run(ctx context.Context) error {
	var err error
	s.watcher, err = newFileWatcher(s.root)
	if err != nil {
		return fmt.Errorf("creating new file watcher: %w", err)
	}

	go s.debounceEvents(ctx, 125*time.Millisecond, func(event fsnotify.Event) {
		slog.Info("got reloading event", "event", event)
		s.watcher.watcher.Close()
		if err := s.rebuildAndReload(ctx); err != nil {
			slog.Error("rebuild failed", "error", err)
			return
		}
		s.reloader.triggerReload()
		// TODO: dedupe this with the above
		var err error
		s.watcher, err = newFileWatcher(s.root)
		if err != nil {
			slog.Error("creating new file watcher", "error", err)
		}
	})

	var socketPath string
	socketPath, s.socketListener, err = s.createUnixSocket()
	if err != nil {
		return fmt.Errorf("creating Unix socket: %w", err)
	}

	addr := "0.0.0.0:" + strconv.Itoa(s.port)
	proxyTCPListener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listening on port %d: %w", s.port, err)
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
	proxy.ModifyResponse = s.modifyResponse

	mux := http.NewServeMux()
	mux.Handle("/", proxy)
	mux.Handle(wsDevReloadURLPath, s.reloader)

	server := http.Server{Handler: mux}
	fmt.Fprintf(os.Stdout, "\x1b[1;36m↑↑ PUSHUP DEV RELOADER ON http://%s ↑↑\x1b[0m\n", addr)
	// FIXME: shutdown
	go server.Serve(proxyTCPListener)

	if err := s.rebuildAndReload(ctx); err != nil {
		return fmt.Errorf("starting child server failed: %w", err)
	}

	defer func() {
		if err := s.Cleanup(); err != nil {
			slog.Error("cleanup failed", "error", err)
		}
	}()

	<-ctx.Done()
	return server.Shutdown(context.Background())
}

func (s *DevServer) modifyResponse(res *http.Response) error {
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
		doc, err := s.appendDevReloaderScript(res.Body)
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

func (s *DevServer) appendDevReloaderScript(r io.Reader) (*html.Node, error) {
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

var devReloaderScript = fmt.Sprintf(`
(function() {
    var socket = new WebSocket('ws://' + window.location.host + '%s');
    socket.onclose = function() {
        console.log('dev server disconnected - attempting to reconnect ...');
        setTimeout(function() { location.reload(); }, 1000);
    };
    socket.onmessage = function(event) {
        if (event.data === '%s') {
            location.reload();
        }
    };
})();
`, wsDevReloadURLPath, reloadMsg)

func (s *DevServer) rebuildAndReload(ctx context.Context) error {
	if s.childCmd != nil && s.childCmd.Process != nil {
		if err := s.childCmd.Process.Signal(os.Interrupt); err != nil {
			slog.Warn("failed to interrupt child process", "error", err)
			s.childCmd.Process.Kill()
		}
		slog.Debug("waiting for childing process", "pid", s.childCmd.Process.Pid)
		s.childCmd.Process.Wait()
	}

	slog.Debug("Pushup build")
	{
		// TODO: use command.Build() instead but currently would create an import cycle
		buildCmd := exec.Command("pushup", "build")
		buildCmd.Dir = s.root
		if output, err := buildCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("pushup build failed: %s, %w", string(output), err)
		}
	}

	binPath := "." + string(filepath.Separator) + s.devBinaryName()
	{
		buildCmd := exec.Command("go", "build", "-o", binPath)
		buildCmd.Dir = s.root
		if output, err := buildCmd.CombinedOutput(); err != nil {
			os.Remove(binPath)
			return fmt.Errorf("go build failed: %s, %w", string(output), err)
		}
	}

	if err := s.createChildServer(ctx, binPath); err != nil {
		return fmt.Errorf("creating child server: %w", err)
	}

	return nil
}

func (s *DevServer) devBinaryName() string {
	if runtime.GOOS == "windows" {
		return ".pushup-dev.exe"
	}
	return ".pushup-dev"
}

func (s *DevServer) createChildServer(ctx context.Context, binPath string) error {
	socketFile, err := s.socketListener.(*net.UnixListener).File()
	if err != nil {
		return fmt.Errorf("getting file of Unix socket: %w", err)
	}

	s.childCmd = exec.CommandContext(ctx, binPath)
	s.childCmd.Stdout = os.Stdout
	s.childCmd.Stderr = os.Stderr
	s.childCmd.ExtraFiles = append(s.childCmd.ExtraFiles, socketFile)
	s.childCmd.Env = append(os.Environ(), "PUSHUP_LISTENER_FD=3")

	return s.childCmd.Start()
}

func (s *DevServer) Cleanup() error {
	binPath := filepath.Join(s.root, s.devBinaryName())
	if err := os.Remove(binPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove dev binary: %w", err)
	}
	return nil
}

func (s *DevServer) createUnixSocket() (string, net.Listener, error) {
	path := filepath.Join("/tmp", "pushup-"+strconv.Itoa(os.Getpid())+".sock")
	os.Remove(path)
	slog.Debug("unix socket", "path", path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return "", nil, fmt.Errorf("creating Unix socket: %w", err)
	}
	os.Chmod(path, 0666)
	return path, ln, nil
}
