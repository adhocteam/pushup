// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"sync"
)

type Pushup__about__3 struct {
	pushupFilePath string
}

func (t *Pushup__about__3) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__about__3) register() {
	routes.add("/about", t)
}

func init() {
	page := new(Pushup__about__3)
	page.pushupFilePath = "about.pushup"
	page.register()
}

func (t *Pushup__about__3) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__about__3) Respond(w http.ResponseWriter, req *http.Request) error {
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
//line about.pushup:2
		title := "About Pushup"
//line about.pushup:3
		var remoteAddr string
//line about.pushup:4
		if req.Header.Get("X-Real-Ip") != "" {
//line about.pushup:5
			remoteAddr = req.Header.Get("X-Real-Ip")
//line about.pushup:6
		} else if req.Header.Get("X-Forwarded-For") != "" {
//line about.pushup:7
			remoteAddr = req.Header.Get("X-Forwarded-For")
//line about.pushup:8
		} else {
//line about.pushup:9
			remoteAddr = req.RemoteAddr
//line about.pushup:10
		}
//line about.pushup:11

//line about.pushup:12
		io.WriteString(w, "\n\n")
//line about.pushup:13
		io.WriteString(w, "<h1>")
//line about.pushup:13
		printEscaped(w, title)
//line about.pushup:13
		io.WriteString(w, "</h1>")
//line about.pushup:14
		io.WriteString(w, "\n\n")
//line about.pushup:15
		io.WriteString(w, "<p>")
//line about.pushup:15
		io.WriteString(w, "Pushup is an old-school but modern web framework for Go.")
//line about.pushup:15
		io.WriteString(w, "</p>")
//line about.pushup:16
		io.WriteString(w, "\n\n")
//line about.pushup:17
		io.WriteString(w, "<p>")
//line about.pushup:17
		io.WriteString(w, "You came from: ")
//line about.pushup:17
		printEscaped(w, remoteAddr)
//line about.pushup:17
		io.WriteString(w, "</p>")
//line about.pushup:18
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
