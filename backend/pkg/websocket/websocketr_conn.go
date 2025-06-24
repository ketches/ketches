package websocket

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/ketches/ketches/internal/app"
)

func NewConn(w http.ResponseWriter, r *http.Request) (*websocket.Conn, app.Error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Allow all origins for WebSocket connections
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, app.NewError(http.StatusInternalServerError, "Failed to upgrade connection")
	}
	return conn, nil
}
