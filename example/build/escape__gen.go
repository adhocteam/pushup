// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"sync"
)

type Pushup__escape__12 struct {
	pushupFilePath string
}

func (t *Pushup__escape__12) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__escape__12) register() {
	routes.add("/escape", t)
}

func init() {
	page := new(Pushup__escape__12)
	page.pushupFilePath = "escape.pushup"
	page.register()
}

func (t *Pushup__escape__12) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__escape__12) Respond(w http.ResponseWriter, req *http.Request) error {
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
//line escape.pushup:2
		emotion := "<em><3</em> & <strong>blue circle</strong>"
//line escape.pushup:3

//line escape.pushup:4
		io.WriteString(w, "\n\n")
//line escape.pushup:5
		io.WriteString(w, "<p ")
//line escape.pushup:5
		io.WriteString(w, "class")
//line escape.pushup:5
		io.WriteString(w, "=\"")
//line escape.pushup:5
		io.WriteString(w, "greeting")
//line escape.pushup:5
		io.WriteString(w, "\">")
//line escape.pushup:5
		io.WriteString(w, "I ")
//line escape.pushup:5
		printEscaped(w, emotion)
//line escape.pushup:5
		io.WriteString(w, " Chicago!")
//line escape.pushup:5
		io.WriteString(w, "</p>")
//line escape.pushup:6
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
