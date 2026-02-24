package websocket

import (
	"encoding/json"
	"io"
	"sync"

	"golang.org/x/net/websocket"
	"k8s.io/client-go/tools/remotecommand"
)

type ResizeMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

type TerminalSizeQueue struct {
	ch chan remotecommand.TerminalSize
	mu sync.Mutex
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

	select {
	case q.ch <- size:
	default:
		<-q.ch
		q.ch <- size
	}
}

func (q *TerminalSizeQueue) Close() {
	close(q.ch)
}

type DemuxReader struct {
	conn      *websocket.Conn
	sizeQueue *TerminalSizeQueue
	pipeR     *io.PipeReader
	pipeW     *io.PipeWriter
	done      chan struct{}
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
	defer dr.pipeW.Close()
	defer close(dr.done)

	for {
		var data []byte
		err := websocket.Message.Receive(dr.conn, &data)
		if err != nil {
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
	dr.pipeW.Close()
	<-dr.done
	return nil
}
