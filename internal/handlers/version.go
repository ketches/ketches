package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
)

var buildTime = time.Now().UTC().Format(time.RFC3339)

func GetVersion(c *gin.Context) {
	api.Success(c, gin.H{
		"version":    app.Version,
		"build_time": buildTime,
	})
}
