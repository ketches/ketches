package websocket

import (
	"github.com/gorilla/websocket"
)

type Reader struct {
	conn *websocket.Conn
}

func NewReader(conn *websocket.Conn) *Reader {
	return &Reader{conn: conn}
}

func (r *Reader) Read(p []byte) (int, error) {
	_, message, err := r.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	copy(p, message)
	return len(message), nil
}
