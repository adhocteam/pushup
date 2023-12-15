// this file is mechanically generated, do not edit!
package pushup

import "github.com/adhocteam/pushup/example/pages/projects/pid__param/users"
import "github.com/adhocteam/pushup/example/pages/x"
import "github.com/adhocteam/pushup/api"
import "github.com/adhocteam/pushup/example/pages"
import "github.com/adhocteam/pushup/example/pages/crud/album/delete"
import "github.com/adhocteam/pushup/example/pages/dyn"
import "github.com/adhocteam/pushup/example/pages/htmx"
import "github.com/adhocteam/pushup/example/pages/partials/architects"
import "github.com/adhocteam/pushup/example/pages/crud/album/edit"
import "github.com/adhocteam/pushup/example/pages/crud/album"
import "github.com/adhocteam/pushup/example/pages/crud"
import "github.com/adhocteam/pushup/example/pages/partials"
import "embed"

var Router *api.Router

//go:embed static
var static embed.FS

func init() {
	routes := new(api.Routes)
	routes.Add("/about", &pages.AboutPage{}, api.RoutePage)
	routes.Add("/alt-layout", &pages.AltLayoutPage{}, api.RoutePage)
	routes.Add("/crud/album/delete/:id", &delete.IdParamPage{}, api.RoutePage)
	routes.Add("/crud/album/edit/:id", &edit.IdParamPage{}, api.RoutePage)
	routes.Add("/crud/album/:id", &album.IdParamPage{}, api.RoutePage)
	routes.Add("/crud/album/new", &album.NewPage{}, api.RoutePage)
	routes.Add("/crud/", &crud.IndexPage{}, api.RoutePage)
	routes.Add("/dump", &pages.DumpPage{}, api.RoutePage)
	routes.Add("/dyn/:name", &dyn.NameParamPage{}, api.RoutePage)
	routes.Add("/escape", &pages.EscapePage{}, api.RoutePage)
	routes.Add("/for", &pages.ForPage{}, api.RoutePage)
	routes.Add("/htmx/active-search", &htmx.ActiveSearchPage{}, api.RoutePage)
	routes.Add("/htmx/active-search/results", &htmx.PagesHtmxActiveSearchResultsPartial{}, api.RoutePartial)
	routes.Add("/htmx/click-to-load", &htmx.ClickToLoadPage{}, api.RoutePage)
	routes.Add("/htmx/click-to-load/rows", &htmx.PagesHtmxClickToLoadRowsPartial{}, api.RoutePartial)
	routes.Add("/htmx/", &htmx.IndexPage{}, api.RoutePage)
	routes.Add("/htmx/value-select", &htmx.ValueSelectPage{}, api.RoutePage)
	routes.Add("/htmx/value-select/models", &htmx.PagesHtmxValueSelectModelsPartial{}, api.RoutePartial)
	routes.Add("/if", &pages.IfPage{}, api.RoutePage)
	routes.Add("/", &pages.IndexPage{}, api.RoutePage)
	routes.Add("/no-layout", &pages.NoLayoutPage{}, api.RoutePage)
	routes.Add("/partials/architects/", &architects.IndexPage{}, api.RoutePage)
	routes.Add("/partials/architects/list", &architects.PagesPartialsArchitectsIndexListPartial{}, api.RoutePartial)
	routes.Add("/partials/", &partials.IndexPage{}, api.RoutePage)
	routes.Add("/partials/nested", &partials.NestedPage{}, api.RoutePage)
	routes.Add("/partials/nested/foo/bar", &partials.PagesPartialsNestedFooBarPartial{}, api.RoutePartial)
	routes.Add("/partials/nested/foo", &partials.PagesPartialsNestedFooPartial{}, api.RoutePartial)
	routes.Add("/partials/nested/first/second/third", &partials.PagesPartialsNestedFirstSecondThirdPartial{}, api.RoutePartial)
	routes.Add("/partials/nested/first/second", &partials.PagesPartialsNestedFirstSecondPartial{}, api.RoutePartial)
	routes.Add("/partials/nested/first", &partials.PagesPartialsNestedFirstPartial{}, api.RoutePartial)
	routes.Add("/projects/:pid/users/:uid", &users.UidParamPage{}, api.RoutePage)
	routes.Add("/source", &pages.SourcePage{}, api.RoutePage)
	routes.Add("/x/sub", &x.SubPage{}, api.RoutePage)
	Router = api.NewRouter(routes)
	Router.AddStatic(static)
}
