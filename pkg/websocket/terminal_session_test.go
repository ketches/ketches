package websocket

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/remotecommand"
)

func TestTerminalSizeQueueCloseIsIdempotent(t *testing.T) {
	queue := NewTerminalSizeQueue()
	queue.Send(remotecommand.TerminalSize{Width: 80, Height: 24})
	queue.Close()
	queue.Close()
	queue.Send(remotecommand.TerminalSize{Width: 100, Height: 40})

	size := queue.Next()
	require.NotNil(t, size)
	require.Equal(t, uint16(80), size.Width)
	require.Nil(t, queue.Next())
}
