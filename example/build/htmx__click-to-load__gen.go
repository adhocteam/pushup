// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
)

type Pushup__click_to_load__15 struct {
	pushupFilePath string
}

func (t *Pushup__click_to_load__15) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__click_to_load__15) register() {
	routes.add("/htmx/click-to-load", t)
}

func init() {
	page := new(Pushup__click_to_load__15)
	page.pushupFilePath = "htmx/click-to-load.pushup"
	page.register()
}

func (t *Pushup__click_to_load__15) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__click_to_load__15) Respond(w http.ResponseWriter, req *http.Request) error {
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
//line click-to-load.pushup:2
		io.WriteString(w, "\n\n")
//line click-to-load.pushup:3
		io.WriteString(w, "<style>")
//line click-to-load.pushup:4
		io.WriteString(w, "\ntable { width: 100%; border-collapse: collapse; }\nth, td { padding: 5px 8px; }\nthead tr { border-bottom: 2px solid #ccc; }\ntbody tr { border-bottom: 1px solid #ccc; }\n")
//line click-to-load.pushup:8
		io.WriteString(w, "</style>")
//line click-to-load.pushup:9
		io.WriteString(w, "\n\n")
//line click-to-load.pushup:11
		var alphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
//line click-to-load.pushup:12
		uid := func() string {
//line click-to-load.pushup:13
			b := make([]rune, 16)
//line click-to-load.pushup:14
			for i := range b {
//line click-to-load.pushup:15
				b[i] = alphabet[rand.Intn(len(alphabet))]
//line click-to-load.pushup:16
			}
//line click-to-load.pushup:17
			return string(b)
//line click-to-load.pushup:18
		}
//line click-to-load.pushup:19

//line click-to-load.pushup:20
		io.WriteString(w, "\n\n")
//line click-to-load.pushup:21
		io.WriteString(w, "<h1>")
//line click-to-load.pushup:21
		io.WriteString(w, "htmx example: Click to load")
//line click-to-load.pushup:21
		io.WriteString(w, "</h1>")
//line click-to-load.pushup:22
		io.WriteString(w, "\n\n")
//line click-to-load.pushup:23
		io.WriteString(w, "<table>")
//line click-to-load.pushup:24
		io.WriteString(w, "\n    ")
//line click-to-load.pushup:24
		io.WriteString(w, "<thead>")
//line click-to-load.pushup:25
		io.WriteString(w, "\n        ")
//line click-to-load.pushup:25
		io.WriteString(w, "<tr>")
//line click-to-load.pushup:26
		io.WriteString(w, "\n            ")
//line click-to-load.pushup:26
		io.WriteString(w, "<th>")
//line click-to-load.pushup:26
		io.WriteString(w, "Name")
//line click-to-load.pushup:26
		io.WriteString(w, "</th>")
//line click-to-load.pushup:27
		io.WriteString(w, "\n            ")
//line click-to-load.pushup:27
		io.WriteString(w, "<th>")
//line click-to-load.pushup:27
		io.WriteString(w, "Email")
//line click-to-load.pushup:27
		io.WriteString(w, "</th>")
//line click-to-load.pushup:28
		io.WriteString(w, "\n            ")
//line click-to-load.pushup:28
		io.WriteString(w, "<th>")
//line click-to-load.pushup:28
		io.WriteString(w, "ID")
//line click-to-load.pushup:28
		io.WriteString(w, "</th>")
//line click-to-load.pushup:29
		io.WriteString(w, "\n        ")
//line click-to-load.pushup:29
		io.WriteString(w, "</tr>")
//line click-to-load.pushup:30
		io.WriteString(w, "\n    ")
//line click-to-load.pushup:30
		io.WriteString(w, "</thead>")
//line click-to-load.pushup:31
		io.WriteString(w, "\n    ")
//line click-to-load.pushup:31
		io.WriteString(w, "<tbody>")
//line click-to-load.pushup:32
		io.WriteString(w, "\n        ")
		for i := 0; i < 5; i++ {
//line click-to-load.pushup:33
			io.WriteString(w, "\n            ")
//line click-to-load.pushup:33
//line click-to-load.pushup:33
			io.WriteString(w, "<tr>")
//line click-to-load.pushup:34
			io.WriteString(w, "\n                ")
//line click-to-load.pushup:34
//line click-to-load.pushup:34
			io.WriteString(w, "<td>")
//line click-to-load.pushup:34
			io.WriteString(w, "Agent Smith")
			io.WriteString(w, "</td>")
//line click-to-load.pushup:35
			io.WriteString(w, "\n                ")
//line click-to-load.pushup:35
//line click-to-load.pushup:35
			io.WriteString(w, "<td>")
//line click-to-load.pushup:35
			io.WriteString(w, "void")
//line click-to-load.pushup:35
			printEscaped(w, i)
//line click-to-load.pushup:35
			io.WriteString(w, "&#x40;pizza.null")
			io.WriteString(w, "</td>")
//line click-to-load.pushup:36
			io.WriteString(w, "\n                ")
//line click-to-load.pushup:36
//line click-to-load.pushup:36
			io.WriteString(w, "<td>")
//line click-to-load.pushup:36
			printEscaped(w, uid())
			io.WriteString(w, "</td>")
//line click-to-load.pushup:37
			io.WriteString(w, "\n            ")
			io.WriteString(w, "</tr>")
		}
//line click-to-load.pushup:39
		io.WriteString(w, "\n        ")
//line click-to-load.pushup:39
		io.WriteString(w, "<tr ")
//line click-to-load.pushup:39
		io.WriteString(w, "id")
//line click-to-load.pushup:39
		io.WriteString(w, "=\"")
//line click-to-load.pushup:39
		io.WriteString(w, "replace-me")
//line click-to-load.pushup:39
		io.WriteString(w, "\">")
//line click-to-load.pushup:40
		io.WriteString(w, "\n            ")
//line click-to-load.pushup:40
		io.WriteString(w, "<td ")
//line click-to-load.pushup:40
		io.WriteString(w, "colspan")
//line click-to-load.pushup:40
		io.WriteString(w, "=")
//line click-to-load.pushup:40
		io.WriteString(w, "3")
//line click-to-load.pushup:40
		io.WriteString(w, ">")
//line click-to-load.pushup:41
		io.WriteString(w, "\n                ")
//line click-to-load.pushup:41
		io.WriteString(w, "<button ")
//line click-to-load.pushup:41
		io.WriteString(w, "hx-get")
//line click-to-load.pushup:41
		io.WriteString(w, "=\"")
//line click-to-load.pushup:41
		io.WriteString(w, "./contacts?page=1")
//line click-to-load.pushup:41
		io.WriteString(w, "\" ")
//line click-to-load.pushup:41
		io.WriteString(w, "hx-target")
//line click-to-load.pushup:41
		io.WriteString(w, "=\"")
//line click-to-load.pushup:41
		io.WriteString(w, "#replace-me")
//line click-to-load.pushup:41
		io.WriteString(w, "\" ")
//line click-to-load.pushup:41
		io.WriteString(w, "hx-swap")
//line click-to-load.pushup:41
		io.WriteString(w, "=\"")
//line click-to-load.pushup:41
		io.WriteString(w, "outerHTML")
//line click-to-load.pushup:41
		io.WriteString(w, "\">")
//line click-to-load.pushup:42
		io.WriteString(w, "\n                    Load more agents\n                ")
//line click-to-load.pushup:43
		io.WriteString(w, "</button>")
//line click-to-load.pushup:44
		io.WriteString(w, "\n            ")
//line click-to-load.pushup:44
		io.WriteString(w, "</td>")
//line click-to-load.pushup:45
		io.WriteString(w, "\n        ")
//line click-to-load.pushup:45
		io.WriteString(w, "</tr>")
//line click-to-load.pushup:46
		io.WriteString(w, "\n    ")
//line click-to-load.pushup:46
		io.WriteString(w, "</tbody>")
//line click-to-load.pushup:47
		io.WriteString(w, "\n")
//line click-to-load.pushup:47
		io.WriteString(w, "</table>")
//line click-to-load.pushup:48
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
