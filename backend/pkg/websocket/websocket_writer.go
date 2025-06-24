package websocket

import (
	"github.com/gorilla/websocket"
)

type Writer struct {
	conn *websocket.Conn
}

func NewWriter(conn *websocket.Conn) *Writer {
	return &Writer{conn: conn}
}

func (w *Writer) Write(p []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
