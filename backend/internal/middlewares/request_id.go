package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/pkg/uuid"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		api.SetRequestID(c, uuid.New())
		c.Next()
	}
}
