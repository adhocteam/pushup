// this file is mechanically generated, do not edit!
package build

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

type Pushup__for__13 struct {
	pushupFilePath string
}

func (t *Pushup__for__13) buildCliArgs() []string {
	return []string{"../pushup", "run", "-build-pkg", "github.com/AdHocRandD/pushup/example/build"}
}

func (t *Pushup__for__13) register() {
	routes.add("/for", t)
}

func init() {
	page := new(Pushup__for__13)
	page.pushupFilePath = "for.pushup"
	page.register()
}

func (t *Pushup__for__13) filePath() string {
	return t.pushupFilePath
}

func (t *Pushup__for__13) Respond(w http.ResponseWriter, req *http.Request) error {
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
//line for.pushup:2
		io.WriteString(w, "\n\n")
//line for.pushup:4
		type product struct {
//line for.pushup:5
			name string
//line for.pushup:6
			price float32
//line for.pushup:7
		}
//line for.pushup:8
		products := []product{
//line for.pushup:9
			{"Big Kahuna Burger", 3.99},
//line for.pushup:10
			{"Everlasting Gobstopper", 0.25},
//line for.pushup:11
			{"Nike Air Mags", 249.99},
//line for.pushup:12
		}
//line for.pushup:13

//line for.pushup:14
		io.WriteString(w, "\n\n")
//line for.pushup:15
		io.WriteString(w, "<ul>")
//line for.pushup:16
		io.WriteString(w, "\n")
		for _, p := range products {
//line for.pushup:17
			io.WriteString(w, "\n    ")
//line for.pushup:17
//line for.pushup:17
			io.WriteString(w, "<li>")
//line for.pushup:17
			printEscaped(w, p.name)
//line for.pushup:17
			io.WriteString(w, " ($")
//line for.pushup:17
			printEscaped(w, fmt.Sprintf("%.2f", p.price))
//line for.pushup:17
			io.WriteString(w, ")")
			io.WriteString(w, "</li>")
		}
//line for.pushup:19
		io.WriteString(w, "\n")
//line for.pushup:19
		io.WriteString(w, "</ul>")
//line for.pushup:20
		io.WriteString(w, "\n")

		if renderLayout {
			yield <- struct{}{}
			wg.Wait()
		}
		// End user Go code and HTML
	}
	return nil
}
