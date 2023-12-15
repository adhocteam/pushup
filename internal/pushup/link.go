// this file is mechanically generated, do not edit!
package pushup

import "github.com/adhocteam/pushup/testdata"
import "github.com/adhocteam/pushup/api"

var Router *api.Router

func init() {
	routes := new(api.Routes)
	routes.Add("/../testdata/textelement", &testdata.TextelementPage{}, api.RoutePage)
	Router = api.NewRouter(routes)
}
