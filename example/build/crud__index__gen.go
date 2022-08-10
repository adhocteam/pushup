// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"sync"
)

type Pushup__index__9 struct {
	pushupFilePath string
}

func (t *Pushup__index__9) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__index__9) register() {
	routes.add("/crud", t)
}

func init() {
	page := new(Pushup__index__9)
	page.pushupFilePath = "crud/index.pushup"
	page.register()
}

func (t *Pushup__index__9) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__index__9) Respond(w http.ResponseWriter, req *http.Request) error {
	renderLayout := true
//line index.pushup:2
	albums, err := getAlbums(DB, 0, 0)
//line index.pushup:3
	if err != nil {
//line index.pushup:4
		return err
//line index.pushup:5
	}
//line index.pushup:6

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
//line index.pushup:7
		io.WriteString(w, "\n\n")
//line index.pushup:8
		io.WriteString(w, "<h1>")
//line index.pushup:8
		io.WriteString(w, "CRUD example")
//line index.pushup:8
		io.WriteString(w, "</h1>")
//line index.pushup:9
		io.WriteString(w, "\n\n")
//line index.pushup:10
		io.WriteString(w, "<h2>")
//line index.pushup:10
		io.WriteString(w, "Album collection")
//line index.pushup:10
		io.WriteString(w, "</h2>")
//line index.pushup:11
		io.WriteString(w, "\n\n")
//line index.pushup:12
		io.WriteString(w, "<p>")
//line index.pushup:12
		io.WriteString(w, "<a ")
//line index.pushup:12
		io.WriteString(w, "href")
//line index.pushup:12
		io.WriteString(w, "=\"")
//line index.pushup:12
		io.WriteString(w, "/crud/album/new")
//line index.pushup:12
		io.WriteString(w, "\">")
//line index.pushup:12
		io.WriteString(w, "Add album")
//line index.pushup:12
		io.WriteString(w, "</a>")
//line index.pushup:12
		io.WriteString(w, "</p>")
//line index.pushup:13
		io.WriteString(w, "\n\n")
//line index.pushup:14
		io.WriteString(w, "<style>")
//line index.pushup:15
		io.WriteString(w, "\nul.albums {\n    display: flex;\n    justify-content: space-evenly;\n    flex-flow: row wrap;\n    list-style: none;\n    margin: 0;\n    padding: 0;\n}\n.albums li {\n    height: 10vh;\n    width: 140px;\n    border: 1px solid #ddd;\n    margin: 1em 0;\n}\n")
//line index.pushup:29
		io.WriteString(w, "</style>")
//line index.pushup:30
		io.WriteString(w, "\n\n")
//line index.pushup:31
		io.WriteString(w, "<ul ")
//line index.pushup:31
		io.WriteString(w, "class")
//line index.pushup:31
		io.WriteString(w, "=\"")
//line index.pushup:31
		io.WriteString(w, "albums")
//line index.pushup:31
		io.WriteString(w, "\">")
//line index.pushup:32
		io.WriteString(w, "\n    ")
		for _, album := range albums {
//line index.pushup:33
			io.WriteString(w, "\n        ")
//line index.pushup:33
//line index.pushup:33
			io.WriteString(w, "<li>")
//line index.pushup:33
//line index.pushup:33
			io.WriteString(w, "<a ")
//line index.pushup:33
			io.WriteString(w, "href")
//line index.pushup:33
			io.WriteString(w, "=\"")
//line index.pushup:33
			io.WriteString(w, "/crud/album/")
//line index.pushup:1
			printEscaped(w, album.id)
//line index.pushup:33
			io.WriteString(w, "\">")
//line index.pushup:33
//line index.pushup:33
			io.WriteString(w, "<b>")
//line index.pushup:33
			printEscaped(w, album.title)
			io.WriteString(w, "</b>")
//line index.pushup:33
//line index.pushup:33
			io.WriteString(w, "<br/>")
			io.WriteString(w, "</br>")
//line index.pushup:33
			printEscaped(w, album.artist)
			io.WriteString(w, "</a href=\"/crud/album/^album.id\">")
			io.WriteString(w, "</li>")
		}
//line index.pushup:35
		io.WriteString(w, "\n")
//line index.pushup:35
		io.WriteString(w, "</ul>")
//line index.pushup:36
		io.WriteString(w, "\n\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
