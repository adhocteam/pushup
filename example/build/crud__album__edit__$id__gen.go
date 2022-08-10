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

type Pushup__DollarSign_id__7 struct {
	pushupFilePath string
}

func (t *Pushup__DollarSign_id__7) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__DollarSign_id__7) register() {
	routes.add("/crud/album/edit/:id", t)
}

func init() {
	page := new(Pushup__DollarSign_id__7)
	page.pushupFilePath = "crud/album/edit/$id.pushup"
	page.register()
}

func (t *Pushup__DollarSign_id__7) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__DollarSign_id__7) Respond(w http.ResponseWriter, req *http.Request) error {
	renderLayout := true
//line $id.pushup:7
	id, err := strconv.Atoi(getParam(req, "id"))
//line $id.pushup:8
	if err != nil {
//line $id.pushup:9
		return err
//line $id.pushup:10
	}
//line $id.pushup:11

//line $id.pushup:12
	album, err := getAlbumById(DB, id)
//line $id.pushup:13
	if err != nil {
//line $id.pushup:14
		return err
//line $id.pushup:15
	}
//line $id.pushup:16

//line $id.pushup:17
	isNumber := func(val string) (int, bool) {
//line $id.pushup:18
		n, err := strconv.Atoi(val)
//line $id.pushup:19
		if err != nil {
//line $id.pushup:20
			return 0, false
//line $id.pushup:21
		}
//line $id.pushup:22
		return n, true
//line $id.pushup:23
	}
//line $id.pushup:24

//line $id.pushup:25
	errors := make(map[string]string)
//line $id.pushup:26

//line $id.pushup:27
	if req.Method == "POST" {
//line $id.pushup:28
		album.artist = strings.TrimSpace(req.FormValue("artist"))
//line $id.pushup:29
		album.title = strings.TrimSpace(req.FormValue("title"))
//line $id.pushup:30
		releasedRaw := strings.TrimSpace(req.FormValue("released"))
//line $id.pushup:31
		lengthRaw := strings.TrimSpace(req.FormValue("length"))
//line $id.pushup:32

//line $id.pushup:33
		if album.artist == "" {
//line $id.pushup:34
			errors["artist"] = "artist name is required"
//line $id.pushup:35
		}
//line $id.pushup:36
		if album.title == "" {
//line $id.pushup:37
			errors["title"] = "title is required"
//line $id.pushup:38
		}
//line $id.pushup:39
		var releasedIsNum bool
//line $id.pushup:40
		album.released, releasedIsNum = isNumber(releasedRaw)
//line $id.pushup:41
		if releasedRaw == "" {
//line $id.pushup:42
			errors["released"] = "release year is required"
//line $id.pushup:43
		} else if !releasedIsNum || !(album.released >= 1900 && album.released <= time.Now().Year()) {
//line $id.pushup:44
			errors["released"] = "release year must be between 1900 and this year"
//line $id.pushup:45
		}
//line $id.pushup:46
		var lengthIsNum bool
//line $id.pushup:47
		album.length, lengthIsNum = isNumber(lengthRaw)
//line $id.pushup:48
		if lengthRaw == "" {
//line $id.pushup:49
			errors["length"] = "length is required"
//line $id.pushup:50
		} else if !lengthIsNum || !(album.length > 0) {
//line $id.pushup:51
			errors["length"] = "length must be a number greater than 0"
//line $id.pushup:52
		}
//line $id.pushup:53

//line $id.pushup:54
		if len(errors) == 0 {
//line $id.pushup:55
			if err := editAlbum(DB, id, album); err != nil {
//line $id.pushup:56
				log.Printf("error: %v", err)
//line $id.pushup:57
				http.Error(w, http.StatusText(500), 500)
//line $id.pushup:58
				return nil
//line $id.pushup:59
			}
//line $id.pushup:60
			http.Redirect(w, req, "/crud", 301)
//line $id.pushup:61
		}
//line $id.pushup:62
	}
//line $id.pushup:63

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
		io.WriteString(w, "\n")
//line $id.pushup:3
		io.WriteString(w, "\n")
//line $id.pushup:4
		io.WriteString(w, "\n")
//line $id.pushup:5
		io.WriteString(w, "\n\n")
//line $id.pushup:64
		io.WriteString(w, "\n\n")
//line $id.pushup:65
		io.WriteString(w, "<h1>")
//line $id.pushup:65
		io.WriteString(w, "Edit ")
//line $id.pushup:65
		printEscaped(w, album.title)
//line $id.pushup:65
		io.WriteString(w, "</h1>")
//line $id.pushup:66
		io.WriteString(w, "\n\n")
//line $id.pushup:67
		io.WriteString(w, "<p>")
//line $id.pushup:67
		io.WriteString(w, "<a ")
//line $id.pushup:67
		io.WriteString(w, "href")
//line $id.pushup:67
		io.WriteString(w, "=\"")
//line $id.pushup:67
		io.WriteString(w, "/crud/album/")
//line $id.pushup:1
		printEscaped(w, album.id)
//line $id.pushup:67
		io.WriteString(w, "\">")
//line $id.pushup:67
		io.WriteString(w, "Cancel")
//line $id.pushup:67
		io.WriteString(w, "</a>")
//line $id.pushup:67
		io.WriteString(w, "</p>")
//line $id.pushup:68
		io.WriteString(w, "\n\n")
//line $id.pushup:69
		io.WriteString(w, "<style>")
//line $id.pushup:70
		io.WriteString(w, "\nform { border: 1px solid #ddd; background: #f3f4f0; padding: 2rem; }\nform h2, form h3 { margin: 0 0 0.5rem 0; }\nform section { margin: 0 0 1rem 0; }\n.form-element { display: flex; justify-content: align-items: center; flex-end; padding: 1em 0; }\n.form-element > label { flex: 1; }\n.form-element > input { flex: 2; padding: 0.5rem; }\nform button { font-size: 1.5rem; padding: 1rem; }\n")
//line $id.pushup:77
		io.WriteString(w, "</style>")
//line $id.pushup:78
		io.WriteString(w, "\n\n")
//line $id.pushup:79
		io.WriteString(w, "<form ")
//line $id.pushup:79
		io.WriteString(w, "method")
//line $id.pushup:79
		io.WriteString(w, "=\"")
//line $id.pushup:79
		io.WriteString(w, "post")
//line $id.pushup:79
		io.WriteString(w, "\">")
//line $id.pushup:80
		io.WriteString(w, "\n    ")
		if len(errors) > 0 {
//line $id.pushup:81
			io.WriteString(w, "\n        ")
//line $id.pushup:81
//line $id.pushup:81
			io.WriteString(w, "<section ")
//line $id.pushup:81
			io.WriteString(w, "style")
//line $id.pushup:81
			io.WriteString(w, "=\"")
//line $id.pushup:81
			io.WriteString(w, "color: red")
//line $id.pushup:81
			io.WriteString(w, "\">")
//line $id.pushup:82
			io.WriteString(w, "\n            ")
//line $id.pushup:82
//line $id.pushup:82
			io.WriteString(w, "<h3>")
//line $id.pushup:82
			io.WriteString(w, "Errors")
			io.WriteString(w, "</h3>")
//line $id.pushup:83
			io.WriteString(w, "\n            ")
//line $id.pushup:83
//line $id.pushup:83
			io.WriteString(w, "<ul>")
//line $id.pushup:84
			io.WriteString(w, "\n                ")
			for _, message := range errors {
//line $id.pushup:85
				io.WriteString(w, "\n                    ")
//line $id.pushup:85
//line $id.pushup:85
				io.WriteString(w, "<li>")
//line $id.pushup:85
				printEscaped(w, message)
				io.WriteString(w, "</li>")
			}
//line $id.pushup:87
			io.WriteString(w, "\n            ")
			io.WriteString(w, "</ul>")
//line $id.pushup:88
			io.WriteString(w, "\n        ")
			io.WriteString(w, "</section style=\"color: red\">")
		}
//line $id.pushup:90
		io.WriteString(w, "\n    ")
//line $id.pushup:90
		io.WriteString(w, "<section>")
//line $id.pushup:91
		io.WriteString(w, "\n        ")
//line $id.pushup:91
		io.WriteString(w, "<h2>")
//line $id.pushup:91
		io.WriteString(w, "Artist")
//line $id.pushup:91
		io.WriteString(w, "</h2>")
//line $id.pushup:92
		io.WriteString(w, "\n        ")
//line $id.pushup:92
		io.WriteString(w, "<div ")
//line $id.pushup:92
		io.WriteString(w, "class")
//line $id.pushup:92
		io.WriteString(w, "=\"")
//line $id.pushup:92
		io.WriteString(w, "form-element")
//line $id.pushup:92
		io.WriteString(w, "\">")
//line $id.pushup:93
		io.WriteString(w, "\n            ")
//line $id.pushup:93
		io.WriteString(w, "<label ")
//line $id.pushup:93
		io.WriteString(w, "for")
//line $id.pushup:93
		io.WriteString(w, "=\"")
//line $id.pushup:93
		io.WriteString(w, "artist")
//line $id.pushup:93
		io.WriteString(w, "\">")
//line $id.pushup:93
		io.WriteString(w, "Artist name")
//line $id.pushup:93
		io.WriteString(w, "</label>")
//line $id.pushup:94
		io.WriteString(w, "\n            ")
//line $id.pushup:94
		io.WriteString(w, "<input ")
//line $id.pushup:94
		io.WriteString(w, "type")
//line $id.pushup:94
		io.WriteString(w, "=\"")
//line $id.pushup:94
		io.WriteString(w, "text")
//line $id.pushup:94
		io.WriteString(w, "\" ")
//line $id.pushup:94
		io.WriteString(w, "name")
//line $id.pushup:94
		io.WriteString(w, "=\"")
//line $id.pushup:94
		io.WriteString(w, "artist")
//line $id.pushup:94
		io.WriteString(w, "\" ")
//line $id.pushup:94
		io.WriteString(w, "id")
//line $id.pushup:94
		io.WriteString(w, "=\"")
//line $id.pushup:94
		io.WriteString(w, "artist")
//line $id.pushup:94
		io.WriteString(w, "\" ")
//line $id.pushup:94
		io.WriteString(w, "value")
//line $id.pushup:94
		io.WriteString(w, "=\"")
//line $id.pushup:1
		printEscaped(w, album.artist)
//line $id.pushup:94
		io.WriteString(w, "\">")
//line $id.pushup:95
		io.WriteString(w, "\n        ")
//line $id.pushup:95
		io.WriteString(w, "</div>")
//line $id.pushup:96
		io.WriteString(w, "\n    ")
//line $id.pushup:96
		io.WriteString(w, "</section>")
//line $id.pushup:97
		io.WriteString(w, "\n\n    ")
//line $id.pushup:98
		io.WriteString(w, "<section>")
//line $id.pushup:99
		io.WriteString(w, "\n        ")
//line $id.pushup:99
		io.WriteString(w, "<h2>")
//line $id.pushup:99
		io.WriteString(w, "Album info")
//line $id.pushup:99
		io.WriteString(w, "</h2>")
//line $id.pushup:100
		io.WriteString(w, "\n        ")
//line $id.pushup:100
		io.WriteString(w, "<div ")
//line $id.pushup:100
		io.WriteString(w, "class")
//line $id.pushup:100
		io.WriteString(w, "=\"")
//line $id.pushup:100
		io.WriteString(w, "form-element")
//line $id.pushup:100
		io.WriteString(w, "\">")
//line $id.pushup:101
		io.WriteString(w, "\n            ")
//line $id.pushup:101
		io.WriteString(w, "<label ")
//line $id.pushup:101
		io.WriteString(w, "for")
//line $id.pushup:101
		io.WriteString(w, "=\"")
//line $id.pushup:101
		io.WriteString(w, "title")
//line $id.pushup:101
		io.WriteString(w, "\">")
//line $id.pushup:101
		io.WriteString(w, "Title")
//line $id.pushup:101
		io.WriteString(w, "</label>")
//line $id.pushup:102
		io.WriteString(w, "\n            ")
//line $id.pushup:102
		io.WriteString(w, "<input ")
//line $id.pushup:102
		io.WriteString(w, "type")
//line $id.pushup:102
		io.WriteString(w, "=\"")
//line $id.pushup:102
		io.WriteString(w, "text")
//line $id.pushup:102
		io.WriteString(w, "\" ")
//line $id.pushup:102
		io.WriteString(w, "name")
//line $id.pushup:102
		io.WriteString(w, "=\"")
//line $id.pushup:102
		io.WriteString(w, "title")
//line $id.pushup:102
		io.WriteString(w, "\" ")
//line $id.pushup:102
		io.WriteString(w, "id")
//line $id.pushup:102
		io.WriteString(w, "=\"")
//line $id.pushup:102
		io.WriteString(w, "title")
//line $id.pushup:102
		io.WriteString(w, "\" ")
//line $id.pushup:102
		io.WriteString(w, "value")
//line $id.pushup:102
		io.WriteString(w, "=\"")
//line $id.pushup:1
		printEscaped(w, album.title)
//line $id.pushup:102
		io.WriteString(w, "\">")
//line $id.pushup:103
		io.WriteString(w, "\n        ")
//line $id.pushup:103
		io.WriteString(w, "</div>")
//line $id.pushup:104
		io.WriteString(w, "\n        ")
//line $id.pushup:104
		io.WriteString(w, "<div ")
//line $id.pushup:104
		io.WriteString(w, "class")
//line $id.pushup:104
		io.WriteString(w, "=\"")
//line $id.pushup:104
		io.WriteString(w, "form-element")
//line $id.pushup:104
		io.WriteString(w, "\">")
//line $id.pushup:105
		io.WriteString(w, "\n            ")
//line $id.pushup:105
		io.WriteString(w, "<label ")
//line $id.pushup:105
		io.WriteString(w, "for")
//line $id.pushup:105
		io.WriteString(w, "=\"")
//line $id.pushup:105
		io.WriteString(w, "released")
//line $id.pushup:105
		io.WriteString(w, "\">")
//line $id.pushup:105
		io.WriteString(w, "Year released")
//line $id.pushup:105
		io.WriteString(w, "</label>")
//line $id.pushup:106
		io.WriteString(w, "\n            ")
//line $id.pushup:106
		io.WriteString(w, "<input ")
//line $id.pushup:106
		io.WriteString(w, "type")
//line $id.pushup:106
		io.WriteString(w, "=\"")
//line $id.pushup:106
		io.WriteString(w, "text")
//line $id.pushup:106
		io.WriteString(w, "\" ")
//line $id.pushup:106
		io.WriteString(w, "name")
//line $id.pushup:106
		io.WriteString(w, "=\"")
//line $id.pushup:106
		io.WriteString(w, "released")
//line $id.pushup:106
		io.WriteString(w, "\" ")
//line $id.pushup:106
		io.WriteString(w, "id")
//line $id.pushup:106
		io.WriteString(w, "=\"")
//line $id.pushup:106
		io.WriteString(w, "released")
//line $id.pushup:106
		io.WriteString(w, "\" ")
//line $id.pushup:106
		io.WriteString(w, "value")
//line $id.pushup:106
		io.WriteString(w, "=\"")
//line $id.pushup:1
		printEscaped(w, album.released)
//line $id.pushup:106
		io.WriteString(w, "\">")
//line $id.pushup:107
		io.WriteString(w, "\n        ")
//line $id.pushup:107
		io.WriteString(w, "</div>")
//line $id.pushup:108
		io.WriteString(w, "\n        ")
//line $id.pushup:108
		io.WriteString(w, "<div ")
//line $id.pushup:108
		io.WriteString(w, "class")
//line $id.pushup:108
		io.WriteString(w, "=\"")
//line $id.pushup:108
		io.WriteString(w, "form-element")
//line $id.pushup:108
		io.WriteString(w, "\">")
//line $id.pushup:109
		io.WriteString(w, "\n            ")
//line $id.pushup:109
		io.WriteString(w, "<label ")
//line $id.pushup:109
		io.WriteString(w, "for")
//line $id.pushup:109
		io.WriteString(w, "=\"")
//line $id.pushup:109
		io.WriteString(w, "length")
//line $id.pushup:109
		io.WriteString(w, "\">")
//line $id.pushup:109
		io.WriteString(w, "Length (minutes)")
//line $id.pushup:109
		io.WriteString(w, "</label>")
//line $id.pushup:110
		io.WriteString(w, "\n            ")
//line $id.pushup:110
		io.WriteString(w, "<input ")
//line $id.pushup:110
		io.WriteString(w, "type")
//line $id.pushup:110
		io.WriteString(w, "=\"")
//line $id.pushup:110
		io.WriteString(w, "number")
//line $id.pushup:110
		io.WriteString(w, "\" ")
//line $id.pushup:110
		io.WriteString(w, "name")
//line $id.pushup:110
		io.WriteString(w, "=\"")
//line $id.pushup:110
		io.WriteString(w, "length")
//line $id.pushup:110
		io.WriteString(w, "\" ")
//line $id.pushup:110
		io.WriteString(w, "id")
//line $id.pushup:110
		io.WriteString(w, "=\"")
//line $id.pushup:110
		io.WriteString(w, "length")
//line $id.pushup:110
		io.WriteString(w, "\" ")
//line $id.pushup:110
		io.WriteString(w, "value")
//line $id.pushup:110
		io.WriteString(w, "=\"")
//line $id.pushup:1
		printEscaped(w, album.length)
//line $id.pushup:110
		io.WriteString(w, "\">")
//line $id.pushup:111
		io.WriteString(w, "\n        ")
//line $id.pushup:111
		io.WriteString(w, "</div>")
//line $id.pushup:112
		io.WriteString(w, "\n    ")
//line $id.pushup:112
		io.WriteString(w, "</section>")
//line $id.pushup:113
		io.WriteString(w, "\n\n    ")
//line $id.pushup:114
		io.WriteString(w, "<section>")
//line $id.pushup:115
		io.WriteString(w, "\n        ")
//line $id.pushup:115
		io.WriteString(w, "<div>")
//line $id.pushup:116
		io.WriteString(w, "\n            ")
//line $id.pushup:116
		io.WriteString(w, "<button ")
//line $id.pushup:116
		io.WriteString(w, "type")
//line $id.pushup:116
		io.WriteString(w, "=\"")
//line $id.pushup:116
		io.WriteString(w, "submit")
//line $id.pushup:116
		io.WriteString(w, "\">")
//line $id.pushup:116
		io.WriteString(w, "Update album")
//line $id.pushup:116
		io.WriteString(w, "</button>")
//line $id.pushup:117
		io.WriteString(w, "\n            ")
//line $id.pushup:117
		io.WriteString(w, "<input ")
//line $id.pushup:117
		io.WriteString(w, "type")
//line $id.pushup:117
		io.WriteString(w, "=\"")
//line $id.pushup:117
		io.WriteString(w, "hidden")
//line $id.pushup:117
		io.WriteString(w, "\" ")
//line $id.pushup:117
		io.WriteString(w, "name")
//line $id.pushup:117
		io.WriteString(w, "=\"")
//line $id.pushup:117
		io.WriteString(w, "id")
//line $id.pushup:117
		io.WriteString(w, "\" ")
//line $id.pushup:117
		io.WriteString(w, "value")
//line $id.pushup:117
		io.WriteString(w, "=\"")
//line $id.pushup:1
		printEscaped(w, album.id)
//line $id.pushup:117
		io.WriteString(w, "\">")
//line $id.pushup:118
		io.WriteString(w, "\n        ")
//line $id.pushup:118
		io.WriteString(w, "</div>")
//line $id.pushup:119
		io.WriteString(w, "\n    ")
//line $id.pushup:119
		io.WriteString(w, "</section>")
//line $id.pushup:120
		io.WriteString(w, "\n")
//line $id.pushup:120
		io.WriteString(w, "</form>")
//line $id.pushup:121
		io.WriteString(w, "\n\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
