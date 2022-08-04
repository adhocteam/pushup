package build

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type page interface {
	// FIXME(paulsmith): return a pushup.Response object instead and don't take a writer
	Respond(http.ResponseWriter, *http.Request) error
	filePath() string
}

// FIXME(paulsmith): add a wrapper type for easily going between a component and a http.Handler
// TODO(paulsmith): HTTP methods? how to handle? right now, not dealt with at route level

// NOTE(paulsmith): routing inspired by https://benhoyt.com/writings/go-routing/

type routeList []*route

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

func newRoute(path string, c page) *route {
	p := regexPatFromRoute(path)
	result := new(route)
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

func Respond(w http.ResponseWriter, r *http.Request) error {
	route := getRouteFromPath(r.URL.Path)
	if route == nil {
		return NotFound
	}
	matches := route.regex.FindStringSubmatch(r.URL.Path)
	params := zipMap(route.slugs, matches[1:])
	// NOTE(paulsmith): since we totally control the Respond() method on
	// the component interface, we probably should pass the params to
	// Respond instead of wrapping the request object with context values.
	ctx := context.WithValue(r.Context(), ctxKey{}, params)
	if err := route.page.Respond(w, r.WithContext(ctx)); err != nil {
		return err
	}

	return nil
}

func mostSpecificMatch(routes []*route, path string) *route {
	if len(routes) == 1 {
		return routes[0]
	}

	most := routes[0]

	for _, route := range routes[1:] {
		if len(route.slugs) < len(most.slugs) {
			most = route
		}
	}

	return most
}

func getRouteFromPath(path string) *route {
	var matchedRoutes []*route

	for _, route := range routes {
		if route.regex.MatchString(path) {
			matchedRoutes = append(matchedRoutes, route)
		}
	}

	if len(matchedRoutes) == 0 {
		return nil
	}

	return mostSpecificMatch(matchedRoutes, path)
}

func getParam(r *http.Request, slug string) string {
	params := r.Context().Value(ctxKey{}).(map[string]string)
	return params[slug]
}

type layout interface {
	Respond(yield chan struct{}, w http.ResponseWriter, req *http.Request) error
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

func printEscaped(w io.Writer, val any) {
	switch val := val.(type) {
	case string:
		io.WriteString(w, template.HTMLEscapeString(val))
	case fmt.Stringer:
		io.WriteString(w, template.HTMLEscapeString(val.String()))
	case []byte:
		template.HTMLEscape(w, val)
	case int:
		io.WriteString(w, strconv.Itoa(val))
	default:
		io.WriteString(w, template.HTMLEscapeString(fmt.Sprint(val)))
	}
}
