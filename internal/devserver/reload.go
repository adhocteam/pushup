package devserver

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

type reloader struct {
	sync.RWMutex
	clients map[*websocket.Conn]bool
}

func newReloader() *reloader {
	r := &reloader{clients: make(map[*websocket.Conn]bool)}
	return r
}

func (r *reloader) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	conn, err := websocket.Accept(w, req, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		slog.Error("accepting websocket connection", "error", err)
		return
	}
	defer conn.CloseNow()
	slog.Debug("handling websocket connection", "conn", conn)

	r.Lock()
	r.clients[conn] = true
	r.Unlock()

	ctx := req.Context()
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			r.Lock()
			delete(r.clients, conn)
			r.Unlock()
			return
		}
	}
}

var reloadMsg = []byte("reload")

func (r *reloader) triggerReload() {
	r.Lock()
	defer r.Unlock()

	ctx := context.Background()
	for client := range r.clients {
		err := client.Write(ctx, websocket.MessageText, []byte("reload"))
		if err != nil {
			slog.Error("getting writer on client connection", "error", err)
			client.Close(websocket.StatusInternalError, "write failed")
			continue
		}
		client.Close(websocket.StatusGoingAway, "reloading server")
	}

	clear(r.clients)
}
