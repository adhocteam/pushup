package pushup

import (
	"net/http"

	"github.com/adhocteam/pushup/api"
)

func Param(r *http.Request, name string) string {
	params := api.ParamsFromContext(r.Context())
	return params[name]
}
