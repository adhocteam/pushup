// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"sync"
)

type Pushup__dump__10 struct {
	pushupFilePath string
}

func (t *Pushup__dump__10) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__dump__10) register() {
	routes.add("/dump", t)
}

func init() {
	page := new(Pushup__dump__10)
	page.pushupFilePath = "dump.pushup"
	page.register()
}

func (t *Pushup__dump__10) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__dump__10) Respond(w http.ResponseWriter, req *http.Request) error {
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
//line dump.pushup:1
		io.WriteString(w, "<h2>")
//line dump.pushup:1
		io.WriteString(w, "Dump request headers")
//line dump.pushup:1
		io.WriteString(w, "</h2>")
//line dump.pushup:2
		io.WriteString(w, "\n")
//line dump.pushup:2
		io.WriteString(w, "<ul>")
//line dump.pushup:3
		io.WriteString(w, "\n")
		for key := range req.Header {
//line dump.pushup:4
			io.WriteString(w, "\n    ")
//line dump.pushup:4
//line dump.pushup:4
			io.WriteString(w, "<li>")
//line dump.pushup:5
			io.WriteString(w, "\n        ")
//line dump.pushup:5
//line dump.pushup:5
			io.WriteString(w, "<b>")
//line dump.pushup:5
			printEscaped(w, key)
			io.WriteString(w, "</b>")
//line dump.pushup:5
			io.WriteString(w, ": ")
//line dump.pushup:5
			printEscaped(w, req.Header.Get(key))
//line dump.pushup:6
			io.WriteString(w, "\n    ")
			io.WriteString(w, "</li>")
		}
//line dump.pushup:8
		io.WriteString(w, "\n")
//line dump.pushup:8
		io.WriteString(w, "</ul>")
//line dump.pushup:9
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
