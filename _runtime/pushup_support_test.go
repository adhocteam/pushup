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
