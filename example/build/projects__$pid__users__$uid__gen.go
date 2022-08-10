// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"sync"
)

type Pushup__DollarSign_uid__23 struct {
	pushupFilePath string
}

func (t *Pushup__DollarSign_uid__23) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__DollarSign_uid__23) register() {
	routes.add("/projects/:pid/users/:uid", t)
}

func init() {
	page := new(Pushup__DollarSign_uid__23)
	page.pushupFilePath = "projects/$pid/users/$uid.pushup"
	page.register()
}

func (t *Pushup__DollarSign_uid__23) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__DollarSign_uid__23) Respond(w http.ResponseWriter, req *http.Request) error {
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
//line $uid.pushup:1
		io.WriteString(w, "<p>")
//line $uid.pushup:1
		io.WriteString(w, "Project ")
//line $uid.pushup:1
		printEscaped(w, getParam(req, "pid"))
//line $uid.pushup:1
		io.WriteString(w, "</p>")
//line $uid.pushup:2
		io.WriteString(w, "\n")
//line $uid.pushup:2
		io.WriteString(w, "<p>")
//line $uid.pushup:2
		io.WriteString(w, "User ")
//line $uid.pushup:2
		printEscaped(w, getParam(req, "uid"))
//line $uid.pushup:2
		io.WriteString(w, "</p>")
//line $uid.pushup:3
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
