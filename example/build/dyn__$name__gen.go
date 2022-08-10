// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"sync"
)

type Pushup__DollarSign_name__11 struct {
	pushupFilePath string
}

func (t *Pushup__DollarSign_name__11) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__DollarSign_name__11) register() {
	routes.add("/dyn/:name", t)
}

func init() {
	page := new(Pushup__DollarSign_name__11)
	page.pushupFilePath = "dyn/$name.pushup"
	page.register()
}

func (t *Pushup__DollarSign_name__11) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__DollarSign_name__11) Respond(w http.ResponseWriter, req *http.Request) error {
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
//line $name.pushup:1
		io.WriteString(w, "<h2>")
//line $name.pushup:1
		io.WriteString(w, "Dynamic path segments")
//line $name.pushup:1
		io.WriteString(w, "</h2>")
//line $name.pushup:2
		io.WriteString(w, "\n\n")
//line $name.pushup:3
		io.WriteString(w, "<p>")
//line $name.pushup:3
		io.WriteString(w, "This page lives at ")
//line $name.pushup:3
		io.WriteString(w, "<tt>")
//line $name.pushup:3
		io.WriteString(w, "dyn/$name.pushup")
//line $name.pushup:3
		io.WriteString(w, "</tt>")
//line $name.pushup:3
		io.WriteString(w, " in the project directory.")
//line $name.pushup:3
		io.WriteString(w, "</p>")
//line $name.pushup:4
		io.WriteString(w, "\n\n")
//line $name.pushup:5
		io.WriteString(w, "<p>")
//line $name.pushup:5
		io.WriteString(w, "The ")
//line $name.pushup:5
		io.WriteString(w, "<tt>")
//line $name.pushup:5
		io.WriteString(w, "$name")
//line $name.pushup:5
		io.WriteString(w, "</tt>")
//line $name.pushup:5
		io.WriteString(w, " part matches that part of the path, and is made available\n    to Pushup pages with ")
//line $name.pushup:6
		io.WriteString(w, "<tt>")
//line $name.pushup:6
		io.WriteString(w, "getParam(req, \"name\")")
//line $name.pushup:6
		io.WriteString(w, "</tt>")
//line $name.pushup:6
		io.WriteString(w, ".\n")
//line $name.pushup:7
		io.WriteString(w, "</p>")
//line $name.pushup:8
		io.WriteString(w, "\n\n")
//line $name.pushup:9
		io.WriteString(w, "<p>")
//line $name.pushup:9
		io.WriteString(w, "<tt>")
//line $name.pushup:9
		io.WriteString(w, "$name")
//line $name.pushup:9
		io.WriteString(w, "</tt>")
//line $name.pushup:9
		io.WriteString(w, " can be pretty much anything, treat it like a URL slug.\n    Whatever the characters after the ")
//line $name.pushup:10
		io.WriteString(w, "<tt>")
//line $name.pushup:10
		io.WriteString(w, "$")
//line $name.pushup:10
		io.WriteString(w, "</tt>")
//line $name.pushup:10
		io.WriteString(w, " is the name to pass to\n    ")
//line $name.pushup:11
		io.WriteString(w, "<tt>")
//line $name.pushup:11
		io.WriteString(w, "getParam()")
//line $name.pushup:11
		io.WriteString(w, "</tt>")
//line $name.pushup:11
		io.WriteString(w, ".\n")
//line $name.pushup:12
		io.WriteString(w, "</p>")
//line $name.pushup:13
		io.WriteString(w, "\n\n")
//line $name.pushup:14
		io.WriteString(w, "<p>")
//line $name.pushup:14
		io.WriteString(w, "This also works with directories named starting with a '$'. For example,\n    ")
//line $name.pushup:15
		io.WriteString(w, "<tt>")
//line $name.pushup:15
		io.WriteString(w, "users/$userID/projects/$projectID.pushup")
//line $name.pushup:15
		io.WriteString(w, "</tt>")
//line $name.pushup:16
		io.WriteString(w, "\n")
//line $name.pushup:16
		io.WriteString(w, "</p>")
//line $name.pushup:17
		io.WriteString(w, "\n\n")
//line $name.pushup:18
		io.WriteString(w, "<hr/>")
//line $name.pushup:19
		io.WriteString(w, "\n\n")
//line $name.pushup:20
		io.WriteString(w, "<dl>")
//line $name.pushup:21
		io.WriteString(w, "\n    ")
//line $name.pushup:21
		io.WriteString(w, "<dd>")
//line $name.pushup:21
		io.WriteString(w, "Filesystem:")
//line $name.pushup:21
		io.WriteString(w, "</dd>")
//line $name.pushup:21
		io.WriteString(w, " ")
//line $name.pushup:21
		io.WriteString(w, "<dt>")
//line $name.pushup:21
		io.WriteString(w, "<tt>")
//line $name.pushup:21
		io.WriteString(w, "&lt;project root&gt;/pages/dyn/$name.pushup")
//line $name.pushup:21
		io.WriteString(w, "</tt>")
//line $name.pushup:21
		io.WriteString(w, "</dt>")
//line $name.pushup:22
		io.WriteString(w, "\n    ")
//line $name.pushup:22
		io.WriteString(w, "<dd>")
//line $name.pushup:22
		io.WriteString(w, "Generated route:")
//line $name.pushup:22
		io.WriteString(w, "</dd>")
//line $name.pushup:22
		io.WriteString(w, " ")
//line $name.pushup:22
		io.WriteString(w, "<dt>")
//line $name.pushup:22
		io.WriteString(w, "<tt>")
//line $name.pushup:22
		io.WriteString(w, "/dyn/([")
//line $name.pushup:22
		io.WriteString(w, "^")
//line $name.pushup:22
		io.WriteString(w, "/]+)")
//line $name.pushup:22
		io.WriteString(w, "</tt>")
//line $name.pushup:22
		io.WriteString(w, "</dt>")
//line $name.pushup:23
		io.WriteString(w, "\n")
//line $name.pushup:23
		io.WriteString(w, "</dl>")
//line $name.pushup:24
		io.WriteString(w, "\n\n")
//line $name.pushup:25
		io.WriteString(w, "<hr/>")
//line $name.pushup:26
		io.WriteString(w, "\n\n")
//line $name.pushup:27
		io.WriteString(w, "<h3>")
//line $name.pushup:27
		io.WriteString(w, "Live example")
//line $name.pushup:27
		io.WriteString(w, "</h3>")
//line $name.pushup:28
		io.WriteString(w, "\n\n")
//line $name.pushup:29
		io.WriteString(w, "<p>")
//line $name.pushup:29
		io.WriteString(w, "<code>")
//line $name.pushup:29
		io.WriteString(w, "Hello, ")
//line $name.pushup:29
		io.WriteString(w, "^")
//line $name.pushup:29
		io.WriteString(w, "(getParam(req, \"name\"))")
//line $name.pushup:29
		io.WriteString(w, "</code>")
//line $name.pushup:29
		io.WriteString(w, ": ")
//line $name.pushup:29
		io.WriteString(w, "<b>")
//line $name.pushup:29
		io.WriteString(w, "Hello, ")
//line $name.pushup:29
		printEscaped(w, getParam(req, "name"))
//line $name.pushup:29
		io.WriteString(w, "!")
//line $name.pushup:29
		io.WriteString(w, "</b>")
//line $name.pushup:29
		io.WriteString(w, "</p>")
//line $name.pushup:30
		io.WriteString(w, "\n")
//line $name.pushup:30
		io.WriteString(w, "<p>")
//line $name.pushup:30
		io.WriteString(w, "URL: ")
//line $name.pushup:30
		io.WriteString(w, "<tt>")
//line $name.pushup:30
		printEscaped(w, req.URL.Path)
//line $name.pushup:30
		io.WriteString(w, "</tt>")
//line $name.pushup:30
		io.WriteString(w, "</p>")
//line $name.pushup:31
		io.WriteString(w, "\n")
		if getParam(req, "name") == "world" {
//line $name.pushup:32
			io.WriteString(w, "\n    ")
//line $name.pushup:32
//line $name.pushup:32
			io.WriteString(w, "<p>")
//line $name.pushup:32
//line $name.pushup:32
			io.WriteString(w, "<a ")
//line $name.pushup:32
			io.WriteString(w, "href")
//line $name.pushup:32
			io.WriteString(w, "=\"")
//line $name.pushup:32
			io.WriteString(w, "/dyn/Pushup")
//line $name.pushup:32
			io.WriteString(w, "\">")
//line $name.pushup:32
			io.WriteString(w, "/dyn/Pushup")
			io.WriteString(w, "</a href=\"/dyn/Pushup\">")
			io.WriteString(w, "</p>")
		} else {
//line $name.pushup:34
			io.WriteString(w, "\n    ")
//line $name.pushup:34
//line $name.pushup:34
			io.WriteString(w, "<p>")
//line $name.pushup:34
//line $name.pushup:34
			io.WriteString(w, "<a ")
//line $name.pushup:34
			io.WriteString(w, "href")
//line $name.pushup:34
			io.WriteString(w, "=\"")
//line $name.pushup:34
			io.WriteString(w, "/dyn/world")
//line $name.pushup:34
			io.WriteString(w, "\">")
//line $name.pushup:34
			io.WriteString(w, "/dyn/world")
			io.WriteString(w, "</a href=\"/dyn/world\">")
			io.WriteString(w, "</p>")
		}
//line $name.pushup:36
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
