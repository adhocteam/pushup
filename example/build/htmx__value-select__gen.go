// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"sync"
)

type Pushup__value_select__19 struct {
	pushupFilePath string
}

func (t *Pushup__value_select__19) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__value_select__19) register() {
	routes.add("/htmx/value-select", t)
}

func init() {
	page := new(Pushup__value_select__19)
	page.pushupFilePath = "htmx/value-select.pushup"
	page.register()
}

func (t *Pushup__value_select__19) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__value_select__19) Respond(w http.ResponseWriter, req *http.Request) error {
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
//line value-select.pushup:1
		io.WriteString(w, "<h1>")
//line value-select.pushup:1
		io.WriteString(w, "htmx example: Value select")
//line value-select.pushup:1
		io.WriteString(w, "</h1>")
//line value-select.pushup:2
		io.WriteString(w, "\n\n")
//line value-select.pushup:3
		io.WriteString(w, "<h2>")
//line value-select.pushup:3
		io.WriteString(w, "Pick a make and model")
//line value-select.pushup:3
		io.WriteString(w, "</h2>")
//line value-select.pushup:4
		io.WriteString(w, "\n\n")
//line value-select.pushup:5
		io.WriteString(w, "<div>")
//line value-select.pushup:6
		io.WriteString(w, "\n    ")
//line value-select.pushup:6
		io.WriteString(w, "<label>")
//line value-select.pushup:6
		io.WriteString(w, "Make")
//line value-select.pushup:6
		io.WriteString(w, "</label>")
//line value-select.pushup:7
		io.WriteString(w, "\n    ")
//line value-select.pushup:7
		io.WriteString(w, "<select ")
//line value-select.pushup:7
		io.WriteString(w, "name")
//line value-select.pushup:7
		io.WriteString(w, "=\"")
//line value-select.pushup:7
		io.WriteString(w, "make")
//line value-select.pushup:7
		io.WriteString(w, "\" ")
//line value-select.pushup:7
		io.WriteString(w, "hx-get")
//line value-select.pushup:7
		io.WriteString(w, "=\"")
//line value-select.pushup:7
		io.WriteString(w, "./models")
//line value-select.pushup:7
		io.WriteString(w, "\" ")
//line value-select.pushup:7
		io.WriteString(w, "hx-target")
//line value-select.pushup:7
		io.WriteString(w, "=\"")
//line value-select.pushup:7
		io.WriteString(w, "#models")
//line value-select.pushup:7
		io.WriteString(w, "\">")
//line value-select.pushup:8
		io.WriteString(w, "\n        ")
//line value-select.pushup:8
		io.WriteString(w, "<option ")
//line value-select.pushup:8
		io.WriteString(w, "value")
//line value-select.pushup:8
		io.WriteString(w, "=\"")
//line value-select.pushup:8
		io.WriteString(w, "Apple silicon")
//line value-select.pushup:8
		io.WriteString(w, "\">")
//line value-select.pushup:8
		io.WriteString(w, "Apple silicon")
//line value-select.pushup:8
		io.WriteString(w, "</option>")
//line value-select.pushup:9
		io.WriteString(w, "\n        ")
//line value-select.pushup:9
		io.WriteString(w, "<option ")
//line value-select.pushup:9
		io.WriteString(w, "value")
//line value-select.pushup:9
		io.WriteString(w, "=\"")
//line value-select.pushup:9
		io.WriteString(w, "Intel")
//line value-select.pushup:9
		io.WriteString(w, "\">")
//line value-select.pushup:9
		io.WriteString(w, "Intel")
//line value-select.pushup:9
		io.WriteString(w, "</option>")
//line value-select.pushup:10
		io.WriteString(w, "\n        ")
//line value-select.pushup:10
		io.WriteString(w, "<option ")
//line value-select.pushup:10
		io.WriteString(w, "value")
//line value-select.pushup:10
		io.WriteString(w, "=\"")
//line value-select.pushup:10
		io.WriteString(w, "AMD")
//line value-select.pushup:10
		io.WriteString(w, "\">")
//line value-select.pushup:10
		io.WriteString(w, "AMD")
//line value-select.pushup:10
		io.WriteString(w, "</option>")
//line value-select.pushup:11
		io.WriteString(w, "\n    ")
//line value-select.pushup:11
		io.WriteString(w, "</select>")
//line value-select.pushup:12
		io.WriteString(w, "\n")
//line value-select.pushup:12
		io.WriteString(w, "</div>")
//line value-select.pushup:13
		io.WriteString(w, "\n\n")
//line value-select.pushup:14
		io.WriteString(w, "<div>")
//line value-select.pushup:15
		io.WriteString(w, "\n    ")
//line value-select.pushup:15
		io.WriteString(w, "<label>")
//line value-select.pushup:15
		io.WriteString(w, "Model")
//line value-select.pushup:15
		io.WriteString(w, "</label>")
//line value-select.pushup:16
		io.WriteString(w, "\n    ")
//line value-select.pushup:16
		io.WriteString(w, "<select ")
//line value-select.pushup:16
		io.WriteString(w, "id")
//line value-select.pushup:16
		io.WriteString(w, "=\"")
//line value-select.pushup:16
		io.WriteString(w, "models")
//line value-select.pushup:16
		io.WriteString(w, "\" ")
//line value-select.pushup:16
		io.WriteString(w, "name")
//line value-select.pushup:16
		io.WriteString(w, "=\"")
//line value-select.pushup:16
		io.WriteString(w, "model")
//line value-select.pushup:16
		io.WriteString(w, "\">")
//line value-select.pushup:17
		io.WriteString(w, "\n        ")
//line value-select.pushup:17
		io.WriteString(w, "<option ")
//line value-select.pushup:17
		io.WriteString(w, "value")
//line value-select.pushup:17
		io.WriteString(w, "=\"")
//line value-select.pushup:17
		io.WriteString(w, "M1")
//line value-select.pushup:17
		io.WriteString(w, "\">")
//line value-select.pushup:17
		io.WriteString(w, "M1")
//line value-select.pushup:17
		io.WriteString(w, "</option>")
//line value-select.pushup:18
		io.WriteString(w, "\n        ")
//line value-select.pushup:18
		io.WriteString(w, "<option ")
//line value-select.pushup:18
		io.WriteString(w, "value")
//line value-select.pushup:18
		io.WriteString(w, "=\"")
//line value-select.pushup:18
		io.WriteString(w, "M2")
//line value-select.pushup:18
		io.WriteString(w, "\">")
//line value-select.pushup:18
		io.WriteString(w, "M2")
//line value-select.pushup:18
		io.WriteString(w, "</option>")
//line value-select.pushup:19
		io.WriteString(w, "\n    ")
//line value-select.pushup:19
		io.WriteString(w, "</select>")
//line value-select.pushup:20
		io.WriteString(w, "\n")
//line value-select.pushup:20
		io.WriteString(w, "</div>")
//line value-select.pushup:21
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
