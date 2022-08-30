package build

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRegexPatFromRoute(t *testing.T) {
	tests := []struct {
		route string
		want  routePat
	}{
		{
			"/",
			routePat{"/", nil},
		},
		{
			"/foo",
			routePat{"/foo", nil},
		},
		{
			"/:foo",
			routePat{"/([^/]+)", []string{"foo"}},
		},
		{
			"/:foo/bar",
			routePat{"/([^/]+)/bar", []string{"foo"}},
		},
		{
			"/foo/:bar",
			routePat{"/foo/([^/]+)", []string{"bar"}},
		},
		{
			"/foo/:bar/:quux",
			routePat{"/foo/([^/]+)/([^/]+)", []string{"bar", "quux"}},
		},
		{
			"/:foo/bar/:quux",
			routePat{"/([^/]+)/bar/([^/]+)", []string{"foo", "quux"}},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := regexPatFromRoute(test.route)
			diff := cmp.Diff(test.want, got, cmp.AllowUnexported(routePat{}))
			if diff != "" {
				t.Errorf("(-want, +got)\n%s", diff)
			}
		})
	}
}

func TestMostSpecificMatch(t *testing.T) {
	tests := []struct {
		routes []*route
		path   string
		want   int
	}{
		{
			[]*route{
				newRoute("/", nil, routePage),
			},
			"/",
			0,
		},
		{
			[]*route{
				newRoute("/:id", nil, routePage),
				newRoute("/new", nil, routePage),
			},
			"/new",
			1,
		},
		{
			[]*route{
				newRoute("/:name/:thing1/:thing2", nil, routePage),
				newRoute("/:name/foo/baz", nil, routePage),
			},
			"/foo/bar/baz",
			1,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := mostSpecificMatch(test.routes, test.path)
			if want := test.routes[test.want]; want != got {
				t.Errorf("want %v, got %v", want, got)
			}
		})
	}
}

type dummyPage struct{}

func (p *dummyPage) Respond(http.ResponseWriter, *http.Request) error {
	return nil
}

func (p *dummyPage) filePath() string {
	return ""
}

var _ page = (*dummyPage)(nil)

func TestIsPartialRoute(t *testing.T) {
	oldRoutes := routes
	defer func() {
		routes = oldRoutes
	}()
	dummy := new(dummyPage)
	routes = routeList{}
	routes.add("/sports/leagues/", dummy, routePage)
	routes.add("/sports/leagues/teams", dummy, routePartial)
	routes.add("/fruits/:name/", dummy, routePage)
	routes.add("/fruits/:name/nutrition", dummy, routePartial)
	tests := []struct {
		mainRoute string
		path      string
		want      bool
	}{
		{mainRoute: "/sports/leagues/", path: "/sports/leagues/", want: false},
		{mainRoute: "/sports/leagues/", path: "/sports/leagues/teams", want: true},
		{mainRoute: "/fruits/:name/", path: "/fruits/cherry/", want: false},
		{mainRoute: "/fruits/:name/", path: "/fruits/cherry/nutrition", want: true},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := isPartialRoute(test.mainRoute, test.path)
			if test.want != got {
				t.Errorf("want %t, got %t", test.want, got)
			}
		})
	}
}

func TestDisplayPartialHere(t *testing.T) {
	oldRoutes := routes
	defer func() {
		routes = oldRoutes
	}()
	dummy := new(dummyPage)
	routes = routeList{}
	routes.add("/sports/", dummy, routePage)
	routes.add("/sports/leagues", dummy, routePartial)
	routes.add("/dyn/:name", dummy, routePage)
	routes.add("/dyn/:name/foo", dummy, routePartial)
	routes.add("/dyn/:name/foo/bar", dummy, routePartial)
	routes.add("/dyn/:name/quux", dummy, routePartial)
	routes.add("/nested/", dummy, routePage)
	routes.add("/nested/foo", dummy, routePartial)
	routes.add("/nested/foo/bar", dummy, routePartial)
	tests := []struct {
		mainRoute   string
		partialPath string
		requestPath string
		want        bool
	}{
		{mainRoute: "/sports/", partialPath: "leagues", requestPath: "/sports/", want: true},
		{mainRoute: "/sports/", partialPath: "leagues", requestPath: "/sports/leagues", want: true},
		{mainRoute: "/dyn/:name", partialPath: "foo", requestPath: "/dyn/world", want: true},
		{mainRoute: "/dyn/:name", partialPath: "foo", requestPath: "/dyn/world/foo", want: true},
		{mainRoute: "/dyn/:name", partialPath: "foo/bar", requestPath: "/dyn/world/foo", want: true},
		{mainRoute: "/dyn/:name", partialPath: "foo/bar", requestPath: "/dyn/world/foo/bar", want: true},
		{mainRoute: "/dyn/:name", partialPath: "quux", requestPath: "/dyn/world/foo", want: false},
		{mainRoute: "/nested/", partialPath: "foo", requestPath: "/nested/", want: true},
		{mainRoute: "/nested/", partialPath: "foo/bar", requestPath: "/nested/", want: true},
		{mainRoute: "/nested/", partialPath: "foo", requestPath: "/nested/foo", want: true},
		{mainRoute: "/nested/", partialPath: "foo/bar", requestPath: "/nested/foo", want: true},
		{mainRoute: "/nested/", partialPath: "foo", requestPath: "/nested/foo/bar", want: false},
		{mainRoute: "/nested/", partialPath: "foo/bar", requestPath: "/nested/foo/bar", want: true},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := displayPartialHere(test.mainRoute, test.partialPath, test.requestPath)
			if test.want != got {
				t.Errorf("want %t, got %t", test.want, got)
			}
		})
	}
}

func TestMatchURLPathSegmentPrefix(t *testing.T) {
	const segmatch = `([^/]+)`
	tests := []struct {
		re   string
		url  string
		want bool
	}{
		{re: "/", url: "/", want: true},
		{re: "/", url: "/foo", want: false},
		{re: "/", url: "/foo/bar", want: false},
		{re: "/", url: "/foo/bar/", want: false},
		{re: "/foo/bar", url: "/foo", want: true},
		{re: "/foo/bar", url: "/foo/bar", want: true},
		{re: "/dyn/" + segmatch + "/", url: "/dyn/world/", want: true},
		{re: "/dyn/" + segmatch + "/", url: "/dyn/world/extra", want: false},
		{re: "/dyn/" + segmatch + "/extra", url: "/dyn/world/something/else", want: false},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := matchURLPathSegmentPrefix(regexp.MustCompile(test.re), test.url)
			if test.want != got {
				t.Errorf("want %t, got %t", test.want, got)
			}
		})
	}
}
