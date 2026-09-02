package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	maxRequestBodyBytes = 16 << 20
	maxUploadBodyBytes  = 256 << 20
)

// RequestBodyLimit bounds request bodies before any binder or middleware can
// read them. Upload endpoints receive a larger, still finite, streaming limit.
func RequestBodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := int64(maxRequestBodyBytes)
		if strings.Contains(c.Request.URL.Path, "/upload") {
			limit = maxUploadBodyBytes
		}
		if c.Request.ContentLength > limit {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
