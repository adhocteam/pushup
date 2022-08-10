// this file is mechanically generated, do not edit!
package build

import (
	"io"
	"log"
	"net/http"
)

type Pushup__models__18 struct {
	pushupFilePath string
}

func (t *Pushup__models__18) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__models__18) register() {
	routes.add("/htmx/models", t)
}

func init() {
	page := new(Pushup__models__18)
	page.pushupFilePath = "htmx/models.pushup"
	page.register()
}

func (t *Pushup__models__18) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__models__18) Respond(w http.ResponseWriter, req *http.Request) error {
	// Begin user Go code and HTML
	{
//line models.pushup:4
		io.WriteString(w, "\n\n")
//line models.pushup:6
		type model struct {
//line models.pushup:7
			value string
//line models.pushup:8
			label string
//line models.pushup:9
		}
//line models.pushup:10
		makesAndModels := map[string][]model{
//line models.pushup:11
			"Apple silicon": []model{
//line models.pushup:12
				{value: "M1", label: "M1"},
//line models.pushup:13
				{value: "M2", label: "M2"},
//line models.pushup:14
			},
//line models.pushup:15
			"Intel": []model{
//line models.pushup:16
				{value: "i3", label: "Core i3"},
//line models.pushup:17
				{value: "i5", label: "Core i5"},
//line models.pushup:18
				{value: "i7", label: "Core i7"},
//line models.pushup:19
			},
//line models.pushup:20
			"AMD": []model{
//line models.pushup:21
				{value: "ryzen5", label: "Ryzen 5"},
//line models.pushup:22
				{value: "ryzen7", label: "Ryzen 7"},
//line models.pushup:23
				{value: "ryzen9", label: "Ryzen 9"},
//line models.pushup:24
			},
//line models.pushup:25
		}
//line models.pushup:26
		var models []model
//line models.pushup:27
		make := req.FormValue("make")
//line models.pushup:28
		log.Printf("make: %v", make)
//line models.pushup:29
		if m, ok := makesAndModels[make]; ok {
//line models.pushup:30
			models = m
//line models.pushup:31
		}
//line models.pushup:32

//line models.pushup:33
		io.WriteString(w, "\n\n")
		for _, model := range models {
//line models.pushup:35
			io.WriteString(w, "\n    ")
//line models.pushup:35
//line models.pushup:35
			io.WriteString(w, "<option ")
//line models.pushup:35
			io.WriteString(w, "value")
//line models.pushup:35
			io.WriteString(w, "=\"")
//line models.pushup:1
			printEscaped(w, model.value)
//line models.pushup:35
			io.WriteString(w, "\">")
//line models.pushup:35
			printEscaped(w, model.label)
			io.WriteString(w, "</option value=\"^model.value\">")
		}
//line models.pushup:37
		io.WriteString(w, "\n")
		// End user Go code and HTML
	}
	return nil
}
