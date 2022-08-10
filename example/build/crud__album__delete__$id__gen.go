// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
)

type Pushup__DollarSign_id__6 struct {
	pushupFilePath string
}

func (t *Pushup__DollarSign_id__6) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__DollarSign_id__6) register() {
	routes.add("/crud/album/delete/:id", t)
}

func init() {
	page := new(Pushup__DollarSign_id__6)
	page.pushupFilePath = "crud/album/delete/$id.pushup"
	page.register()
}

func (t *Pushup__DollarSign_id__6) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__DollarSign_id__6) Respond(w http.ResponseWriter, req *http.Request) error {
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

//line $id.pushup:14
	if req.Method == "POST" && req.FormValue("_method") == "delete" {
//line $id.pushup:15
		if err := deleteAlbum(DB, id); err != nil {
//line $id.pushup:16
			return err
//line $id.pushup:17
		}
//line $id.pushup:18
		http.Redirect(w, req, "/crud", 301)
//line $id.pushup:19
		return nil
//line $id.pushup:20
	}
//line $id.pushup:21

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
//line $id.pushup:22
		io.WriteString(w, "\n\n")
//line $id.pushup:23
		io.WriteString(w, "<h1>")
//line $id.pushup:23
		io.WriteString(w, "Delete ")
//line $id.pushup:23
		printEscaped(w, album.title)
//line $id.pushup:23
		io.WriteString(w, "?")
//line $id.pushup:23
		io.WriteString(w, "</h1>")
//line $id.pushup:24
		io.WriteString(w, "\n\n")
//line $id.pushup:25
		io.WriteString(w, "<p>")
//line $id.pushup:25
		io.WriteString(w, "Are you sure?")
//line $id.pushup:25
		io.WriteString(w, "</p>")
//line $id.pushup:26
		io.WriteString(w, "\n\n")
//line $id.pushup:27
		io.WriteString(w, "<p>")
//line $id.pushup:27
		io.WriteString(w, "<a ")
//line $id.pushup:27
		io.WriteString(w, "href")
//line $id.pushup:27
		io.WriteString(w, "=\"")
//line $id.pushup:27
		io.WriteString(w, "/crud/album/")
//line $id.pushup:1
		printEscaped(w, album.id)
//line $id.pushup:27
		io.WriteString(w, "\">")
//line $id.pushup:27
		io.WriteString(w, "No, get me out of here")
//line $id.pushup:27
		io.WriteString(w, "</a>")
//line $id.pushup:27
		io.WriteString(w, "</p>")
//line $id.pushup:28
		io.WriteString(w, "\n")
//line $id.pushup:28
		io.WriteString(w, "<form ")
//line $id.pushup:28
		io.WriteString(w, "method")
//line $id.pushup:28
		io.WriteString(w, "=\"")
//line $id.pushup:28
		io.WriteString(w, "post")
//line $id.pushup:28
		io.WriteString(w, "\">")
//line $id.pushup:29
		io.WriteString(w, "\n    ")
//line $id.pushup:29
		io.WriteString(w, "<input ")
//line $id.pushup:29
		io.WriteString(w, "type")
//line $id.pushup:29
		io.WriteString(w, "=\"")
//line $id.pushup:29
		io.WriteString(w, "hidden")
//line $id.pushup:29
		io.WriteString(w, "\" ")
//line $id.pushup:29
		io.WriteString(w, "name")
//line $id.pushup:29
		io.WriteString(w, "=\"")
//line $id.pushup:29
		io.WriteString(w, "_method")
//line $id.pushup:29
		io.WriteString(w, "\" ")
//line $id.pushup:29
		io.WriteString(w, "value")
//line $id.pushup:29
		io.WriteString(w, "=\"")
//line $id.pushup:29
		io.WriteString(w, "delete")
//line $id.pushup:29
		io.WriteString(w, "\">")
//line $id.pushup:30
		io.WriteString(w, "\n    ")
//line $id.pushup:30
		io.WriteString(w, "<input ")
//line $id.pushup:30
		io.WriteString(w, "type")
//line $id.pushup:30
		io.WriteString(w, "=\"")
//line $id.pushup:30
		io.WriteString(w, "hidden")
//line $id.pushup:30
		io.WriteString(w, "\" ")
//line $id.pushup:30
		io.WriteString(w, "name")
//line $id.pushup:30
		io.WriteString(w, "=\"")
//line $id.pushup:30
		io.WriteString(w, "id")
//line $id.pushup:30
		io.WriteString(w, "\" ")
//line $id.pushup:30
		io.WriteString(w, "value")
//line $id.pushup:30
		io.WriteString(w, "=\"")
//line $id.pushup:1
		printEscaped(w, album.id)
//line $id.pushup:30
		io.WriteString(w, "\">")
//line $id.pushup:31
		io.WriteString(w, "\n    ")
//line $id.pushup:31
		io.WriteString(w, "<button>")
//line $id.pushup:31
		io.WriteString(w, "Yes, delete")
//line $id.pushup:31
		io.WriteString(w, "</button>")
//line $id.pushup:32
		io.WriteString(w, "\n")
//line $id.pushup:32
		io.WriteString(w, "</form>")
//line $id.pushup:33
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
