package middlewares

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
)

const (
	csrfCookieName = app.CSRFCookieName
	csrfHeaderName = app.CSRFHeaderName
)

func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			c.Next()
			return
		}

		cookieValue, err := c.Cookie(csrfCookieName)
		if err != nil || cookieValue == "" {
			api.Error(c, http.StatusForbidden, errors.New("csrf token is required"))
			c.Abort()
			return
		}

		headerValue := c.GetHeader(csrfHeaderName)
		if headerValue == "" || headerValue != cookieValue {
			api.Error(c, http.StatusForbidden, errors.New("csrf token is invalid"))
			c.Abort()
			return
		}

		c.Next()
	}
}
