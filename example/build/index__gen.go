// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type Pushup__index__21 struct {
	pushupFilePath string
}

func (t *Pushup__index__21) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__index__21) register() {
	routes.add("/", t)
}

func init() {
	page := new(Pushup__index__21)
	page.pushupFilePath = "index.pushup"
	page.register()
}

func (t *Pushup__index__21) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__index__21) Respond(w http.ResponseWriter, req *http.Request) error {
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
//line index.pushup:2
		io.WriteString(w, "\n\n")
//line index.pushup:4
		title := "Examples of Pushup"
//line index.pushup:5
		greeting := "Hello"
//line index.pushup:6
		name := "world"
//line index.pushup:7

//line index.pushup:8
		io.WriteString(w, "\n\n")
//line index.pushup:9
		io.WriteString(w, "<h2>")
//line index.pushup:9
		printEscaped(w, title)
//line index.pushup:9
		io.WriteString(w, "</h2>")
//line index.pushup:10
		io.WriteString(w, "\n\n")
//line index.pushup:11
		io.WriteString(w, "<h2 ")
//line index.pushup:11
		io.WriteString(w, "style")
//line index.pushup:11
		io.WriteString(w, "=\"")
//line index.pushup:11
		io.WriteString(w, "color: green")
//line index.pushup:11
		io.WriteString(w, "\">")
//line index.pushup:11
		printEscaped(w, greeting)
//line index.pushup:11
		io.WriteString(w, ", ")
//line index.pushup:11
		printEscaped(w, name)
//line index.pushup:11
		io.WriteString(w, "!")
//line index.pushup:11
		io.WriteString(w, "</h2>")
//line index.pushup:12
		io.WriteString(w, "\n\n")
//line index.pushup:13
		io.WriteString(w, "<h3 ")
//line index.pushup:13
		io.WriteString(w, "style")
//line index.pushup:13
		io.WriteString(w, "=\"")
//line index.pushup:13
		io.WriteString(w, "color: orange")
//line index.pushup:13
		io.WriteString(w, "\">")
//line index.pushup:13
		io.WriteString(w, "Pushup is a modern web framework for Go with an old-school PHP vibe.")
//line index.pushup:13
		io.WriteString(w, "</h3>")
//line index.pushup:14
		io.WriteString(w, "\n\n")
//line index.pushup:15
		io.WriteString(w, "<h4>")
//line index.pushup:15
		io.WriteString(w, "Features")
//line index.pushup:15
		io.WriteString(w, "</h4>")
//line index.pushup:16
		io.WriteString(w, "\n\n")
//line index.pushup:17
		io.WriteString(w, "<ul>")
//line index.pushup:18
		io.WriteString(w, "\n    ")
//line index.pushup:18
		io.WriteString(w, "<li>")
//line index.pushup:18
		io.WriteString(w, "Compiled")
//line index.pushup:18
		io.WriteString(w, "</li>")
//line index.pushup:19
		io.WriteString(w, "\n    ")
//line index.pushup:19
		io.WriteString(w, "<li>")
//line index.pushup:19
		io.WriteString(w, "Modern hypertext")
//line index.pushup:19
		io.WriteString(w, "</li>")
//line index.pushup:20
		io.WriteString(w, "\n    ")
//line index.pushup:20
		io.WriteString(w, "<li>")
//line index.pushup:20
		io.WriteString(w, "Filesystem-based routing")
//line index.pushup:20
		io.WriteString(w, "</li>")
//line index.pushup:21
		io.WriteString(w, "\n    ")
//line index.pushup:21
		io.WriteString(w, "<li>")
//line index.pushup:21
		io.WriteString(w, "Live-reloading dev mode")
//line index.pushup:21
		io.WriteString(w, "</li>")
//line index.pushup:22
		io.WriteString(w, "\n    ")
//line index.pushup:22
		io.WriteString(w, "<li>")
//line index.pushup:22
		io.WriteString(w, "Builds on Go ")
//line index.pushup:22
		io.WriteString(w, "<tt>")
//line index.pushup:22
		io.WriteString(w, "net/http")
//line index.pushup:22
		io.WriteString(w, "</tt>")
//line index.pushup:22
		io.WriteString(w, "</li>")
//line index.pushup:23
		io.WriteString(w, "\n")
//line index.pushup:23
		io.WriteString(w, "</ul>")
//line index.pushup:24
		io.WriteString(w, "\n\n")
//line index.pushup:25
		io.WriteString(w, "<p>")
//line index.pushup:25
		io.WriteString(w, "It is currently ")
//line index.pushup:25
		printEscaped(w, time.Now().Format(time.UnixDate))
//line index.pushup:25
		io.WriteString(w, ".")
//line index.pushup:25
		io.WriteString(w, "</p>")
//line index.pushup:26
		io.WriteString(w, "\n\n")
//line index.pushup:27
		io.WriteString(w, "<ul>")
//line index.pushup:28
		io.WriteString(w, "\n    ")
//line index.pushup:28
		io.WriteString(w, "<li>")
//line index.pushup:28
		io.WriteString(w, "<a ")
//line index.pushup:28
		io.WriteString(w, "href")
//line index.pushup:28
		io.WriteString(w, "=\"")
//line index.pushup:28
		io.WriteString(w, "/about")
//line index.pushup:28
		io.WriteString(w, "\">")
//line index.pushup:28
		io.WriteString(w, "About")
//line index.pushup:28
		io.WriteString(w, "</a>")
//line index.pushup:28
		io.WriteString(w, "</li>")
//line index.pushup:29
		io.WriteString(w, "\n    ")
//line index.pushup:29
		io.WriteString(w, "<li>")
//line index.pushup:29
		io.WriteString(w, "<a ")
//line index.pushup:29
		io.WriteString(w, "href")
//line index.pushup:29
		io.WriteString(w, "=\"")
//line index.pushup:29
		io.WriteString(w, "/escape")
//line index.pushup:29
		io.WriteString(w, "\">")
//line index.pushup:29
		io.WriteString(w, "Escaping")
//line index.pushup:29
		io.WriteString(w, "</a>")
//line index.pushup:29
		io.WriteString(w, "</li>")
//line index.pushup:30
		io.WriteString(w, "\n    ")
//line index.pushup:30
		io.WriteString(w, "<li>")
//line index.pushup:30
		io.WriteString(w, "<a ")
//line index.pushup:30
		io.WriteString(w, "href")
//line index.pushup:30
		io.WriteString(w, "=\"")
//line index.pushup:30
		io.WriteString(w, "/if")
//line index.pushup:30
		io.WriteString(w, "\">")
//line index.pushup:30
		io.WriteString(w, "If")
//line index.pushup:30
		io.WriteString(w, "</a>")
//line index.pushup:30
		io.WriteString(w, "</li>")
//line index.pushup:31
		io.WriteString(w, "\n    ")
//line index.pushup:31
		io.WriteString(w, "<li>")
//line index.pushup:31
		io.WriteString(w, "<a ")
//line index.pushup:31
		io.WriteString(w, "href")
//line index.pushup:31
		io.WriteString(w, "=\"")
//line index.pushup:31
		io.WriteString(w, "/for")
//line index.pushup:31
		io.WriteString(w, "\">")
//line index.pushup:31
		io.WriteString(w, "For")
//line index.pushup:31
		io.WriteString(w, "</a>")
//line index.pushup:31
		io.WriteString(w, "</li>")
//line index.pushup:32
		io.WriteString(w, "\n    ")
//line index.pushup:32
		io.WriteString(w, "<li>")
//line index.pushup:32
		io.WriteString(w, "<a ")
//line index.pushup:32
		io.WriteString(w, "href")
//line index.pushup:32
		io.WriteString(w, "=\"")
//line index.pushup:32
		io.WriteString(w, "/dump")
//line index.pushup:32
		io.WriteString(w, "\">")
//line index.pushup:32
		io.WriteString(w, "Dump")
//line index.pushup:32
		io.WriteString(w, "</a>")
//line index.pushup:32
		io.WriteString(w, "</li>")
//line index.pushup:33
		io.WriteString(w, "\n    ")
//line index.pushup:33
		io.WriteString(w, "<li>")
//line index.pushup:33
		io.WriteString(w, "<a ")
//line index.pushup:33
		io.WriteString(w, "href")
//line index.pushup:33
		io.WriteString(w, "=\"")
//line index.pushup:33
		io.WriteString(w, "/alt-layout")
//line index.pushup:33
		io.WriteString(w, "\">")
//line index.pushup:33
		io.WriteString(w, "Non-default layout")
//line index.pushup:33
		io.WriteString(w, "</a>")
//line index.pushup:33
		io.WriteString(w, "</li>")
//line index.pushup:34
		io.WriteString(w, "\n    ")
//line index.pushup:34
		io.WriteString(w, "<li>")
//line index.pushup:34
		io.WriteString(w, "<a ")
//line index.pushup:34
		io.WriteString(w, "href")
//line index.pushup:34
		io.WriteString(w, "=\"")
//line index.pushup:34
		io.WriteString(w, "/no-layout")
//line index.pushup:34
		io.WriteString(w, "\">")
//line index.pushup:34
		io.WriteString(w, "No layout")
//line index.pushup:34
		io.WriteString(w, "</a>")
//line index.pushup:34
		io.WriteString(w, "</li>")
//line index.pushup:35
		io.WriteString(w, "\n    ")
//line index.pushup:35
		io.WriteString(w, "<li>")
//line index.pushup:35
		io.WriteString(w, "<a ")
//line index.pushup:35
		io.WriteString(w, "href")
//line index.pushup:35
		io.WriteString(w, "=\"")
//line index.pushup:35
		io.WriteString(w, "/x/sub")
//line index.pushup:35
		io.WriteString(w, "\">")
//line index.pushup:35
		io.WriteString(w, "Subdirectory route (")
//line index.pushup:35
		io.WriteString(w, "<tt>")
//line index.pushup:35
		io.WriteString(w, "/x/sub")
//line index.pushup:35
		io.WriteString(w, "</tt>")
//line index.pushup:35
		io.WriteString(w, ")")
//line index.pushup:35
		io.WriteString(w, "</a>")
//line index.pushup:35
		io.WriteString(w, "</li>")
//line index.pushup:36
		io.WriteString(w, "\n    ")
//line index.pushup:36
		io.WriteString(w, "<li>")
//line index.pushup:36
		io.WriteString(w, "<a ")
//line index.pushup:36
		io.WriteString(w, "href")
//line index.pushup:36
		io.WriteString(w, "=\"")
//line index.pushup:36
		io.WriteString(w, "/dyn/world")
//line index.pushup:36
		io.WriteString(w, "\">")
//line index.pushup:36
		io.WriteString(w, "Dynamic path segments")
//line index.pushup:36
		io.WriteString(w, "</a>")
//line index.pushup:36
		io.WriteString(w, "</li>")
//line index.pushup:37
		io.WriteString(w, "\n    ")
//line index.pushup:37
		io.WriteString(w, "<li>")
//line index.pushup:37
		io.WriteString(w, "<a ")
//line index.pushup:37
		io.WriteString(w, "href")
//line index.pushup:37
		io.WriteString(w, "=\"")
//line index.pushup:37
		io.WriteString(w, "/htmx")
//line index.pushup:37
		io.WriteString(w, "\">")
//line index.pushup:37
		io.WriteString(w, "htmx examples")
//line index.pushup:37
		io.WriteString(w, "</a>")
//line index.pushup:37
		io.WriteString(w, "</li>")
//line index.pushup:38
		io.WriteString(w, "\n    ")
//line index.pushup:38
		io.WriteString(w, "<li>")
//line index.pushup:38
		io.WriteString(w, "<a ")
//line index.pushup:38
		io.WriteString(w, "href")
//line index.pushup:38
		io.WriteString(w, "=\"")
//line index.pushup:38
		io.WriteString(w, "/crud")
//line index.pushup:38
		io.WriteString(w, "\">")
//line index.pushup:38
		io.WriteString(w, "CRUD")
//line index.pushup:38
		io.WriteString(w, "</a>")
//line index.pushup:38
		io.WriteString(w, "</li>")
//line index.pushup:39
		io.WriteString(w, "\n")
//line index.pushup:39
		io.WriteString(w, "</ul>")
//line index.pushup:40
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
