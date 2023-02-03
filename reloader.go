package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func watchForReload(ctx context.Context, root string, reload chan struct{}) {
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
		reload <- struct{}{}
		watcher.Close()
	})

	if err := watchDirRecursively(watcher, root); err != nil {
		panic(fmt.Errorf("adding dir to watch: %w", err))
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
	//nolint:errcheck
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
			//nolint:errcheck
			w.Write([]byte("event: reload\ndata: \n\n"))
		case <-r.Context().Done():
			if d.verboseLogging {
				log.Printf("client disconnected")
			}
			close(done)
			break loop
		case <-time.After(1 * time.Second):
			//nolint:errcheck
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
