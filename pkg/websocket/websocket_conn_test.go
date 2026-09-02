package websocket

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	xwebsocket "golang.org/x/net/websocket"
)

func dialTestWebSocket(t *testing.T, serverURL, origin string) *xwebsocket.Conn {
	t.Helper()
	config, err := xwebsocket.NewConfig("ws"+strings.TrimPrefix(serverURL, "http"), origin)
	require.NoError(t, err)
	conn, err := xwebsocket.DialConfig(config)
	require.NoError(t, err)
	return conn
}

func TestNewConnRejectsOriginOutsideAllowlist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = NewConn(w, r, func(origin string) bool { return origin == "https://allowed.example" })
	}))
	defer server.Close()

	config, err := xwebsocket.NewConfig("ws"+strings.TrimPrefix(server.URL, "http"), "https://blocked.example")
	require.NoError(t, err)
	_, err = xwebsocket.DialConfig(config)
	require.Error(t, err)
	// The HTTP response is a handshake rejection rather than an upgraded
	// connection, so the dial error must not be nil.
}

func TestNewConnRejectsMissingOrigin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = NewConn(w, r, func(string) bool { return true })
	}))
	defer server.Close()

	conn, err := net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
	require.NoError(t, err)
	defer conn.Close()
	_, err = fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n", strings.TrimPrefix(server.URL, "http://"))
	require.NoError(t, err)
	statusLine, err := bufio.NewReader(conn).ReadString('\n')
	require.NoError(t, err)
	require.Contains(t, statusLine, "403 Forbidden")
}

func TestWriterPairSerializesConcurrentStreams(t *testing.T) {
	const writes = 24
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := NewConn(w, r, func(string) bool { return true })
		if err != nil {
			return
		}
		defer conn.Close()
		stdout, stderr := NewWriterPair(conn)
		defer stdout.Close()
		defer stderr.Close()

		var wg sync.WaitGroup
		for i := 0; i < writes; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				writer := stdout
				if i%2 == 1 {
					writer = stderr
				}
				_, _ = writer.Write([]byte(fmt.Sprintf("message-%02d", i)))
			}(i)
		}
		wg.Wait()
	}))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL, "https://allowed.example")
	defer conn.Close()
	received := make(map[string]struct{}, writes)
	for i := 0; i < writes; i++ {
		var msg []byte
		require.NoError(t, xwebsocket.Message.Receive(conn, &msg))
		received[string(msg)] = struct{}{}
	}
	require.Len(t, received, writes)
	for i := 0; i < writes; i++ {
		_, ok := received[fmt.Sprintf("message-%02d", i)]
		require.True(t, ok)
	}
}

func TestDemuxReaderDoneWhenClientDisconnects(t *testing.T) {
	readerDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := NewConn(w, r, func(string) bool { return true })
		if err != nil {
			return
		}
		defer conn.Close()
		queue := NewTerminalSizeQueue()
		reader := NewDemuxReader(conn, queue)
		defer reader.Close()
		<-reader.Done()
		close(readerDone)
	}))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL, "https://allowed.example")
	_ = conn.Close()
	select {
	case <-readerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("demux reader did not stop after client disconnect")
	}
}

func TestDemuxReaderRejectsOversizedFrame(t *testing.T) {
	readerErr := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := NewConn(w, r, func(string) bool { return true })
		if err != nil {
			return
		}
		defer conn.Close()
		queue := NewTerminalSizeQueue()
		defer queue.Close()
		reader := NewDemuxReader(conn, queue)
		defer reader.Close()
		<-reader.Done()
		readerErr <- reader.Err()
	}))
	defer server.Close()

	conn := dialTestWebSocket(t, server.URL, "https://allowed.example")
	defer conn.Close()
	_ = xwebsocket.Message.Send(conn, make([]byte, MaxPayloadBytes+1))
	select {
	case err := <-readerErr:
		require.True(t, errors.Is(err, xwebsocket.ErrFrameTooLarge), "unexpected read error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("demux reader accepted an oversized frame")
	}
}
