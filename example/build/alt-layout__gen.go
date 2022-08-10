// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"sync"
)

type Pushup__alt_layout__4 struct {
	pushupFilePath string
}

func (t *Pushup__alt_layout__4) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__alt_layout__4) register() {
	routes.add("/alt-layout", t)
}

func init() {
	page := new(Pushup__alt_layout__4)
	page.pushupFilePath = "alt-layout.pushup"
	page.register()
}

func (t *Pushup__alt_layout__4) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__alt_layout__4) Respond(w http.ResponseWriter, req *http.Request) error {
	renderLayout := true

	yield := make(chan struct{})
	var wg sync.WaitGroup
	if renderLayout {
		layout := getLayout("netscape")
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
//line alt-layout.pushup:2
		io.WriteString(w, "\n\n")
//line alt-layout.pushup:3
		io.WriteString(w, "<p>")
//line alt-layout.pushup:3
		io.WriteString(w, "Greetings.")
//line alt-layout.pushup:3
		io.WriteString(w, "</p>")
//line alt-layout.pushup:4
		io.WriteString(w, "\n")
//line alt-layout.pushup:4
		io.WriteString(w, "<p>")
//line alt-layout.pushup:4
		io.WriteString(w, "This component has a non-default layout, specified with the ")
//line alt-layout.pushup:4
		io.WriteString(w, "^")
//line alt-layout.pushup:4
		io.WriteString(w, "layout directive.")
//line alt-layout.pushup:4
		io.WriteString(w, "</p>")
//line alt-layout.pushup:5
		io.WriteString(w, "\n")
//line alt-layout.pushup:5
		io.WriteString(w, "<p>")
//line alt-layout.pushup:5
		io.WriteString(w, "<a ")
//line alt-layout.pushup:5
		io.WriteString(w, "href")
//line alt-layout.pushup:5
		io.WriteString(w, "=\"")
//line alt-layout.pushup:5
		io.WriteString(w, "/")
//line alt-layout.pushup:5
		io.WriteString(w, "\" ")
//line alt-layout.pushup:5
		io.WriteString(w, "hx-boost")
//line alt-layout.pushup:5
		io.WriteString(w, "=\"")
//line alt-layout.pushup:5
		io.WriteString(w, "false")
//line alt-layout.pushup:5
		io.WriteString(w, "\">")
//line alt-layout.pushup:5
		io.WriteString(w, "Back home")
//line alt-layout.pushup:5
		io.WriteString(w, "</a>")
//line alt-layout.pushup:5
		io.WriteString(w, "</p>")
//line alt-layout.pushup:6
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
