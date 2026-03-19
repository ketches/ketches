package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListBuilderSessions(c *gin.Context) {
	projectID := c.Param("projectID")

	sessions, err := services.ListBuilderSessions(c.Request.Context(), projectID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, sessions)
}

func CreateBuilderSession(c *gin.Context) {
	projectID := c.Param("projectID")
	userID, ok := requireBuilderSessionUserID(c)
	if !ok {
		return
	}

	var req models.CreateBuilderSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	detail, err := services.CreateBuilderSession(c.Request.Context(), projectID, userID, &req)
	if err != nil {
		api.Error(c, builderSessionErrorStatus(err), err)
		return
	}

	api.Created(c, detail)
}

func GetBuilderSession(c *gin.Context) {
	projectID := c.Param("projectID")
	sessionID := c.Param("sessionID")

	detail, err := services.GetBuilderSessionDetail(c.Request.Context(), projectID, sessionID)
	if err != nil {
		api.Error(c, builderSessionErrorStatus(err), err)
		return
	}

	api.Success(c, detail)
}

func PostBuilderSessionMessage(c *gin.Context) {
	projectID := c.Param("projectID")
	sessionID := c.Param("sessionID")
	userID, ok := requireBuilderSessionUserID(c)
	if !ok {
		return
	}

	var req models.AppendBuilderSessionMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	detail, err := services.AppendBuilderSessionMessage(c.Request.Context(), projectID, sessionID, userID, &req)
	if err != nil {
		api.Error(c, builderSessionErrorStatus(err), err)
		return
	}

	api.Created(c, detail)
}

func requireBuilderSessionUserID(c *gin.Context) (string, bool) {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return "", false
	}
	return claims.UserID, true
}

func builderSessionErrorStatus(err error) int {
	switch {
	case errors.Is(err, services.ErrBuilderSessionNotFound):
		return http.StatusNotFound
	case errors.Is(err, services.ErrBuilderSessionNotAppendable):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func StreamBuilderRunLogs(c *gin.Context) {
	projectID := c.Param("projectID")
	sessionID := c.Param("sessionID")
	runID := c.Param("runID")

	run, err := services.GetBuilderRun(c.Request.Context(), projectID, sessionID, runID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	if run.ExecutionLog != "" {
		c.SSEvent("log", run.ExecutionLog)
		c.Writer.Flush()
	}

	if run.Status == entities.BuilderRunStatusSucceeded ||
		run.Status == entities.BuilderRunStatusFailed ||
		run.Status == entities.BuilderRunStatusCancelled {
		c.SSEvent("done", "stream ended")
		c.Writer.Flush()
		return
	}

	c.SSEvent("done", "stream ended")
	c.Writer.Flush()
}

func ListBuilderWorkspaceFiles(c *gin.Context) {
	projectID := c.Param("projectID")
	sessionID := c.Param("sessionID")
	path := c.Query("path")

	result, err := services.ListBuilderWorkspaceFiles(c.Request.Context(), projectID, sessionID, path)
	if err != nil {
		api.Error(c, builderSessionErrorStatus(err), err)
		return
	}

	api.Success(c, result)
}

func ReadBuilderWorkspaceFile(c *gin.Context) {
	projectID := c.Param("projectID")
	sessionID := c.Param("sessionID")
	path := c.Query("path")

	if path == "" {
		api.Error(c, http.StatusBadRequest, errors.New("path parameter is required"))
		return
	}

	result, err := services.ReadBuilderWorkspaceFile(c.Request.Context(), projectID, sessionID, path)
	if err != nil {
		api.Error(c, builderSessionErrorStatus(err), err)
		return
	}

	api.Success(c, result)
}

func DownloadBuilderWorkspace(c *gin.Context) {
	projectID := c.Param("projectID")
	sessionID := c.Param("sessionID")

	c.Header("Content-Type", "application/x-tar")
	c.Header("Content-Disposition", `attachment; filename="workspace.tar.gz"`)

	if err := services.DownloadBuilderWorkspace(c.Request.Context(), projectID, sessionID, c.Writer); err != nil {
		c.Error(err)
	}
}
