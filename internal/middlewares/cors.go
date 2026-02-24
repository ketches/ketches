package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}
		if isAllowedOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, Authorization, X-Requested-With")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isAllowedOrigin(origin string) bool {
	allowedOrigins := getAllowedOrigins()

	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}

		if allowed == origin {
			return true
		}

		if strings.Contains(allowed, "*.") {
			protocol := ""
			pattern := allowed

			if strings.HasPrefix(allowed, "https://*.") {
				protocol = "https://"
				pattern = allowed[len("https://*."):]
			} else if strings.HasPrefix(allowed, "http://*.") {
				protocol = "http://"
				pattern = allowed[len("http://*."):]
			} else if strings.HasPrefix(allowed, "*.") {
				pattern = allowed[2:]
			}

			if protocol != "" {
				expectedPrefix := protocol
				expectedSuffix := "." + pattern
				if strings.HasPrefix(origin, expectedPrefix) && strings.HasSuffix(origin, expectedSuffix) {
					return true
				}
			} else {
				expectedSuffix := "." + pattern
				if strings.HasSuffix(origin, expectedSuffix) {
					return true
				}
			}
		}
	}

	return false
}

func getAllowedOrigins() []string {
	originsStr := app.Config.CORSAllowedOrigins

	if originsStr == "" {
		return []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
		}
	}

	origins := strings.Split(originsStr, ",")
	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
