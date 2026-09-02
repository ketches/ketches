package middlewares

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// StreamingResponse removes the server's absolute response deadline for
// long-lived streams. Stream handlers remain bounded by request cancellation
// and their own idle-timeout policies.
func StreamingResponse() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := http.NewResponseController(c.Writer).SetWriteDeadline(time.Time{})
		if err != nil && !errors.Is(err, http.ErrNotSupported) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize response stream"})
			return
		}
		c.Next()
	}
}
