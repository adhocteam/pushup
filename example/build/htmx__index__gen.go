// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"sync"
)

type Pushup__index__17 struct {
	pushupFilePath string
}

func (t *Pushup__index__17) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__index__17) register() {
	routes.add("/htmx", t)
}

func init() {
	page := new(Pushup__index__17)
	page.pushupFilePath = "htmx/index.pushup"
	page.register()
}

func (t *Pushup__index__17) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__index__17) Respond(w http.ResponseWriter, req *http.Request) error {
	renderLayout := true

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
//line index.pushup:1
		io.WriteString(w, "<h1>")
//line index.pushup:1
		io.WriteString(w, "htmx examples")
//line index.pushup:1
		io.WriteString(w, "</h1>")
//line index.pushup:2
		io.WriteString(w, "\n\n")
//line index.pushup:3
		io.WriteString(w, "<ul>")
//line index.pushup:4
		io.WriteString(w, "\n    ")
//line index.pushup:4
		io.WriteString(w, "<li>")
//line index.pushup:4
		io.WriteString(w, "<a ")
//line index.pushup:4
		io.WriteString(w, "href")
//line index.pushup:4
		io.WriteString(w, "=\"")
//line index.pushup:4
		io.WriteString(w, "/htmx/click-to-load")
//line index.pushup:4
		io.WriteString(w, "\">")
//line index.pushup:4
		io.WriteString(w, "Click to load")
//line index.pushup:4
		io.WriteString(w, "</a>")
//line index.pushup:4
		io.WriteString(w, "</li>")
//line index.pushup:5
		io.WriteString(w, "\n    ")
//line index.pushup:5
		io.WriteString(w, "<li>")
//line index.pushup:5
		io.WriteString(w, "<a ")
//line index.pushup:5
		io.WriteString(w, "href")
//line index.pushup:5
		io.WriteString(w, "=\"")
//line index.pushup:5
		io.WriteString(w, "/htmx/value-select")
//line index.pushup:5
		io.WriteString(w, "\">")
//line index.pushup:5
		io.WriteString(w, "Value select")
//line index.pushup:5
		io.WriteString(w, "</a>")
//line index.pushup:5
		io.WriteString(w, "</li>")
//line index.pushup:6
		io.WriteString(w, "\n    ")
//line index.pushup:6
		io.WriteString(w, "<li>")
//line index.pushup:6
		io.WriteString(w, "<a ")
//line index.pushup:6
		io.WriteString(w, "href")
//line index.pushup:6
		io.WriteString(w, "=\"")
//line index.pushup:6
		io.WriteString(w, "/htmx/active-search")
//line index.pushup:6
		io.WriteString(w, "\">")
//line index.pushup:6
		io.WriteString(w, "Active search")
//line index.pushup:6
		io.WriteString(w, "</a>")
//line index.pushup:6
		io.WriteString(w, "</li>")
//line index.pushup:7
		io.WriteString(w, "\n")
//line index.pushup:7
		io.WriteString(w, "</ul>")
//line index.pushup:8
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
