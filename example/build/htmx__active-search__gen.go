// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

type Pushup__active_search__14 struct {
	pushupFilePath string
}

func (t *Pushup__active_search__14) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__active_search__14) register() {
	routes.add("/htmx/active-search", t)
}

func init() {
	page := new(Pushup__active_search__14)
	page.pushupFilePath = "htmx/active-search.pushup"
	page.register()
}

func (t *Pushup__active_search__14) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__active_search__14) Respond(w http.ResponseWriter, req *http.Request) error {
	renderLayout := true
//line active-search.pushup:4
	var results []string
//line active-search.pushup:5
	search := strings.TrimSpace(strings.ToLower(req.FormValue("search")))
//line active-search.pushup:6
	if search != "" {
//line active-search.pushup:7
		for _, name := range fakeNames {
//line active-search.pushup:8
			if strings.Contains(strings.ToLower(name), search) {
//line active-search.pushup:9
				results = append(results, name)
//line active-search.pushup:10
			}
//line active-search.pushup:11
		}
//line active-search.pushup:12
	}
//line active-search.pushup:13

//line active-search.pushup:14
	if req.Header.Get("HX-Request") == "true" {
//line active-search.pushup:15
		renderLayout = false
//line active-search.pushup:16
	}
//line active-search.pushup:17

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
//line active-search.pushup:2
		io.WriteString(w, "\n\n")
//line active-search.pushup:18
		io.WriteString(w, "\n\n")
		if req.Header.Get("HX-Request") == "true" {
//line active-search.pushup:20
			io.WriteString(w, "\n    ")
//line active-search.pushup:21
			io.WriteString(w, "\n        ")
			for _, result := range results {
//line active-search.pushup:22
				io.WriteString(w, "\n            ")
//line active-search.pushup:22
//line active-search.pushup:22
				io.WriteString(w, "<tr>")
//line active-search.pushup:23
				io.WriteString(w, "\n                ")
//line active-search.pushup:23
//line active-search.pushup:23
				io.WriteString(w, "<td>")
//line active-search.pushup:23
				printEscaped(w, result)
				io.WriteString(w, "</td>")
//line active-search.pushup:24
				io.WriteString(w, "\n            ")
				io.WriteString(w, "</tr>")
			}
//line active-search.pushup:26
			io.WriteString(w, "\n    ")
		} else {
//line active-search.pushup:28
			io.WriteString(w, "\n    ")
//line active-search.pushup:29
			io.WriteString(w, "\n        ")
//line active-search.pushup:29
//line active-search.pushup:29
			io.WriteString(w, "<h1>")
//line active-search.pushup:29
			io.WriteString(w, "htmx example: Active search")
			io.WriteString(w, "</h1>")
//line active-search.pushup:30
			io.WriteString(w, "\n\n        ")
//line active-search.pushup:31
//line active-search.pushup:31
			io.WriteString(w, "<input ")
//line active-search.pushup:31
			io.WriteString(w, "class")
//line active-search.pushup:31
			io.WriteString(w, "=\"")
//line active-search.pushup:31
			io.WriteString(w, "form-control")
//line active-search.pushup:31
			io.WriteString(w, "\" ")
//line active-search.pushup:31
			io.WriteString(w, "type")
//line active-search.pushup:31
			io.WriteString(w, "=\"")
//line active-search.pushup:31
			io.WriteString(w, "search")
//line active-search.pushup:31
			io.WriteString(w, "\"\n               ")
//line active-search.pushup:32
			io.WriteString(w, "name")
//line active-search.pushup:32
			io.WriteString(w, "=\"")
//line active-search.pushup:32
			io.WriteString(w, "search")
//line active-search.pushup:32
			io.WriteString(w, "\" ")
//line active-search.pushup:32
			io.WriteString(w, "placeholder")
//line active-search.pushup:32
			io.WriteString(w, "=\"")
//line active-search.pushup:32
			io.WriteString(w, "Begin Typing To Search Users...")
//line active-search.pushup:32
			io.WriteString(w, "\"\n               ")
//line active-search.pushup:33
			io.WriteString(w, "hx-get")
//line active-search.pushup:33
			io.WriteString(w, "=\"")
//line active-search.pushup:33
			io.WriteString(w, "/htmx/active-search")
//line active-search.pushup:33
			io.WriteString(w, "\"\n               ")
//line active-search.pushup:34
			io.WriteString(w, "hx-push-url")
//line active-search.pushup:34
			io.WriteString(w, "=\"")
//line active-search.pushup:34
			io.WriteString(w, "true")
//line active-search.pushup:34
			io.WriteString(w, "\"\n               ")
//line active-search.pushup:35
			io.WriteString(w, "hx-trigger")
//line active-search.pushup:35
			io.WriteString(w, "=\"")
//line active-search.pushup:35
			io.WriteString(w, "keyup changed delay:500ms, search")
//line active-search.pushup:35
			io.WriteString(w, "\"\n               ")
//line active-search.pushup:36
			io.WriteString(w, "hx-target")
//line active-search.pushup:36
			io.WriteString(w, "=\"")
//line active-search.pushup:36
			io.WriteString(w, "#search-results")
//line active-search.pushup:36
			io.WriteString(w, "\"\n               ")
//line active-search.pushup:37
			io.WriteString(w, "hx-indicator")
//line active-search.pushup:37
			io.WriteString(w, "=\"")
//line active-search.pushup:37
			io.WriteString(w, ".htmx-indicator")
//line active-search.pushup:37
			io.WriteString(w, "\" />")
			io.WriteString(w, "</input class=\"form-control\" type=\"search\" name=\"search\" placeholder=\"Begin Typing To Search Users...\" hx-get=\"/htmx/active-search\" hx-push-url=\"true\" hx-trigger=\"keyup changed delay:500ms, search\" hx-target=\"#search-results\" hx-indicator=\".htmx-indicator\">")
//line active-search.pushup:38
			io.WriteString(w, "\n\n        ")
//line active-search.pushup:39
//line active-search.pushup:39
			io.WriteString(w, "<table ")
//line active-search.pushup:39
			io.WriteString(w, "class")
//line active-search.pushup:39
			io.WriteString(w, "=\"")
//line active-search.pushup:39
			io.WriteString(w, "table")
//line active-search.pushup:39
			io.WriteString(w, "\">")
//line active-search.pushup:40
			io.WriteString(w, "\n            ")
//line active-search.pushup:40
//line active-search.pushup:40
			io.WriteString(w, "<thead>")
//line active-search.pushup:41
			io.WriteString(w, "\n            ")
//line active-search.pushup:41
//line active-search.pushup:41
			io.WriteString(w, "<tr>")
//line active-search.pushup:42
			io.WriteString(w, "\n              ")
//line active-search.pushup:42
//line active-search.pushup:42
			io.WriteString(w, "<th>")
//line active-search.pushup:42
			io.WriteString(w, "Name")
			io.WriteString(w, "</th>")
//line active-search.pushup:43
			io.WriteString(w, "\n            ")
			io.WriteString(w, "</tr>")
//line active-search.pushup:44
			io.WriteString(w, "\n            ")
			io.WriteString(w, "</thead>")
//line active-search.pushup:45
			io.WriteString(w, "\n            ")
//line active-search.pushup:45
//line active-search.pushup:45
			io.WriteString(w, "<tbody ")
//line active-search.pushup:45
			io.WriteString(w, "id")
//line active-search.pushup:45
			io.WriteString(w, "=\"")
//line active-search.pushup:45
			io.WriteString(w, "search-results")
//line active-search.pushup:45
			io.WriteString(w, "\">")
//line active-search.pushup:46
			io.WriteString(w, "\n                ")
			for _, result := range results {
//line active-search.pushup:47
				io.WriteString(w, "\n                    ")
//line active-search.pushup:47
//line active-search.pushup:47
				io.WriteString(w, "<tr>")
//line active-search.pushup:48
				io.WriteString(w, "\n                        ")
//line active-search.pushup:48
//line active-search.pushup:48
				io.WriteString(w, "<td>")
//line active-search.pushup:48
				printEscaped(w, result)
				io.WriteString(w, "</td>")
//line active-search.pushup:49
				io.WriteString(w, "\n                    ")
				io.WriteString(w, "</tr>")
			}
//line active-search.pushup:51
			io.WriteString(w, "\n            ")
			io.WriteString(w, "</tbody id=\"search-results\">")
//line active-search.pushup:52
			io.WriteString(w, "\n        ")
			io.WriteString(w, "</table class=\"table\">")
//line active-search.pushup:53
			io.WriteString(w, "\n    ")
		}
//line active-search.pushup:55
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
