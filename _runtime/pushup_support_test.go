package build

import (
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
				newRoute("/", nil),
			},
			"/",
			0,
		},
		{
			[]*route{
				newRoute("/:id", nil),
				newRoute("/new", nil),
			},
			"/new",
			1,
		},
		{
			[]*route{
				newRoute("/:name/:thing1/:thing2", nil),
				newRoute("/:name/foo/baz", nil),
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
