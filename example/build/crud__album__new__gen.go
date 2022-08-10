// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Pushup__new__8 struct {
	pushupFilePath string
}

func (t *Pushup__new__8) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__new__8) register() {
	routes.add("/crud/album/new", t)
}

func init() {
	page := new(Pushup__new__8)
	page.pushupFilePath = "crud/album/new.pushup"
	page.register()
}

func (t *Pushup__new__8) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__new__8) Respond(w http.ResponseWriter, req *http.Request) error {
	renderLayout := true
//line new.pushup:9
	isNumber := func(val string) (int, bool) {
//line new.pushup:10
		n, err := strconv.Atoi(val)
//line new.pushup:11
		if err != nil {
//line new.pushup:12
			return 0, false
//line new.pushup:13
		}
//line new.pushup:14
		return n, true
//line new.pushup:15
	}
//line new.pushup:16

//line new.pushup:17
	errors := make(map[string]string)
//line new.pushup:18

//line new.pushup:19
	if req.Method == "POST" {
//line new.pushup:20
		artist := strings.TrimSpace(req.FormValue("artist"))
//line new.pushup:21
		title := strings.TrimSpace(req.FormValue("title"))
//line new.pushup:22
		releasedRaw := strings.TrimSpace(req.FormValue("released"))
//line new.pushup:23
		lengthRaw := strings.TrimSpace(req.FormValue("length"))
//line new.pushup:24

//line new.pushup:25
		if artist == "" {
//line new.pushup:26
			errors["artist"] = "artist name is required"
//line new.pushup:27
		}
//line new.pushup:28
		if title == "" {
//line new.pushup:29
			errors["title"] = "title is required"
//line new.pushup:30
		}
//line new.pushup:31
		released, releasedIsNum := isNumber(releasedRaw)
//line new.pushup:32
		if releasedRaw == "" {
//line new.pushup:33
			errors["released"] = "release year is required"
//line new.pushup:34
		} else if !releasedIsNum || !(released >= 1900 && released <= time.Now().Year()) {
//line new.pushup:35
			errors["released"] = "release year must be between 1900 and this year"
//line new.pushup:36
		}
//line new.pushup:37
		length, lengthIsNum := isNumber(lengthRaw)
//line new.pushup:38
		if lengthRaw == "" {
//line new.pushup:39
			errors["length"] = "length is required"
//line new.pushup:40
		} else if !lengthIsNum || !(length > 0) {
//line new.pushup:41
			errors["length"] = "length must be a number greater than 0"
//line new.pushup:42
		}
//line new.pushup:43

//line new.pushup:44
		if len(errors) == 0 {
//line new.pushup:45
			a := &album{artist: artist, title: title, released: released, length: length}
//line new.pushup:46
			if err := addAlbum(DB, a); err != nil {
//line new.pushup:47
				log.Printf("error: %v", err)
//line new.pushup:48
				http.Error(w, http.StatusText(500), 500)
//line new.pushup:49
				return nil
//line new.pushup:50
			}
//line new.pushup:51
			http.Redirect(w, req, "/crud", 301)
//line new.pushup:52
		}
//line new.pushup:53
	}
//line new.pushup:54

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
//line new.pushup:2
		io.WriteString(w, "\n\n")
//line new.pushup:4
		io.WriteString(w, "\n")
//line new.pushup:5
		io.WriteString(w, "\n")
//line new.pushup:6
		io.WriteString(w, "\n")
//line new.pushup:7
		io.WriteString(w, "\n\n")
//line new.pushup:55
		io.WriteString(w, "\n\n")
//line new.pushup:56
		io.WriteString(w, "<h1>")
//line new.pushup:56
		io.WriteString(w, "Add new album")
//line new.pushup:56
		io.WriteString(w, "</h1>")
//line new.pushup:57
		io.WriteString(w, "\n\n")
//line new.pushup:58
		io.WriteString(w, "<p>")
//line new.pushup:58
		io.WriteString(w, "<a ")
//line new.pushup:58
		io.WriteString(w, "href")
//line new.pushup:58
		io.WriteString(w, "=\"")
//line new.pushup:58
		io.WriteString(w, "/crud")
//line new.pushup:58
		io.WriteString(w, "\">")
//line new.pushup:58
		io.WriteString(w, "Cancel")
//line new.pushup:58
		io.WriteString(w, "</a>")
//line new.pushup:58
		io.WriteString(w, "</p>")
//line new.pushup:59
		io.WriteString(w, "\n\n")
//line new.pushup:60
		io.WriteString(w, "<style>")
//line new.pushup:61
		io.WriteString(w, "\nform { border: 1px solid #ddd; background: #f3f4f0; padding: 2rem; }\nform h2, form h3 { margin: 0 0 0.5rem 0; }\nform section { margin: 0 0 1rem 0; }\n.form-element { display: flex; justify-content: align-items: center; flex-end; padding: 1em 0; }\n.form-element > label { flex: 1; }\n.form-element > input { flex: 2; padding: 0.5rem; }\nform button { font-size: 1.5rem; padding: 1rem; }\n")
//line new.pushup:68
		io.WriteString(w, "</style>")
//line new.pushup:69
		io.WriteString(w, "\n\n")
//line new.pushup:70
		io.WriteString(w, "<form ")
//line new.pushup:70
		io.WriteString(w, "method")
//line new.pushup:70
		io.WriteString(w, "=\"")
//line new.pushup:70
		io.WriteString(w, "post")
//line new.pushup:70
		io.WriteString(w, "\">")
//line new.pushup:71
		io.WriteString(w, "\n    ")
		if len(errors) > 0 {
//line new.pushup:72
			io.WriteString(w, "\n        ")
//line new.pushup:72
//line new.pushup:72
			io.WriteString(w, "<section ")
//line new.pushup:72
			io.WriteString(w, "style")
//line new.pushup:72
			io.WriteString(w, "=\"")
//line new.pushup:72
			io.WriteString(w, "color: red")
//line new.pushup:72
			io.WriteString(w, "\">")
//line new.pushup:73
			io.WriteString(w, "\n            ")
//line new.pushup:73
//line new.pushup:73
			io.WriteString(w, "<h3>")
//line new.pushup:73
			io.WriteString(w, "Errors")
			io.WriteString(w, "</h3>")
//line new.pushup:74
			io.WriteString(w, "\n            ")
//line new.pushup:74
//line new.pushup:74
			io.WriteString(w, "<ul>")
//line new.pushup:75
			io.WriteString(w, "\n                ")
			for _, message := range errors {
//line new.pushup:76
				io.WriteString(w, "\n                    ")
//line new.pushup:76
//line new.pushup:76
				io.WriteString(w, "<li>")
//line new.pushup:76
				printEscaped(w, message)
				io.WriteString(w, "</li>")
			}
//line new.pushup:78
			io.WriteString(w, "\n            ")
			io.WriteString(w, "</ul>")
//line new.pushup:79
			io.WriteString(w, "\n        ")
			io.WriteString(w, "</section style=\"color: red\">")
		}
//line new.pushup:81
		io.WriteString(w, "\n    ")
//line new.pushup:81
		io.WriteString(w, "<section>")
//line new.pushup:82
		io.WriteString(w, "\n        ")
//line new.pushup:82
		io.WriteString(w, "<h2>")
//line new.pushup:82
		io.WriteString(w, "Artist")
//line new.pushup:82
		io.WriteString(w, "</h2>")
//line new.pushup:83
		io.WriteString(w, "\n        ")
//line new.pushup:83
		io.WriteString(w, "<div ")
//line new.pushup:83
		io.WriteString(w, "class")
//line new.pushup:83
		io.WriteString(w, "=\"")
//line new.pushup:83
		io.WriteString(w, "form-element")
//line new.pushup:83
		io.WriteString(w, "\">")
//line new.pushup:84
		io.WriteString(w, "\n            ")
//line new.pushup:84
		io.WriteString(w, "<label ")
//line new.pushup:84
		io.WriteString(w, "for")
//line new.pushup:84
		io.WriteString(w, "=\"")
//line new.pushup:84
		io.WriteString(w, "artist")
//line new.pushup:84
		io.WriteString(w, "\">")
//line new.pushup:84
		io.WriteString(w, "Artist name")
//line new.pushup:84
		io.WriteString(w, "</label>")
//line new.pushup:85
		io.WriteString(w, "\n            ")
//line new.pushup:85
		io.WriteString(w, "<!-- NOTE there is a workaround for a parsing issue: if you refer to Go strings\n                 inside a double-quoted HTML attribute, then the Go string must use the backquote\n                 or raw literal style, not the double-quoted style. (or use single-quoted HTML\n                 attributes.) arguably, Pushup should emit quoted wrapped attributes to avoid\n                 potential quote escaping issue of the Go code having quotes that inadvertently\n                 prematurely close the rendered HTML attribute. -->")
//line new.pushup:91
		io.WriteString(w, "\n            ")
//line new.pushup:91
		io.WriteString(w, "<input ")
//line new.pushup:91
		io.WriteString(w, "type")
//line new.pushup:91
		io.WriteString(w, "=\"")
//line new.pushup:91
		io.WriteString(w, "text")
//line new.pushup:91
		io.WriteString(w, "\" ")
//line new.pushup:91
		io.WriteString(w, "name")
//line new.pushup:91
		io.WriteString(w, "=\"")
//line new.pushup:91
		io.WriteString(w, "artist")
//line new.pushup:91
		io.WriteString(w, "\" ")
//line new.pushup:91
		io.WriteString(w, "id")
//line new.pushup:91
		io.WriteString(w, "=\"")
//line new.pushup:91
		io.WriteString(w, "artist")
//line new.pushup:91
		io.WriteString(w, "\" ")
//line new.pushup:91
		io.WriteString(w, "value")
//line new.pushup:91
		io.WriteString(w, "=\"")
//line new.pushup:1
		printEscaped(w, req.FormValue(`artist`))
//line new.pushup:91
		io.WriteString(w, "\">")
//line new.pushup:92
		io.WriteString(w, "\n        ")
//line new.pushup:92
		io.WriteString(w, "</div>")
//line new.pushup:93
		io.WriteString(w, "\n    ")
//line new.pushup:93
		io.WriteString(w, "</section>")
//line new.pushup:94
		io.WriteString(w, "\n\n    ")
//line new.pushup:95
		io.WriteString(w, "<section>")
//line new.pushup:96
		io.WriteString(w, "\n        ")
//line new.pushup:96
		io.WriteString(w, "<h2>")
//line new.pushup:96
		io.WriteString(w, "Album info")
//line new.pushup:96
		io.WriteString(w, "</h2>")
//line new.pushup:97
		io.WriteString(w, "\n        ")
//line new.pushup:97
		io.WriteString(w, "<div ")
//line new.pushup:97
		io.WriteString(w, "class")
//line new.pushup:97
		io.WriteString(w, "=\"")
//line new.pushup:97
		io.WriteString(w, "form-element")
//line new.pushup:97
		io.WriteString(w, "\">")
//line new.pushup:98
		io.WriteString(w, "\n            ")
//line new.pushup:98
		io.WriteString(w, "<label ")
//line new.pushup:98
		io.WriteString(w, "for")
//line new.pushup:98
		io.WriteString(w, "=\"")
//line new.pushup:98
		io.WriteString(w, "title")
//line new.pushup:98
		io.WriteString(w, "\">")
//line new.pushup:98
		io.WriteString(w, "Title")
//line new.pushup:98
		io.WriteString(w, "</label>")
//line new.pushup:99
		io.WriteString(w, "\n            ")
//line new.pushup:99
		io.WriteString(w, "<input ")
//line new.pushup:99
		io.WriteString(w, "type")
//line new.pushup:99
		io.WriteString(w, "=\"")
//line new.pushup:99
		io.WriteString(w, "text")
//line new.pushup:99
		io.WriteString(w, "\" ")
//line new.pushup:99
		io.WriteString(w, "name")
//line new.pushup:99
		io.WriteString(w, "=\"")
//line new.pushup:99
		io.WriteString(w, "title")
//line new.pushup:99
		io.WriteString(w, "\" ")
//line new.pushup:99
		io.WriteString(w, "id")
//line new.pushup:99
		io.WriteString(w, "=\"")
//line new.pushup:99
		io.WriteString(w, "title")
//line new.pushup:99
		io.WriteString(w, "\" ")
//line new.pushup:99
		io.WriteString(w, "value")
//line new.pushup:99
		io.WriteString(w, "=\"")
//line new.pushup:1
		printEscaped(w, req.FormValue(`title`))
//line new.pushup:99
		io.WriteString(w, "\">")
//line new.pushup:100
		io.WriteString(w, "\n        ")
//line new.pushup:100
		io.WriteString(w, "</div>")
//line new.pushup:101
		io.WriteString(w, "\n        ")
//line new.pushup:101
		io.WriteString(w, "<div ")
//line new.pushup:101
		io.WriteString(w, "class")
//line new.pushup:101
		io.WriteString(w, "=\"")
//line new.pushup:101
		io.WriteString(w, "form-element")
//line new.pushup:101
		io.WriteString(w, "\">")
//line new.pushup:102
		io.WriteString(w, "\n            ")
//line new.pushup:102
		io.WriteString(w, "<label ")
//line new.pushup:102
		io.WriteString(w, "for")
//line new.pushup:102
		io.WriteString(w, "=\"")
//line new.pushup:102
		io.WriteString(w, "released")
//line new.pushup:102
		io.WriteString(w, "\">")
//line new.pushup:102
		io.WriteString(w, "Year released")
//line new.pushup:102
		io.WriteString(w, "</label>")
//line new.pushup:103
		io.WriteString(w, "\n            ")
//line new.pushup:103
		io.WriteString(w, "<input ")
//line new.pushup:103
		io.WriteString(w, "type")
//line new.pushup:103
		io.WriteString(w, "=\"")
//line new.pushup:103
		io.WriteString(w, "text")
//line new.pushup:103
		io.WriteString(w, "\" ")
//line new.pushup:103
		io.WriteString(w, "name")
//line new.pushup:103
		io.WriteString(w, "=\"")
//line new.pushup:103
		io.WriteString(w, "released")
//line new.pushup:103
		io.WriteString(w, "\" ")
//line new.pushup:103
		io.WriteString(w, "id")
//line new.pushup:103
		io.WriteString(w, "=\"")
//line new.pushup:103
		io.WriteString(w, "released")
//line new.pushup:103
		io.WriteString(w, "\" ")
//line new.pushup:103
		io.WriteString(w, "value")
//line new.pushup:103
		io.WriteString(w, "=\"")
//line new.pushup:1
		printEscaped(w, req.FormValue(`released`))
//line new.pushup:103
		io.WriteString(w, "\">")
//line new.pushup:104
		io.WriteString(w, "\n        ")
//line new.pushup:104
		io.WriteString(w, "</div>")
//line new.pushup:105
		io.WriteString(w, "\n        ")
//line new.pushup:105
		io.WriteString(w, "<div ")
//line new.pushup:105
		io.WriteString(w, "class")
//line new.pushup:105
		io.WriteString(w, "=\"")
//line new.pushup:105
		io.WriteString(w, "form-element")
//line new.pushup:105
		io.WriteString(w, "\">")
//line new.pushup:106
		io.WriteString(w, "\n            ")
//line new.pushup:106
		io.WriteString(w, "<label ")
//line new.pushup:106
		io.WriteString(w, "for")
//line new.pushup:106
		io.WriteString(w, "=\"")
//line new.pushup:106
		io.WriteString(w, "length")
//line new.pushup:106
		io.WriteString(w, "\">")
//line new.pushup:106
		io.WriteString(w, "Length (minutes)")
//line new.pushup:106
		io.WriteString(w, "</label>")
//line new.pushup:107
		io.WriteString(w, "\n            ")
//line new.pushup:107
		io.WriteString(w, "<input ")
//line new.pushup:107
		io.WriteString(w, "type")
//line new.pushup:107
		io.WriteString(w, "=\"")
//line new.pushup:107
		io.WriteString(w, "number")
//line new.pushup:107
		io.WriteString(w, "\" ")
//line new.pushup:107
		io.WriteString(w, "name")
//line new.pushup:107
		io.WriteString(w, "=\"")
//line new.pushup:107
		io.WriteString(w, "length")
//line new.pushup:107
		io.WriteString(w, "\" ")
//line new.pushup:107
		io.WriteString(w, "id")
//line new.pushup:107
		io.WriteString(w, "=\"")
//line new.pushup:107
		io.WriteString(w, "length")
//line new.pushup:107
		io.WriteString(w, "\" ")
//line new.pushup:107
		io.WriteString(w, "value")
//line new.pushup:107
		io.WriteString(w, "=\"")
//line new.pushup:1
		printEscaped(w, req.FormValue(`length`))
//line new.pushup:107
		io.WriteString(w, "\">")
//line new.pushup:108
		io.WriteString(w, "\n        ")
//line new.pushup:108
		io.WriteString(w, "</div>")
//line new.pushup:109
		io.WriteString(w, "\n    ")
//line new.pushup:109
		io.WriteString(w, "</section>")
//line new.pushup:110
		io.WriteString(w, "\n\n    ")
//line new.pushup:111
		io.WriteString(w, "<section>")
//line new.pushup:112
		io.WriteString(w, "\n        ")
//line new.pushup:112
		io.WriteString(w, "<div>")
//line new.pushup:113
		io.WriteString(w, "\n            ")
//line new.pushup:113
		io.WriteString(w, "<button ")
//line new.pushup:113
		io.WriteString(w, "type")
//line new.pushup:113
		io.WriteString(w, "=\"")
//line new.pushup:113
		io.WriteString(w, "submit")
//line new.pushup:113
		io.WriteString(w, "\">")
//line new.pushup:113
		io.WriteString(w, "Add album")
//line new.pushup:113
		io.WriteString(w, "</button>")
//line new.pushup:114
		io.WriteString(w, "\n        ")
//line new.pushup:114
		io.WriteString(w, "</div>")
//line new.pushup:115
		io.WriteString(w, "\n    ")
//line new.pushup:115
		io.WriteString(w, "</section>")
//line new.pushup:116
		io.WriteString(w, "\n")
//line new.pushup:116
		io.WriteString(w, "</form>")
//line new.pushup:117
		io.WriteString(w, "\n\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
