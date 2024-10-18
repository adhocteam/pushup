package route

import (
	"log/slog"
	"net/http"
	"sync"
)

func Register(path string, responder Responder) {
	logger := slog.Default()
	globalRouterLock.Lock()
	defer globalRouterLock.Unlock()
	logger.Info("Registering", "path", path)
	globalRouter[path] = route{path: path, responder: responder}
}

type route struct {
	path      string
	responder Responder
}

var (
	globalRouter     = make(map[string]route)
	globalRouterLock sync.Mutex
)

type Responder interface {
	Respond(http.ResponseWriter, *http.Request) error
}

type responderHandler struct {
	responder Responder
}

func (h *responderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.responder.Respond(w, r)
	if err != nil {
		// TODO: custom error page
		http.Error(w, http.StatusText(500), 500)
	}
}

func Handler() http.Handler {
	globalRouterLock.Lock()
	defer globalRouterLock.Unlock()

	mux := http.NewServeMux()
	for _, route := range globalRouter {
		mux.Handle(route.path, &responderHandler{responder: route.responder})
	}

	return mux
}
