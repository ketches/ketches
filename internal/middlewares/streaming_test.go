package middlewares

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type writeDeadlineRecorder struct {
	header   http.Header
	deadline time.Time
	status   int
}

func (w *writeDeadlineRecorder) Header() http.Header {
	return w.header
}

func (w *writeDeadlineRecorder) Write(body []byte) (int, error) {
	return len(body), nil
}

func (w *writeDeadlineRecorder) WriteHeader(statusCode int) {
	w.status = statusCode
}

func (w *writeDeadlineRecorder) SetWriteDeadline(deadline time.Time) error {
	w.deadline = deadline
	return nil
}

func TestStreamingResponseClearsServerWriteDeadline(t *testing.T) {
	gin.SetMode(gin.TestMode)

	writer := &writeDeadlineRecorder{
		header:   make(http.Header),
		deadline: time.Now().Add(time.Minute),
	}
	router := gin.New()
	router.GET("/stream", StreamingResponse(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req, err := http.NewRequest(http.MethodGet, "/stream", nil)
	require.NoError(t, err)
	router.ServeHTTP(writer, req)

	assert.True(t, writer.deadline.IsZero())
	assert.Equal(t, http.StatusNoContent, writer.status)
}
