package websocket

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	"golang.org/x/net/websocket"
	"k8s.io/client-go/tools/remotecommand"
)

type ResizeMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

type TerminalSizeQueue struct {
	ch        chan remotecommand.TerminalSize
	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
}

func NewTerminalSizeQueue() *TerminalSizeQueue {
	return &TerminalSizeQueue{
		ch: make(chan remotecommand.TerminalSize, 1),
	}
}

func (q *TerminalSizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-q.ch
	if !ok {
		return nil
	}
	return &size
}

func (q *TerminalSizeQueue) Send(size remotecommand.TerminalSize) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}

	select {
	case q.ch <- size:
	default:
		// Keep the newest resize event without blocking the reader.
		select {
		case <-q.ch:
		default:
		}
		select {
		case q.ch <- size:
		default:
		}
	}
}

func (q *TerminalSizeQueue) Close() {
	q.closeOnce.Do(func() {
		q.mu.Lock()
		q.closed = true
		close(q.ch)
		q.mu.Unlock()
	})
}

type DemuxReader struct {
	conn      *websocket.Conn
	sizeQueue *TerminalSizeQueue
	pipeR     *io.PipeReader
	pipeW     *io.PipeWriter
	done      chan struct{}
	closeOnce sync.Once
	errMu     sync.RWMutex
	err       error
}

func NewDemuxReader(conn *websocket.Conn, sizeQueue *TerminalSizeQueue) *DemuxReader {
	pipeR, pipeW := io.Pipe()
	dr := &DemuxReader{
		conn:      conn,
		sizeQueue: sizeQueue,
		pipeR:     pipeR,
		pipeW:     pipeW,
		done:      make(chan struct{}),
	}
	go dr.readLoop()
	return dr
}

func (dr *DemuxReader) Read(p []byte) (int, error) {
	return dr.pipeR.Read(p)
}

func (dr *DemuxReader) readLoop() {
	defer func() {
		_ = dr.pipeW.Close()
	}()
	defer close(dr.done)

	for {
		_ = dr.conn.SetReadDeadline(time.Now().Add(IdleTimeout))
		var data []byte
		err := websocket.Message.Receive(dr.conn, &data)
		if err != nil {
			dr.errMu.Lock()
			dr.err = err
			dr.errMu.Unlock()
			return
		}

		if len(data) > 0 && data[0] == '{' {
			var msg ResizeMessage
			if err := json.Unmarshal(data, &msg); err == nil && msg.Type == "resize" {
				dr.sizeQueue.Send(remotecommand.TerminalSize{
					Width:  msg.Cols,
					Height: msg.Rows,
				})
				continue
			}
		}

		if _, err := dr.pipeW.Write(data); err != nil {
			return
		}
	}
}

func (dr *DemuxReader) Close() error {
	dr.closeOnce.Do(func() {
		// A blocked Receive must be interrupted before waiting for readLoop.
		_ = dr.conn.SetReadDeadline(time.Now())
		_ = dr.pipeW.Close()
	})
	<-dr.done
	return nil
}

// Done is closed when the client stops sending data or the reader is closed.
func (dr *DemuxReader) Done() <-chan struct{} {
	return dr.done
}

// Err returns the terminal read error, if any.
func (dr *DemuxReader) Err() error {
	dr.errMu.RLock()
	defer dr.errMu.RUnlock()
	return dr.err
}
