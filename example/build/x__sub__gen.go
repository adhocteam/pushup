// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"sync"
)

type Pushup__sub__25 struct {
	pushupFilePath string
}

func (t *Pushup__sub__25) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__sub__25) register() {
	routes.add("/x/sub", t)
}

func init() {
	page := new(Pushup__sub__25)
	page.pushupFilePath = "x/sub.pushup"
	page.register()
}

func (t *Pushup__sub__25) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__sub__25) Respond(w http.ResponseWriter, req *http.Request) error {
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
//line sub.pushup:1
		io.WriteString(w, "<p>")
//line sub.pushup:1
		io.WriteString(w, "This is a page showing how filesystem-based routing works with subdirectories")
//line sub.pushup:1
		io.WriteString(w, "</p>")
//line sub.pushup:2
		io.WriteString(w, "\n")
//line sub.pushup:2
		io.WriteString(w, "<p>")
//line sub.pushup:2
		io.WriteString(w, "URL path: ")
//line sub.pushup:2
		printEscaped(w, req.URL.Path)
//line sub.pushup:2
		io.WriteString(w, "</p>")
//line sub.pushup:3
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
