// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"net/http"
)

type Pushup__default_layout__1 struct {
	pushupFilePath string
}

func (t *Pushup__default_layout__1) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func init() {
	layout := new(Pushup__default_layout__1)
	layout.pushupFilePath = "default.pushup"
	layouts["default"] = layout
}

func (t *Pushup__default_layout__1) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__default_layout__1) Respond(yield chan struct{}, w http.ResponseWriter, req *http.Request) error {
	// Begin user Go code and HTML
	{
//line default.pushup:1
		io.WriteString(w, "<!DOCTYPE html>")
//line default.pushup:2
		io.WriteString(w, "\n")
//line default.pushup:2
		io.WriteString(w, "<html ")
//line default.pushup:2
		io.WriteString(w, "lang")
//line default.pushup:2
		io.WriteString(w, "=\"")
//line default.pushup:2
		io.WriteString(w, "en")
//line default.pushup:2
		io.WriteString(w, "\">")
//line default.pushup:3
		io.WriteString(w, "\n    ")
//line default.pushup:3
		io.WriteString(w, "<head>")
//line default.pushup:4
		io.WriteString(w, "\n        ")
//line default.pushup:4
		io.WriteString(w, "<title>")
//line default.pushup:4
		io.WriteString(w, "Pushup app")
//line default.pushup:4
		io.WriteString(w, "</title>")
//line default.pushup:5
		io.WriteString(w, "\n        ")
//line default.pushup:5
		io.WriteString(w, "<style>")
//line default.pushup:6
		io.WriteString(w, "\n            html {\n                background: #eee;\n            }\n            body {\n                font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", Roboto, \"Helvetica Neue\", Arial, \"Noto Sans\", sans-serif, \"Apple Color Emoji\", \"Segoe UI Emoji\", \"Segoe UI Symbol\", \"Noto Color Emoji\";\n                background: #fff;\n                margin: 0 auto;\n                max-width: 640px;\n                padding: 40px;\n            }\n            header > h1 {\n                font-size: 32px;\n            }\n            main > h1 {\n                font-size: 24px;\n            }\n            footer {\n                margin-top: 1rem;\n                border-top: 1px solid #ccc;\n                padding-top: 1rem;\n                font-size: 12px;\n            }\n        ")
//line default.pushup:28
		io.WriteString(w, "</style>")
//line default.pushup:29
		io.WriteString(w, "\n        ")
//line default.pushup:29
		io.WriteString(w, "<script ")
//line default.pushup:29
		io.WriteString(w, "src")
//line default.pushup:29
		io.WriteString(w, "=\"")
//line default.pushup:29
		io.WriteString(w, "https://unpkg.com/htmx.org@1.7.0")
//line default.pushup:29
		io.WriteString(w, "\">")
//line default.pushup:29
		io.WriteString(w, "</script>")
//line default.pushup:30
		io.WriteString(w, "\n    ")
//line default.pushup:30
		io.WriteString(w, "</head>")
//line default.pushup:31
		io.WriteString(w, "\n    ")
//line default.pushup:31
		io.WriteString(w, "<body>")
//line default.pushup:32
		io.WriteString(w, "\n        ")
//line default.pushup:32
		io.WriteString(w, "<nav>")
//line default.pushup:33
		io.WriteString(w, "\n            ")
//line default.pushup:33
		io.WriteString(w, "<ul>")
//line default.pushup:34
		io.WriteString(w, "\n                ")
//line default.pushup:34
		io.WriteString(w, "<li>")
//line default.pushup:35
		io.WriteString(w, "\n                    ")
//line default.pushup:35
		io.WriteString(w, "<a ")
//line default.pushup:35
		io.WriteString(w, "href")
//line default.pushup:35
		io.WriteString(w, "=\"")
//line default.pushup:35
		io.WriteString(w, "/")
//line default.pushup:35
		io.WriteString(w, "\">")
//line default.pushup:35
		io.WriteString(w, "Home")
//line default.pushup:35
		io.WriteString(w, "</a>")
//line default.pushup:36
		io.WriteString(w, "\n                ")
//line default.pushup:36
		io.WriteString(w, "</li>")
//line default.pushup:37
		io.WriteString(w, "\n            ")
//line default.pushup:37
		io.WriteString(w, "</ul>")
//line default.pushup:38
		io.WriteString(w, "\n        ")
//line default.pushup:38
		io.WriteString(w, "</nav>")
//line default.pushup:39
		io.WriteString(w, "\n        ")
//line default.pushup:39
		io.WriteString(w, "<header>")
//line default.pushup:40
		io.WriteString(w, "\n            ")
//line default.pushup:40
		io.WriteString(w, "<h1>")
//line default.pushup:40
		io.WriteString(w, "Pushup demo")
//line default.pushup:40
		io.WriteString(w, "</h1>")
//line default.pushup:41
		io.WriteString(w, "\n        ")
//line default.pushup:41
		io.WriteString(w, "</header>")
//line default.pushup:42
		io.WriteString(w, "\n        ")
//line default.pushup:42
		io.WriteString(w, "<main>")
//line default.pushup:43
		io.WriteString(w, "\n            ")
		if fl, ok := w.(http.Flusher); ok {
			fl.Flush()
		}
		yield <- struct{}{}
		<-yield
//line default.pushup:44
		io.WriteString(w, "\n        ")
//line default.pushup:44
		io.WriteString(w, "</main>")
//line default.pushup:45
		io.WriteString(w, "\n        ")
//line default.pushup:45
		io.WriteString(w, "<footer>")
//line default.pushup:46
		io.WriteString(w, "\n            ")
//line default.pushup:46
		io.WriteString(w, "<p>")
//line default.pushup:46
		io.WriteString(w, "<a ")
//line default.pushup:46
		io.WriteString(w, "href")
//line default.pushup:46
		io.WriteString(w, "=\"")
//line default.pushup:46
		io.WriteString(w, "/source?route=")
//line default.pushup:1
		printEscaped(w, req.URL.Path)
//line default.pushup:46
		io.WriteString(w, "\">")
//line default.pushup:46
		io.WriteString(w, "view source")
//line default.pushup:46
		io.WriteString(w, "</a>")
//line default.pushup:46
		io.WriteString(w, "</p>")
//line default.pushup:47
		io.WriteString(w, "\n            ")
//line default.pushup:47
		io.WriteString(w, "<p>")
//line default.pushup:47
		io.WriteString(w, "&copy;2022 Ad Hoc")
//line default.pushup:47
		io.WriteString(w, "</p>")
//line default.pushup:48
		io.WriteString(w, "\n        ")
//line default.pushup:48
		io.WriteString(w, "</footer>")
//line default.pushup:49
		io.WriteString(w, "\n    ")
//line default.pushup:49
		io.WriteString(w, "</body>")
//line default.pushup:50
		io.WriteString(w, "\n")
//line default.pushup:50
		io.WriteString(w, "</html>")
//line default.pushup:51
		io.WriteString(w, "\n")
		// End user Go code and HTML
	}
	return nil
}
