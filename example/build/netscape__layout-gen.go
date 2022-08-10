// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"net/http"
)

type Pushup__netscape_layout__2 struct {
	pushupFilePath string
}

func (t *Pushup__netscape_layout__2) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func init() {
	layout := new(Pushup__netscape_layout__2)
	layout.pushupFilePath = "netscape.pushup"
	layouts["netscape"] = layout
}

func (t *Pushup__netscape_layout__2) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__netscape_layout__2) Respond(yield chan struct{}, w http.ResponseWriter, req *http.Request) error {
	// Begin user Go code and HTML
	{
//line netscape.pushup:1
		io.WriteString(w, "<!DOCTYPE html>")
//line netscape.pushup:2
		io.WriteString(w, "\n")
//line netscape.pushup:2
		io.WriteString(w, "<html ")
//line netscape.pushup:2
		io.WriteString(w, "lang")
//line netscape.pushup:2
		io.WriteString(w, "=\"")
//line netscape.pushup:2
		io.WriteString(w, "en")
//line netscape.pushup:2
		io.WriteString(w, "\">")
//line netscape.pushup:3
		io.WriteString(w, "\n    ")
//line netscape.pushup:3
		io.WriteString(w, "<head>")
//line netscape.pushup:4
		io.WriteString(w, "\n        ")
//line netscape.pushup:4
		io.WriteString(w, "<title>")
//line netscape.pushup:4
		io.WriteString(w, "Pushup Navigator")
//line netscape.pushup:4
		io.WriteString(w, "</title>")
//line netscape.pushup:5
		io.WriteString(w, "\n        ")
//line netscape.pushup:5
		io.WriteString(w, "<style>")
//line netscape.pushup:6
		io.WriteString(w, "\n            html {\n                background: #c6c6c6;\n            }\n            body {\n                font-family: Times New Roman, serif;\n            }\n            hr {\n                border-size: 10px;\n            }\n        ")
//line netscape.pushup:15
		io.WriteString(w, "</style>")
//line netscape.pushup:16
		io.WriteString(w, "\n        ")
//line netscape.pushup:16
		io.WriteString(w, "<script ")
//line netscape.pushup:16
		io.WriteString(w, "src")
//line netscape.pushup:16
		io.WriteString(w, "=\"")
//line netscape.pushup:16
		io.WriteString(w, "https://unpkg.com/htmx.org@1.7.0")
//line netscape.pushup:16
		io.WriteString(w, "\">")
//line netscape.pushup:16
		io.WriteString(w, "</script>")
//line netscape.pushup:17
		io.WriteString(w, "\n    ")
//line netscape.pushup:17
		io.WriteString(w, "</head>")
//line netscape.pushup:18
		io.WriteString(w, "\n    ")
//line netscape.pushup:18
		io.WriteString(w, "<body>")
//line netscape.pushup:19
		io.WriteString(w, "\n        ")
//line netscape.pushup:19
		io.WriteString(w, "<header>")
//line netscape.pushup:20
		io.WriteString(w, "\n            ")
//line netscape.pushup:20
		io.WriteString(w, "<h1>")
//line netscape.pushup:20
		io.WriteString(w, "Pushup Navigator")
//line netscape.pushup:20
		io.WriteString(w, "</h1>")
//line netscape.pushup:21
		io.WriteString(w, "\n        ")
//line netscape.pushup:21
		io.WriteString(w, "</header>")
//line netscape.pushup:22
		io.WriteString(w, "\n        ")
//line netscape.pushup:22
		io.WriteString(w, "<main ")
//line netscape.pushup:22
		io.WriteString(w, "hx-boost")
//line netscape.pushup:22
		io.WriteString(w, "=\"")
//line netscape.pushup:22
		io.WriteString(w, "true")
//line netscape.pushup:22
		io.WriteString(w, "\">")
//line netscape.pushup:23
		io.WriteString(w, "\n            ")
		if fl, ok := w.(http.Flusher); ok {
			fl.Flush()
		}
		yield <- struct{}{}
		<-yield
//line netscape.pushup:24
		io.WriteString(w, "\n        ")
//line netscape.pushup:24
		io.WriteString(w, "</main>")
//line netscape.pushup:25
		io.WriteString(w, "\n        ")
//line netscape.pushup:25
		io.WriteString(w, "<hr>")
//line netscape.pushup:26
		io.WriteString(w, "\n        ")
//line netscape.pushup:26
		io.WriteString(w, "<footer>")
//line netscape.pushup:27
		io.WriteString(w, "\n            ")
//line netscape.pushup:27
		io.WriteString(w, "<p>")
//line netscape.pushup:27
		io.WriteString(w, "&copy;2022 Ad Hoc")
//line netscape.pushup:27
		io.WriteString(w, "</p>")
//line netscape.pushup:28
		io.WriteString(w, "\n        ")
//line netscape.pushup:28
		io.WriteString(w, "</footer>")
//line netscape.pushup:29
		io.WriteString(w, "\n    ")
//line netscape.pushup:29
		io.WriteString(w, "</body>")
//line netscape.pushup:30
		io.WriteString(w, "\n")
//line netscape.pushup:30
		io.WriteString(w, "</html>")
//line netscape.pushup:31
		io.WriteString(w, "\n")
		// End user Go code and HTML
	}
	return nil
}
