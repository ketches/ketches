package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
)

func GetVersion(c *gin.Context) {
	api.Success(c, gin.H{
		"version":    app.Version,
		"commit":     app.Commit,
		"build_time": app.BuildTime,
		"tag":        app.Tag,
	})
}
