// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
)

type Pushup__DollarSign_id__5 struct {
	pushupFilePath string
}

func (t *Pushup__DollarSign_id__5) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__DollarSign_id__5) register() {
	routes.add("/crud/album/:id", t)
}

func init() {
	page := new(Pushup__DollarSign_id__5)
	page.pushupFilePath = "crud/album/$id.pushup"
	page.register()
}

func (t *Pushup__DollarSign_id__5) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__DollarSign_id__5) Respond(w http.ResponseWriter, req *http.Request) error {
	renderLayout := true
//line $id.pushup:4
	id, err := strconv.Atoi(getParam(req, "id"))
//line $id.pushup:5
	if err != nil {
//line $id.pushup:6
		return err
//line $id.pushup:7
	}
//line $id.pushup:8

//line $id.pushup:9
	album, err := getAlbumById(DB, id)
//line $id.pushup:10
	if err != nil {
//line $id.pushup:11
		return err
//line $id.pushup:12
	}
//line $id.pushup:13

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
//line $id.pushup:2
		io.WriteString(w, "\n\n")
//line $id.pushup:14
		io.WriteString(w, "\n\n")
//line $id.pushup:15
		io.WriteString(w, "<h1>")
//line $id.pushup:15
		io.WriteString(w, "Album")
//line $id.pushup:15
		io.WriteString(w, "</h1>")
//line $id.pushup:16
		io.WriteString(w, "\n\n")
//line $id.pushup:17
		io.WriteString(w, "<p>")
//line $id.pushup:17
		io.WriteString(w, "<a ")
//line $id.pushup:17
		io.WriteString(w, "href")
//line $id.pushup:17
		io.WriteString(w, "=\"")
//line $id.pushup:17
		io.WriteString(w, "/crud")
//line $id.pushup:17
		io.WriteString(w, "\">")
//line $id.pushup:17
		io.WriteString(w, "Back to album list")
//line $id.pushup:17
		io.WriteString(w, "</a>")
//line $id.pushup:17
		io.WriteString(w, "</p>")
//line $id.pushup:18
		io.WriteString(w, "\n\n")
//line $id.pushup:19
		io.WriteString(w, "<style>")
//line $id.pushup:20
		io.WriteString(w, "\ndl { display: flex; flex-flow: row wrap; }\ndt { font-weight: bold; flex-basis: 20%; padding: 0.1em; }\ndd { font-weight: normal; flex-basis: 70%; flex-grow: 1; padding: 0.1em; }\n")
//line $id.pushup:23
		io.WriteString(w, "</style>")
//line $id.pushup:24
		io.WriteString(w, "\n\n")
//line $id.pushup:25
		io.WriteString(w, "<dl>")
//line $id.pushup:26
		io.WriteString(w, "\n    ")
//line $id.pushup:26
		io.WriteString(w, "<dt>")
//line $id.pushup:26
		io.WriteString(w, "Artist")
//line $id.pushup:26
		io.WriteString(w, "</dt>")
//line $id.pushup:27
		io.WriteString(w, "\n    ")
//line $id.pushup:27
		io.WriteString(w, "<dd>")
//line $id.pushup:27
		printEscaped(w, album.artist)
//line $id.pushup:27
		io.WriteString(w, "</dd>")
//line $id.pushup:28
		io.WriteString(w, "\n\n    ")
//line $id.pushup:29
		io.WriteString(w, "<dt>")
//line $id.pushup:29
		io.WriteString(w, "Title")
//line $id.pushup:29
		io.WriteString(w, "</dt>")
//line $id.pushup:30
		io.WriteString(w, "\n    ")
//line $id.pushup:30
		io.WriteString(w, "<dd>")
//line $id.pushup:30
		printEscaped(w, album.title)
//line $id.pushup:30
		io.WriteString(w, "</dd>")
//line $id.pushup:31
		io.WriteString(w, "\n\n    ")
//line $id.pushup:32
		io.WriteString(w, "<dt>")
//line $id.pushup:32
		io.WriteString(w, "Released")
//line $id.pushup:32
		io.WriteString(w, "</dt>")
//line $id.pushup:33
		io.WriteString(w, "\n    ")
//line $id.pushup:33
		io.WriteString(w, "<dd>")
//line $id.pushup:33
		printEscaped(w, album.released)
//line $id.pushup:33
		io.WriteString(w, "</dd>")
//line $id.pushup:34
		io.WriteString(w, "\n\n    ")
//line $id.pushup:35
		io.WriteString(w, "<dt>")
//line $id.pushup:35
		io.WriteString(w, "Length")
//line $id.pushup:35
		io.WriteString(w, "</dt>")
//line $id.pushup:36
		io.WriteString(w, "\n    ")
//line $id.pushup:36
		io.WriteString(w, "<dd>")
//line $id.pushup:36
		printEscaped(w, album.length)
//line $id.pushup:36
		io.WriteString(w, " minutes")
//line $id.pushup:36
		io.WriteString(w, "</dd>")
//line $id.pushup:37
		io.WriteString(w, "\n")
//line $id.pushup:37
		io.WriteString(w, "</dl>")
//line $id.pushup:38
		io.WriteString(w, "\n\n")
//line $id.pushup:39
		io.WriteString(w, "<p>")
//line $id.pushup:39
		io.WriteString(w, "<a ")
//line $id.pushup:39
		io.WriteString(w, "href")
//line $id.pushup:39
		io.WriteString(w, "=\"")
//line $id.pushup:39
		io.WriteString(w, "./edit/")
//line $id.pushup:1
		printEscaped(w, album.id)
//line $id.pushup:39
		io.WriteString(w, "\">")
//line $id.pushup:39
		io.WriteString(w, "Edit")
//line $id.pushup:39
		io.WriteString(w, "</a>")
//line $id.pushup:39
		io.WriteString(w, ", ")
//line $id.pushup:39
		io.WriteString(w, "<a ")
//line $id.pushup:39
		io.WriteString(w, "href")
//line $id.pushup:39
		io.WriteString(w, "=\"")
//line $id.pushup:39
		io.WriteString(w, "./delete/")
//line $id.pushup:1
		printEscaped(w, album.id)
//line $id.pushup:39
		io.WriteString(w, "\">")
//line $id.pushup:39
		io.WriteString(w, "Delete &hellip;")
//line $id.pushup:39
		io.WriteString(w, "</a>")
//line $id.pushup:39
		io.WriteString(w, "</p>")
//line $id.pushup:40
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
