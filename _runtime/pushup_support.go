package build

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

// FIXME(paulsmith): I think of this as a route but this conflicts with a route in the serve mux
// sense, so calling "component" for now
type component interface {
	// FIXME(paulsmith): return a pushup.Response object instead and don't take a writer
	Render(io.Writer, *http.Request) error
}

// FIXME(paulsmith): add a wrapper type for easily going between a component and a http.Handler

type routeList []route

var routes routeList

func (r *routeList) add(path string, c component) {
	*r = append(*r, newRoute(path, c))
}

type route struct {
	path      string
	regex     *regexp.Regexp
	component component
}

func newRoute(path string, c component) route {
	return route{path, regexp.MustCompile("^" + path + "$"), c}
}

var NotFound = errors.New("page not found")

func Render(w http.ResponseWriter, r *http.Request) error {
	for _, route := range routes {
		matches := route.regex.FindStringSubmatch(r.URL.Path)
		if len(matches) > 0 {
			// TODO(paulsmith): implement matches
			if err := route.component.Render(w, r); err != nil {
				return err
			}
			return nil
		}
	}
	return NotFound
}

type layout interface {
	Render(yield chan struct{}, w io.Writer, req *http.Request) error
}

var layouts = make(map[string]layout)

func getLayout(name string) layout {
	l, ok := layouts[name]
	if !ok {
		panic("couldn't find layout " + name)
	}
	return l
}

func Admin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<h1>Routes</h1>\n<ul>\n")
	for _, route := range routes {
		fmt.Fprintf(w, "\t<li>%s</li>\n", route.path)
	}
	fmt.Fprintf(w, "</ul>\n")
}
