package build

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type page interface {
	// FIXME(paulsmith): return a pushup.Response object instead and don't take a writer
	Render(io.Writer, *http.Request) error
}

// FIXME(paulsmith): add a wrapper type for easily going between a component and a http.Handler
// TODO(paulsmith): HTTP methods? how to handle? right now, not dealt with at route level

// NOTE(paulsmith): routing inspired by https://benhoyt.com/writings/go-routing/

type routeList []route

var routes routeList

func (r *routeList) add(path string, c page) {
	*r = append(*r, newRoute(path, c))
}

type route struct {
	path  string
	regex *regexp.Regexp
	slugs []string
	page  page
}

func newRoute(path string, c page) route {
	p := regexPatFromRoute(path)
	var result route
	result.path = path
	result.regex = regexp.MustCompile("^" + p.pat + "$")
	result.slugs = p.slugs
	result.page = c
	return result
}

type routePat struct {
	pat   string
	slugs []string
}

// regexPatFromRoute produces a regular expression from a route string,
// replacing slugs with capture groups and retaining the slugs so that HTTP
// handlers can retrieve paramaters by slug name.
func regexPatFromRoute(route string) routePat {
	const match = "([^/]+)"
	pathsubs := strings.Split(route, "/")
	var out []string
	var slugs []string
	for _, sub := range pathsubs {
		if strings.HasPrefix(sub, ":") {
			out = append(out, match)
			slugs = append(slugs, sub[1:])
		} else {
			out = append(out, sub)
		}
	}
	return routePat{strings.Join(out, "/"), slugs}
}

var NotFound = errors.New("page not found")

type ctxKey struct{}

func Render(w http.ResponseWriter, r *http.Request) error {
	for _, route := range routes {
		matches := route.regex.FindStringSubmatch(r.URL.Path)
		if len(matches) > 0 {
			params := zipMap(route.slugs, matches[1:])
			// NOTE(paulsmith): since we totally control the Render() method on
			// the component interface, we probably should pass the params to
			// Render instead of wrapping the request object with context values.
			ctx := context.WithValue(r.Context(), ctxKey{}, params)
			if err := route.page.Render(w, r.WithContext(ctx)); err != nil {
				return err
			}
			return nil
		}
	}
	return NotFound
}

func getParam(r *http.Request, slug string) string {
	params := r.Context().Value(ctxKey{}).(map[string]string)
	return params[slug]
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

func zipMap[K comparable, V any](ks []K, vs []V) map[K]V {
	m := make(map[K]V)
	for i := range ks {
		m[ks[i]] = vs[i]
	}
	return m
}
