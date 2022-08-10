// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"math/rand"
	"net/http"
	"strconv"
)

type Pushup__contacts__16 struct {
	pushupFilePath string
}

func (t *Pushup__contacts__16) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__contacts__16) register() {
	routes.add("/htmx/contacts", t)
}

func init() {
	page := new(Pushup__contacts__16)
	page.pushupFilePath = "htmx/contacts.pushup"
	page.register()
}

func (t *Pushup__contacts__16) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__contacts__16) Respond(w http.ResponseWriter, req *http.Request) error {
	// Begin user Go code and HTML
	{
//line contacts.pushup:2
		io.WriteString(w, "\n\n")
//line contacts.pushup:4
		io.WriteString(w, "\n")
//line contacts.pushup:5
		io.WriteString(w, "\n\n")
//line contacts.pushup:7
		pageQuery := req.FormValue("page")
//line contacts.pushup:8
		if pageQuery == "" {
//line contacts.pushup:9
			pageQuery = "1"
//line contacts.pushup:10
		}
//line contacts.pushup:11

//line contacts.pushup:12
		page, err := strconv.Atoi(pageQuery)
//line contacts.pushup:13
		if err != nil {
//line contacts.pushup:14
			return err
//line contacts.pushup:15
		}
//line contacts.pushup:16

//line contacts.pushup:17
		var alphabet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
//line contacts.pushup:18
		uid := func() string {
//line contacts.pushup:19
			b := make([]rune, 16)
//line contacts.pushup:20
			for i := range b {
//line contacts.pushup:21
				b[i] = alphabet[rand.Intn(len(alphabet))]
//line contacts.pushup:22
			}
//line contacts.pushup:23
			return string(b)
//line contacts.pushup:24
		}
//line contacts.pushup:25

//line contacts.pushup:26
		io.WriteString(w, "\n\n")
		for i := 0; i < 5; i++ {
//line contacts.pushup:28
			io.WriteString(w, "\n")
//line contacts.pushup:28
//line contacts.pushup:28
			io.WriteString(w, "<tr>")
//line contacts.pushup:29
			io.WriteString(w, "\n    ")
//line contacts.pushup:29
//line contacts.pushup:29
			io.WriteString(w, "<td>")
//line contacts.pushup:29
			io.WriteString(w, "Agent Smith")
			io.WriteString(w, "</td>")
//line contacts.pushup:30
			io.WriteString(w, "\n    ")
//line contacts.pushup:30
//line contacts.pushup:30
			io.WriteString(w, "<td>")
//line contacts.pushup:30
			io.WriteString(w, "void")
//line contacts.pushup:30
			printEscaped(w, page*5+i)
//line contacts.pushup:30
			io.WriteString(w, "&#x40;pizza.null")
			io.WriteString(w, "</td>")
//line contacts.pushup:31
			io.WriteString(w, "\n    ")
//line contacts.pushup:31
//line contacts.pushup:31
			io.WriteString(w, "<td>")
//line contacts.pushup:31
			printEscaped(w, uid())
			io.WriteString(w, "</td>")
//line contacts.pushup:32
			io.WriteString(w, "\n")
			io.WriteString(w, "</tr>")
		}
//line contacts.pushup:34
		io.WriteString(w, "\n")
//line contacts.pushup:34
		io.WriteString(w, "<tr ")
//line contacts.pushup:34
		io.WriteString(w, "id")
//line contacts.pushup:34
		io.WriteString(w, "=\"")
//line contacts.pushup:34
		io.WriteString(w, "replace-me")
//line contacts.pushup:34
		io.WriteString(w, "\">")
//line contacts.pushup:35
		io.WriteString(w, "\n    ")
//line contacts.pushup:35
		io.WriteString(w, "<td ")
//line contacts.pushup:35
		io.WriteString(w, "colspan")
//line contacts.pushup:35
		io.WriteString(w, "=")
//line contacts.pushup:35
		io.WriteString(w, "3")
//line contacts.pushup:35
		io.WriteString(w, ">")
//line contacts.pushup:36
		io.WriteString(w, "\n        ")
//line contacts.pushup:36
		io.WriteString(w, "<button ")
//line contacts.pushup:36
		io.WriteString(w, "hx-get")
//line contacts.pushup:36
		io.WriteString(w, "=\"")
//line contacts.pushup:36
		io.WriteString(w, "./contacts?page=")
//line contacts.pushup:1
		printEscaped(w, page+1)
//line contacts.pushup:36
		io.WriteString(w, "\" ")
//line contacts.pushup:36
		io.WriteString(w, "hx-target")
//line contacts.pushup:36
		io.WriteString(w, "=\"")
//line contacts.pushup:36
		io.WriteString(w, "#replace-me")
//line contacts.pushup:36
		io.WriteString(w, "\" ")
//line contacts.pushup:36
		io.WriteString(w, "hx-swap")
//line contacts.pushup:36
		io.WriteString(w, "=\"")
//line contacts.pushup:36
		io.WriteString(w, "outerHTML")
//line contacts.pushup:36
		io.WriteString(w, "\">")
//line contacts.pushup:37
		io.WriteString(w, "\n            Load more agents\n        ")
//line contacts.pushup:38
		io.WriteString(w, "</button>")
//line contacts.pushup:39
		io.WriteString(w, "\n    ")
//line contacts.pushup:39
		io.WriteString(w, "</td>")
//line contacts.pushup:40
		io.WriteString(w, "\n")
//line contacts.pushup:40
		io.WriteString(w, "</tr>")
//line contacts.pushup:41
		io.WriteString(w, "\n")
		// End user Go code and HTML
	}
	return nil
}
