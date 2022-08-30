package build

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
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
	routeMatch := getRouteFromPath(r.URL.Path)
	switch routeMatch.response {
	case routeNotFound:
		return NotFound
	case redirectTrailingSlash:
		http.Redirect(w, r, routeMatch.route.path, 301)
		return nil
	case routeFound:
		route := routeMatch.route
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
	default:
		panic("unhandled route match response")
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

type routeMatchResponse int

const (
	routeNotFound routeMatchResponse = iota
	redirectTrailingSlash
	routeFound
)

type routeMatch struct {
	response routeMatchResponse
	route    *route
}

func getRouteFromPath(path string) routeMatch {
	var matchedRoutes []*route

	for _, r := range routes {
		if r.regex.MatchString(path) {
			matchedRoutes = append(matchedRoutes, r)
		}
	}

	if len(matchedRoutes) == 0 {
		// check trailing slash
		if path[len(path)-1] == '/' {
			lessSlash := path[:len(path)-1]
			for _, r := range routes {
				if r.regex.MatchString(lessSlash) {
					return routeMatch{
						response: redirectTrailingSlash,
						route:    &route{path: lessSlash},
					}
				}
			}
		}
	}

	if len(matchedRoutes) == 0 {
		return routeMatch{response: routeNotFound, route: nil}
	}

	return routeMatch{response: routeFound, route: mostSpecificMatch(matchedRoutes, path)}
}

func getParam(r *http.Request, slug string) string {
	params := r.Context().Value(ctxKey{}).(map[string]string)
	return params[slug]
}

type layout interface {
	Respond(w http.ResponseWriter, req *http.Request, sections map[string]chan template.HTML) error
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
	case template.HTML:
		io.WriteString(w, string(val))
	default:
		io.WriteString(w, template.HTMLEscapeString(fmt.Sprint(val)))
	}
}

//{{if .EmbedStatic}}
//go:embed static{{end}}
var static embed.FS

func AddStaticHandler(mux *http.ServeMux) {
	fsys, err := fs.Sub(static, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(fsys))))
}

//go:embed src
var source embed.FS

// GetPageSource gets the source code of the Pushup page at the path. Assumes
// path is relative to the app/pages project directory.
func GetPageSource(path string) []byte {
	fsys, err := fs.Sub(source, filepath.Join("src", "pages"))
	if err != nil {
		panic(err)
	}
	data, err := fsys.(fs.ReadFileFS).ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}

// Inline partials

func isPartialRoute(mainRoute string, path string) bool {
	if path == mainRoute {
		return false
	} else if strings.HasPrefix(path, mainRoute) {
		return true
	} else {
		panic("internal error: unexpected path")
	}
}

func displayPartialHere(partialPath string, path string) bool {
	if strings.HasPrefix(partialPath, path) {
		return true
	}
	return false
}
