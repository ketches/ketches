package handlers

import (
	"fmt"
	"net/http"

	"strings"
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/services"
	"github.com/ketches/ketches/internal/core/exporter"
)

type ImportRequest struct {
	Type             string `json:"type" binding:"required"`
	Content          string `json:"content" binding:"required"`
	ConflictStrategy string `json:"conflict_strategy"`
}

// ImportApps handles application import
func ImportApps(c *gin.Context) {
	envID := c.Param("envID")

	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	// Validate type
	switch req.Type {
	case "dockercompose", "kubernetes", "ketches":
		// valid
	default:
		api.Error(c, http.StatusBadRequest, fmt.Errorf("invalid type: %s", req.Type))
		return
	}

	// Set default conflict strategy
	if req.ConflictStrategy == "" {
		req.ConflictStrategy = "rename"
	}

	// Validate conflict strategy
	switch req.ConflictStrategy {
	case "rename", "ask", "error":
		// valid
	default:
		api.Error(c, http.StatusBadRequest, fmt.Errorf("invalid conflict_strategy: %s", req.ConflictStrategy))
		return
	}

	result, err := services.ImportApps(envID, req.Type, req.Content, req.ConflictStrategy)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, result)
}


// ExportApps handles single application export
func ExportApps(c *gin.Context) {
	appID := c.Param("appID")
	format := c.Query("format")

	if format == "" {
		api.Error(c, http.StatusBadRequest, fmt.Errorf("format parameter is required"))
		return
	}

	result, err := services.ExportApps([]string{appID}, exporter.ExportFormat(format))
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	respondWithExport(c, exporter.ExportFormat(format), result)
}

func respondWithExport(c *gin.Context, format exporter.ExportFormat, content string) {
	switch format {
	case exporter.FormatKubernetes:
		api.Success(c, gin.H{"yaml": content})
	case exporter.FormatKetches:
		api.Success(c, gin.H{"metadata": content})
	case exporter.FormatHelm:
		api.Success(c, gin.H{"chart": content})
	case exporter.FormatDockerCompose:
		api.Success(c, gin.H{"compose": content})
	default:
		api.Success(c, gin.H{"content": content})
	}
}


// ExportEnvApps handles environment applications export
func ExportEnvApps(c *gin.Context) {
	envID := c.Param("envID")
	format := c.Query("format")
	appIDsStr := c.Query("app_ids")

	if format == "" {
		api.Error(c, http.StatusBadRequest, fmt.Errorf("format parameter is required"))
		return
	}

	var appIDs []string
	if appIDsStr != "" {
		appIDs = strings.Split(appIDsStr, ",")
	}

	result, err := services.ExportEnvApps(envID, appIDs, exporter.ExportFormat(format))
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	respondWithExport(c, exporter.ExportFormat(format), result)
}
