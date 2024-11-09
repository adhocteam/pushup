package route

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

type Role int

const (
	RolePage Role = iota
	RolePartial
)

func Register(pattern string, responder Responder, role Role) {
	globalRouterLock.Lock()
	defer globalRouterLock.Unlock()
	slog.Info("Registering", "pattern", pattern)
	globalRouter[pattern] = route{pattern: pattern, responder: responder, role: role}
}

type route struct {
	pattern   string // a net/http ServeMux pattern
	responder Responder
	role      Role
}

var (
	globalRouter     = make(map[string]route)
	globalRouterLock sync.Mutex
)

type Responder interface {
	Respond(http.ResponseWriter, *http.Request) error
}

type responderHandler struct {
	pattern   string
	responder Responder
}

func (h *responderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), patternKey{}, h.pattern)
	err := h.responder.Respond(w, r.Clone(ctx))
	if err != nil {
		// TODO: custom error page
		slog.Error("handling response", "error", err)
		http.Error(w, http.StatusText(500), 500)
	}
}

func Handler() http.Handler {
	globalRouterLock.Lock()
	defer globalRouterLock.Unlock()

	mux := http.NewServeMux()
	for _, route := range globalRouter {
		mux.Handle(route.pattern, &responderHandler{pattern: route.pattern, responder: route.responder})
	}

	return mux
}

type patternKey struct{}

func getPattern(ctx context.Context) string {
	val, ok := ctx.Value(patternKey{}).(string)
	if !ok {
		panic(fmt.Sprintf("unexpected type stored as context value, want string, got %T", ctx.Value(patternKey{})))
	}
	return val
}

func IsPartial(r *http.Request, partialRoutePattern string) bool {
	pattern := getPattern(r.Context())

	if partialRoutePattern == "" {
		globalRouterLock.Lock()
		defer globalRouterLock.Unlock()
		for pat, route := range globalRouter {
			if pattern == pat {
				return route.role == RolePartial
			}
		}
	}
	return pattern == partialRoutePattern
}
