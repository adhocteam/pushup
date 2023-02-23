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

type Responder interface {
	// TODO(paulsmith): return a pushup.Response object instead and don't take
	// a writer
	Respond(http.ResponseWriter, *http.Request) error
}

// FIXME(paulsmith): add a wrapper type for easily going between a component and a http.Handler
// TODO(paulsmith): HTTP methods? how to handle? right now, not dealt with at route level

// NOTE(paulsmith): routing inspired by https://benhoyt.com/writings/go-routing/

type routeList []*route

var routes routeList

type routeRole int

const (
	routePage routeRole = iota
	routePartial
)

func (r *routeList) add(path string, responder Responder, role routeRole) {
	*r = append(*r, newRoute(path, responder, role))
}

type route struct {
	path      string
	regex     *regexp.Regexp
	slugs     []string
	responder Responder
	role      routeRole
}

func newRoute(path string, responder Responder, role routeRole) *route {
	p := regexPatFromRoute(path)
	result := new(route)
	result.path = path
	result.regex = regexp.MustCompile("^" + p.pat + "$")
	result.slugs = p.slugs
	result.responder = responder
	result.role = role
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

var ErrNotFound = errors.New("page not found")

type ctxKey struct{}

func Respond(w http.ResponseWriter, r *http.Request) error {
	routeMatch := getRouteFromPath(r.URL.Path)
	switch routeMatch.response {
	case routeNotFound:
		return ErrNotFound
	case redirectTrailingSlash:
		http.Redirect(w, r, routeMatch.route.path, http.StatusMovedPermanently)
		return nil
	case routeFound:
		route := routeMatch.route
		matches := route.regex.FindStringSubmatch(r.URL.Path)
		params := zipMap(route.slugs, matches[1:])
		if route.role == routePartial {
			w.Header().Set("Pushup-Partial", "true")
		}
		// NOTE(paulsmith): since we totally control the Respond() method on
		// the component interface, we probably should pass the params to
		// Respond instead of wrapping the request object with context values.
		ctx := context.WithValue(r.Context(), ctxKey{}, params)
		if err := route.responder.Respond(w, r.WithContext(ctx)); err != nil {
			return err
		}
		return nil
	default:
		panic("unhandled route match response")
	}
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
		//nolint:errcheck
		io.WriteString(w, template.HTMLEscapeString(val))
	case fmt.Stringer:
		//nolint:errcheck
		io.WriteString(w, template.HTMLEscapeString(val.String()))
	case []byte:
		template.HTMLEscape(w, val)
	case int:
		//nolint:errcheck
		io.WriteString(w, strconv.Itoa(val))
	case template.HTML:
		//nolint:errcheck
		io.WriteString(w, string(val))
	default:
		//nolint:errcheck
		io.WriteString(w, template.HTMLEscapeString(fmt.Sprint(val)))
	}
}

// {{if .EmbedStatic}}
//
//go:embed static{{end}}
var static embed.FS

func AddStaticHandler(mux *http.ServeMux) {
	fsys, err := fs.Sub(static, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(fsys))))
}

// GetStaticContents gets the contents of a static file at the path.
func GetStaticContents(path string) []byte {
	fsys, err := fs.Sub(static, "static")
	if err != nil {
		panic(err)
	}
	data, err := fsys.(fs.ReadFileFS).ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
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

func isPartialRoute(mainRoute string, requestPath string) bool {
	match := getRouteFromPath(requestPath)
	if match.response == routeFound {
		route := match.route
		return route.path != mainRoute
	}
	panic("internal error: unexpected path")
}

func displayPartialHere(mainRoute string, partialPath string, requestPath string) bool {
	var path string
	if mainRoute[len(mainRoute)-1] != '/' {
		path = mainRoute + "/" + partialPath
	} else {
		path = mainRoute + partialPath
	}
	match := getRouteFromPath(path)
	if match.response == routeFound {
		return matchURLPathSegmentPrefix(match.route.regex, requestPath)
	}
	panic("internal error: unexpected path")
}

// matchURLPathSegmentPrefix reports whether a string in the form of a URL
// path matches as a prefix of a regex that is potentially shorter (in terms
// of number of URL path segments) than the string.
func matchURLPathSegmentPrefix(re *regexp.Regexp, s string) bool {
	res := re.String()
	// strip off matching start of string
	if res[0] == '^' {
		res = res[1:]
	}
	// strip off matching end of string
	if res[len(res)-1] == '$' {
		res = res[:len(res)-1]
	}
	var reSegments []string
	var state int
	const (
		stateStart int = iota
		stateInCapture
	)
	var accum []rune
	for _, r := range res {
		switch state {
		case stateStart:
			if r == '/' {
				if len(accum) > 0 {
					reSegments = append(reSegments, string(accum))
					accum = accum[:0]
				}
			} else if r == '(' {
				state = stateInCapture
			} else {
				accum = append(accum, r)
			}
		case stateInCapture:
			if r == ')' {
				if len(accum) > 0 {
					reSegments = append(reSegments, string(accum))
					accum = accum[:0]
					state = stateStart
				}
			} else {
				accum = append(accum, r)
			}
		default:
			panic("unhandled state")
		}
	}
	if len(accum) > 0 {
		reSegments = append(reSegments, string(accum))
	}
	var segments []string
	s = strings.Trim(s, "/")
	if s != "" {
		segments = strings.Split(s, "/")
	}
	for i := 0; i < min(len(reSegments), len(segments)); i++ {
		reseg := reSegments[i]
		seg := segments[i]
		if !regexp.MustCompile(reseg).MatchString(seg) {
			return false
		}
	}
	return len(segments) <= len(reSegments)
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
