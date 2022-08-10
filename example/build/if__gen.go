// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"sync"
)

type Pushup__if__20 struct {
	pushupFilePath string
}

func (t *Pushup__if__20) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__if__20) register() {
	routes.add("/if", t)
}

func init() {
	page := new(Pushup__if__20)
	page.pushupFilePath = "if.pushup"
	page.register()
}

func (t *Pushup__if__20) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__if__20) Respond(w http.ResponseWriter, req *http.Request) error {
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
//line if.pushup:2
		name := req.FormValue("name")
//line if.pushup:3
		bladeRunner := "Deckard"
//line if.pushup:4

//line if.pushup:5
		io.WriteString(w, "\n\n")
		if name == "" {
//line if.pushup:7
			io.WriteString(w, "\n    ")
//line if.pushup:7
//line if.pushup:7
			io.WriteString(w, "<div>")
//line if.pushup:8
			io.WriteString(w, "\n        ")
//line if.pushup:8
//line if.pushup:8
			io.WriteString(w, "<h1>")
//line if.pushup:8
			io.WriteString(w, "Hello, world!")
			io.WriteString(w, "</h1>")
//line if.pushup:9
			io.WriteString(w, "\n        ")
//line if.pushup:9
//line if.pushup:9
			io.WriteString(w, "<a ")
//line if.pushup:9
			io.WriteString(w, "href")
//line if.pushup:9
			io.WriteString(w, "=\"")
//line if.pushup:9
			io.WriteString(w, "/if?name=")
//line if.pushup:1
			printEscaped(w, bladeRunner)
//line if.pushup:9
			io.WriteString(w, "\">")
//line if.pushup:9
			io.WriteString(w, "add ")
//line if.pushup:9
//line if.pushup:9
			io.WriteString(w, "<tt>")
//line if.pushup:9
			io.WriteString(w, "name")
			io.WriteString(w, "</tt>")
//line if.pushup:9
			io.WriteString(w, " to URL query params")
			io.WriteString(w, "</a href=\"/if?name=^bladeRunner\">")
//line if.pushup:10
			io.WriteString(w, "\n    ")
			io.WriteString(w, "</div>")
		} else {
//line if.pushup:12
			io.WriteString(w, "\n    ")
//line if.pushup:12
//line if.pushup:12
			io.WriteString(w, "<div>")
//line if.pushup:13
			io.WriteString(w, "\n        ")
//line if.pushup:13
//line if.pushup:13
			io.WriteString(w, "<h1>")
//line if.pushup:13
			io.WriteString(w, "Hello, ")
//line if.pushup:13
			printEscaped(w, name)
//line if.pushup:13
			io.WriteString(w, "!")
			io.WriteString(w, "</h1>")
//line if.pushup:14
			io.WriteString(w, "\n        ")
//line if.pushup:14
//line if.pushup:14
			io.WriteString(w, "<a ")
//line if.pushup:14
			io.WriteString(w, "href")
//line if.pushup:14
			io.WriteString(w, "=\"")
//line if.pushup:14
			io.WriteString(w, "/if")
//line if.pushup:14
			io.WriteString(w, "\">")
//line if.pushup:14
			io.WriteString(w, "remove ")
//line if.pushup:14
//line if.pushup:14
			io.WriteString(w, "<tt>")
//line if.pushup:14
			io.WriteString(w, "name")
			io.WriteString(w, "</tt>")
//line if.pushup:14
			io.WriteString(w, " from URL query params")
			io.WriteString(w, "</a href=\"/if\">")
//line if.pushup:15
			io.WriteString(w, "\n    ")
			io.WriteString(w, "</div>")
		}
//line if.pushup:17
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
