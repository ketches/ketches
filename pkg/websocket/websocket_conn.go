package websocket

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/websocket"
)

var ErrUpgradeFailed = errors.New("websocket upgrade failed")

var (
	// MaxPayloadBytes keeps terminal clients from allocating unbounded frame
	// buffers while still allowing large pastes and command output.
	MaxPayloadBytes = 1 << 20
	// IdleTimeout bounds sessions that have stopped sending and receiving data.
	IdleTimeout = 30 * time.Minute
	// SessionTimeout bounds the lifetime of a single terminal session.
	SessionTimeout = 12 * time.Hour
	writeQueueSize = 32
	writeTimeout   = 15 * time.Second
)

// OriginValidator is called with the browser Origin during the handshake.
type OriginValidator func(origin string) bool

// NewConn upgrades an HTTP connection to a WebSocket connection
//
// The optional validator is used by HTTP handlers to share the CORS
// allowlist. A missing Origin is always rejected; callers that do not provide
// a validator still get syntactic Origin validation for compatibility.
func NewConn(w http.ResponseWriter, r *http.Request, validators ...OriginValidator) (*websocket.Conn, error) {
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
		Handshake: func(config *websocket.Config, req *http.Request) error {
			origin, err := websocket.Origin(config, req)
			if err != nil || origin == nil {
				return websocket.ErrBadWebSocketOrigin
			}
			if len(validators) > 0 && validators[0] != nil && !validators[0](req.Header.Get("Origin")) {
				return websocket.ErrBadWebSocketOrigin
			}
			config.Origin = origin
			return nil
		},
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			// This line is reached only after a successful handshake/upgrade.
			ws.MaxPayloadBytes = MaxPayloadBytes
			_ = ws.SetReadDeadline(time.Now().Add(IdleTimeout))
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
	conn    *websocket.Conn
	pending []byte
}

// NewReader creates a new Reader from a WebSocket connection
func NewReader(conn *websocket.Conn) *Reader {
	return &Reader{conn: conn}
}

// Read implements io.Reader
func (r *Reader) Read(p []byte) (int, error) {
	if len(r.pending) > 0 {
		n := copy(p, r.pending)
		r.pending = r.pending[n:]
		return n, nil
	}
	if len(p) == 0 {
		return 0, nil
	}
	_ = r.conn.SetReadDeadline(time.Now().Add(IdleTimeout))
	var data []byte
	err := websocket.Message.Receive(r.conn, &data)
	if err != nil {
		return 0, err
	}
	n := copy(p, data)
	if n < len(data) {
		r.pending = data[n:]
	}
	return n, nil
}

// Writer wraps a WebSocket connection to implement io.Writer for stdout/stderr
type Writer struct {
	state     *writerState
	closeOnce sync.Once
}

type writeRequest struct {
	data []byte
	ack  chan error
}

type writerState struct {
	conn     *websocket.Conn
	queue    chan writeRequest
	done     chan struct{}
	stop     chan struct{}
	stopOnce sync.Once
	errMu    sync.RWMutex
	err      error
	refs     atomic.Int32
}

func newWriterState(conn *websocket.Conn) *writerState {
	state := &writerState{
		conn:  conn,
		queue: make(chan writeRequest, writeQueueSize),
		done:  make(chan struct{}),
		stop:  make(chan struct{}),
	}
	state.refs.Store(1)
	go state.run()
	return state
}

func (s *writerState) run() {
	defer close(s.done)
	for {
		select {
		case <-s.stop:
			s.setErr(io.ErrClosedPipe)
			return
		case req := <-s.queue:
			_ = s.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			err := websocket.Message.Send(s.conn, req.data)
			_ = s.conn.SetWriteDeadline(time.Time{})
			req.ack <- err
			if err != nil {
				s.setErr(err)
				return
			}
			_ = s.conn.SetReadDeadline(time.Now().Add(IdleTimeout))
		}
	}
}

func (s *writerState) setErr(err error) {
	s.errMu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.errMu.Unlock()
}

func (s *writerState) getErr() error {
	s.errMu.RLock()
	defer s.errMu.RUnlock()
	if s.err != nil {
		return s.err
	}
	return io.ErrClosedPipe
}

func (s *writerState) close() {
	s.stopOnce.Do(func() { close(s.stop) })
	<-s.done
}

// NewWriter creates a new Writer from a WebSocket connection
func NewWriter(conn *websocket.Conn) *Writer {
	return &Writer{state: newWriterState(conn)}
}

// NewWriterPair returns two Writer handles backed by one serial write pump.
// It is intended for remotecommand's stdout and stderr streams.
func NewWriterPair(conn *websocket.Conn) (*Writer, *Writer) {
	state := newWriterState(conn)
	state.refs.Add(1)
	return &Writer{state: state}, &Writer{state: state}
}

// Write implements io.Writer
func (w *Writer) Write(p []byte) (int, error) {
	if w == nil || w.state == nil {
		return 0, io.ErrClosedPipe
	}
	if len(p) == 0 {
		return 0, nil
	}
	req := writeRequest{data: append([]byte(nil), p...), ack: make(chan error, 1)}
	select {
	case <-w.state.done:
		return 0, w.state.getErr()
	case <-w.state.stop:
		return 0, w.state.getErr()
	case w.state.queue <- req:
	}
	select {
	case err := <-req.ack:
		if err != nil {
			return 0, err
		}
		return len(p), nil
	case <-w.state.done:
		return 0, w.state.getErr()
	}
}

// Close stops the shared writer pump. It is safe to call more than once.
func (w *Writer) Close() error {
	if w == nil || w.state == nil {
		return nil
	}
	w.closeOnce.Do(func() {
		if w.state.refs.Add(-1) == 0 {
			w.state.close()
		}
	})
	return nil
}

// Done is closed when the writer can no longer send data.
func (w *Writer) Done() <-chan struct{} {
	if w == nil || w.state == nil {
		return closedChan()
	}
	return w.state.done
}

func closedChan() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
