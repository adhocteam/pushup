package api

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
)

type Responder interface {
	Respond(http.ResponseWriter, *http.Request) error
}

type RouteRole int

const (
	RoutePage RouteRole = iota
	RoutePartial
)

type route struct {
	path      string
	regex     *regexp.Regexp
	slugs     []string
	responder Responder
	role      RouteRole
}

type Routes []*route

// Add registers a route and its responder, along with its role (page vs
// partial).
// TODO(paulsmith): is this the right API to expose? design of role param,
// specifically
func (r *Routes) Add(route string, responder Responder, role RouteRole) {
	*r = append(*r, newRoute(route, responder, role))
}

func newRoute(path string, responder Responder, role RouteRole) *route {
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

func (routes *Routes) Respond(w http.ResponseWriter, r *http.Request) error {
	routeMatch := getRouteFromPath(routes, r.URL.Path)
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
		if route.role == RoutePartial {
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

func zipMap[K comparable, V any](ks []K, vs []V) map[K]V {
	m := make(map[K]V)
	for i := range ks {
		m[ks[i]] = vs[i]
	}
	return m
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

func getRouteFromPath(routes *Routes, path string) routeMatch {
	var matchedRoutes []*route

	for _, r := range *routes {
		if r.regex.MatchString(path) {
			matchedRoutes = append(matchedRoutes, r)
		}
	}

	if len(matchedRoutes) == 0 {
		// check trailing slash
		if path[len(path)-1] == '/' {
			lessSlash := path[:len(path)-1]
			for _, r := range *routes {
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
