// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"net/http"
)

type Pushup__no_layout__22 struct {
	pushupFilePath string
}

func (t *Pushup__no_layout__22) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__no_layout__22) register() {
	routes.add("/no-layout", t)
}

func init() {
	page := new(Pushup__no_layout__22)
	page.pushupFilePath = "no-layout.pushup"
	page.register()
}

func (t *Pushup__no_layout__22) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__no_layout__22) Respond(w http.ResponseWriter, req *http.Request) error {
	// Begin user Go code and HTML
	{
//line no-layout.pushup:2
		io.WriteString(w, "\n\n")
//line no-layout.pushup:3
		io.WriteString(w, "<!DOCTYPE html>")
//line no-layout.pushup:4
		io.WriteString(w, "\n")
//line no-layout.pushup:4
		io.WriteString(w, "<html>")
//line no-layout.pushup:5
		io.WriteString(w, "\n    ")
//line no-layout.pushup:5
		io.WriteString(w, "<head>")
//line no-layout.pushup:6
		io.WriteString(w, "\n        ")
//line no-layout.pushup:6
		io.WriteString(w, "<title>")
//line no-layout.pushup:6
		io.WriteString(w, "No layout!")
//line no-layout.pushup:6
		io.WriteString(w, "</title>")
//line no-layout.pushup:7
		io.WriteString(w, "\n    ")
//line no-layout.pushup:7
		io.WriteString(w, "</head>")
//line no-layout.pushup:8
		io.WriteString(w, "\n    ")
//line no-layout.pushup:8
		io.WriteString(w, "<body>")
//line no-layout.pushup:9
		io.WriteString(w, "\n        ")
//line no-layout.pushup:9
		io.WriteString(w, "<h1>")
//line no-layout.pushup:9
		io.WriteString(w, "No layout!")
//line no-layout.pushup:9
		io.WriteString(w, "</h1>")
//line no-layout.pushup:10
		io.WriteString(w, "\n        ")
//line no-layout.pushup:10
		io.WriteString(w, "<p>")
//line no-layout.pushup:10
		io.WriteString(w, "This component has no layout.")
//line no-layout.pushup:10
		io.WriteString(w, "</p>")
//line no-layout.pushup:11
		io.WriteString(w, "\n        ")
//line no-layout.pushup:11
		io.WriteString(w, "<p>")
//line no-layout.pushup:11
		io.WriteString(w, "<a ")
//line no-layout.pushup:11
		io.WriteString(w, "href")
//line no-layout.pushup:11
		io.WriteString(w, "=\"")
//line no-layout.pushup:11
		io.WriteString(w, "/")
//line no-layout.pushup:11
		io.WriteString(w, "\" ")
//line no-layout.pushup:11
		io.WriteString(w, "hx-boost")
//line no-layout.pushup:11
		io.WriteString(w, "=\"")
//line no-layout.pushup:11
		io.WriteString(w, "false")
//line no-layout.pushup:11
		io.WriteString(w, "\">")
//line no-layout.pushup:11
		io.WriteString(w, "Back home")
//line no-layout.pushup:11
		io.WriteString(w, "</a>")
//line no-layout.pushup:11
		io.WriteString(w, "</p>")
//line no-layout.pushup:12
		io.WriteString(w, "\n    ")
//line no-layout.pushup:12
		io.WriteString(w, "</body>")
//line no-layout.pushup:13
		io.WriteString(w, "\n")
//line no-layout.pushup:13
		io.WriteString(w, "</html>")
//line no-layout.pushup:14
		io.WriteString(w, "\n")
		// End user Go code and HTML
	}
	return nil
}
