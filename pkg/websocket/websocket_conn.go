package websocket

import (
	"errors"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"
)

var ErrUpgradeFailed = errors.New("websocket upgrade failed")

// NewConn upgrades an HTTP connection to a WebSocket connection
func NewConn(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	done := make(chan struct{})
	serveExited := make(chan struct{})

	var (
		conn   *websocket.Conn
		retErr error
		once   sync.Once
	)

	// Ensure we never hang if the request is canceled before upgrade completes.
	go func() {
		<-r.Context().Done()
		once.Do(func() {
			retErr = r.Context().Err()
			close(done)
		})
	}()

	srv := websocket.Server{
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			// This line is reached only after a successful handshake/upgrade.
			once.Do(func() {
				conn = ws
				close(done)
			})

			// Keep the upgraded connection alive while the outer HTTP handler runs.
			<-r.Context().Done()
		}),
	}

	go func() {
		defer close(serveExited)
		// On handshake failure, ServeHTTP writes an HTTP error response and returns.
		srv.ServeHTTP(w, r)

		// If we get here before Handler ran, the upgrade failed.
		once.Do(func() {
			retErr = ErrUpgradeFailed
			close(done)
		})
	}()

	<-done
	// When returning an error, wait for ServeHTTP to exit so it does not call
	// Hijack() after the handler writes a response (which would panic in Gin).
	if retErr != nil {
		<-serveExited
	}
	return conn, retErr
}

// Reader wraps a WebSocket connection to implement io.Reader for stdin
type Reader struct {
	conn *websocket.Conn
}

// NewReader creates a new Reader from a WebSocket connection
func NewReader(conn *websocket.Conn) *Reader {
	return &Reader{conn: conn}
}

// Read implements io.Reader
func (r *Reader) Read(p []byte) (int, error) {
	var data []byte
	err := websocket.Message.Receive(r.conn, &data)
	if err != nil {
		return 0, err
	}
	return copy(p, data), nil
}

// Writer wraps a WebSocket connection to implement io.Writer for stdout/stderr
type Writer struct {
	conn *websocket.Conn
	// mu   sync.Mutex
}

// NewWriter creates a new Writer from a WebSocket connection
func NewWriter(conn *websocket.Conn) *Writer {
	return &Writer{conn: conn}
}

// Write implements io.Writer
func (w *Writer) Write(p []byte) (int, error) {
	// w.mu.Lock()
	// defer w.mu.Unlock()

	err := websocket.Message.Send(w.conn, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
