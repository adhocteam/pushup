// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type Pushup__source__24 struct {
	pushupFilePath string
}

func (t *Pushup__source__24) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__source__24) register() {
	routes.add("/source", t)
}

func init() {
	page := new(Pushup__source__24)
	page.pushupFilePath = "source.pushup"
	page.register()
}

func (t *Pushup__source__24) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__source__24) Respond(w http.ResponseWriter, req *http.Request) error {
	renderLayout := true
//line source.pushup:6
	path := req.FormValue("route")
//line source.pushup:7
	route := getRouteFromPath(path)
//line source.pushup:8
	if route == nil {
//line source.pushup:9
		http.Error(w, http.StatusText(404), 404)
//line source.pushup:10
		return nil
//line source.pushup:11
	}
//line source.pushup:12
	fpath := filepath.Join("app", "pages", route.page.filePath())
//line source.pushup:13
	b, err := os.ReadFile(fpath)
//line source.pushup:14
	if err != nil {
//line source.pushup:15
		log.Printf("reading file %s: %v", fpath, err)
//line source.pushup:16
		return err
//line source.pushup:17
	}
//line source.pushup:18
	source := string(b)
//line source.pushup:19

	yield := make(chan struct{})
	var wg sync.WaitGroup
	if renderLayout {
		layout := getLayout("default")
		wg.Add(1)
		go func() {
			if err := layout.Respond(yield, w, req); err != nil {
				log.Printf("error responding with layout: %v", err)
				panic(err)
			}
			wg.Done()
		}()
		// Let layout render run until its ^contents is encountered
		<-yield
	}
	// Begin user Go code and HTML
	{
//line source.pushup:2
		io.WriteString(w, "\n")
//line source.pushup:3
		io.WriteString(w, "\n")
//line source.pushup:4
		io.WriteString(w, "\n\n")
//line source.pushup:20
		io.WriteString(w, "\n\n")
//line source.pushup:21
		io.WriteString(w, "<p>")
//line source.pushup:21
		io.WriteString(w, "Source code for Pushup page ")
//line source.pushup:21
		io.WriteString(w, "<a ")
//line source.pushup:21
		io.WriteString(w, "href")
//line source.pushup:21
		io.WriteString(w, "=\"")
//line source.pushup:1
		printEscaped(w, path)
//line source.pushup:21
		io.WriteString(w, "\">")
//line source.pushup:21
		io.WriteString(w, "<b>")
//line source.pushup:21
		printEscaped(w, path)
//line source.pushup:21
		io.WriteString(w, "</b>")
//line source.pushup:21
		io.WriteString(w, "</a>")
//line source.pushup:21
		io.WriteString(w, ":")
//line source.pushup:21
		io.WriteString(w, "</p>")
//line source.pushup:22
		io.WriteString(w, "\n\n")
//line source.pushup:23
		io.WriteString(w, "<style>")
//line source.pushup:24
		io.WriteString(w, "\npre {\n    padding: 1em;\n    background: #f3f4f0;\n    overflow-x: scroll;\n}\n")
//line source.pushup:29
		io.WriteString(w, "</style>")
//line source.pushup:30
		io.WriteString(w, "\n\n")
//line source.pushup:31
		io.WriteString(w, "<pre>")
//line source.pushup:31
		io.WriteString(w, "<code>")
//line source.pushup:31
		printEscaped(w, source)
//line source.pushup:31
		io.WriteString(w, "</code>")
//line source.pushup:31
		io.WriteString(w, "</pre>")
//line source.pushup:32
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
