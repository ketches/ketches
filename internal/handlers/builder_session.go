package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

var builderRunLogsTerminalPollInterval = 100 * time.Millisecond

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

func RequestBuilderRunCancel(c *gin.Context) {
	projectID := c.Param("projectID")
	sessionID := c.Param("sessionID")
	runID := c.Param("runID")

	if _, err := services.RequestBuilderSessionRunCancel(c.Request.Context(), projectID, sessionID, runID); err != nil {
		api.Error(c, builderSessionErrorStatus(err), err)
		return
	}

	detail, err := services.GetBuilderSessionDetail(c.Request.Context(), projectID, sessionID)
	if err != nil {
		api.Error(c, builderSessionErrorStatus(err), err)
		return
	}

	api.Success(c, detail)
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
	case errors.Is(err, services.ErrBuilderRunNotFound):
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
	afterSequence, err := parseBuilderRunLogsAfterCursor(c)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	run, err := services.GetBuilderRun(c.Request.Context(), projectID, sessionID, runID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	liveEvents, unsubscribe := services.SubscribeBuilderRunEvents(runID)
	defer unsubscribe()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	replayedEvents, err := services.ReplayBuilderRunEventsAfterCursor(c.Request.Context(), runID, afterSequence)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	for i := range replayedEvents {
		streamBuilderRunEvent(c, &replayedEvents[i])
		afterSequence = replayedEvents[i].Sequence
	}
	run, err = services.GetBuilderRun(c.Request.Context(), projectID, sessionID, runID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	if isBuilderRunTerminalStatus(run.Status) {
		if drainBuilderRunQueuedLiveEvents(c, liveEvents, &afterSequence) {
			return
		}
		c.SSEvent("done", "stream ended")
		c.Writer.Flush()
		return
	}

	terminalTicker := time.NewTicker(builderRunLogsTerminalPollInterval)
	defer terminalTicker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-terminalTicker.C:
			run, err = services.GetBuilderRun(c.Request.Context(), projectID, sessionID, runID)
			if err != nil {
				return
			}
			if isBuilderRunTerminalStatus(run.Status) {
				if drainBuilderRunQueuedLiveEvents(c, liveEvents, &afterSequence) {
					return
				}
				c.SSEvent("done", "stream ended")
				c.Writer.Flush()
				return
			}
		case event, ok := <-liveEvents:
			if !ok {
				return
			}
			if event.Sequence <= afterSequence {
				continue
			}
			streamBuilderRunEvent(c, &event)
			afterSequence = event.Sequence

			run, err = services.GetBuilderRun(c.Request.Context(), projectID, sessionID, runID)
			if err != nil {
				return
			}
			if isBuilderRunTerminalStatus(run.Status) {
				if drainBuilderRunQueuedLiveEvents(c, liveEvents, &afterSequence) {
					return
				}
				c.SSEvent("done", "stream ended")
				c.Writer.Flush()
				return
			}
		}
	}
}

func drainBuilderRunQueuedLiveEvents(c *gin.Context, liveEvents <-chan entities.BuilderRunEvent, afterSequence *int64) bool {
	if afterSequence == nil {
		return false
	}

	for {
		select {
		case event, ok := <-liveEvents:
			if !ok {
				return true
			}
			if event.Sequence <= *afterSequence {
				continue
			}
			streamBuilderRunEvent(c, &event)
			*afterSequence = event.Sequence
		default:
			return false
		}
	}
}

func parseBuilderRunLogsAfterCursor(c *gin.Context) (int64, error) {
	afterParam := c.Query("after")
	if afterParam == "" {
		return 0, nil
	}

	afterSequence, err := strconv.ParseInt(afterParam, 10, 64)
	if err != nil || afterSequence < 0 {
		return 0, errors.New("after must be a non-negative integer")
	}
	return afterSequence, nil
}

func streamBuilderRunEvent(c *gin.Context, event *entities.BuilderRunEvent) {
	if event == nil || event.Message == "" {
		return
	}
	c.Render(-1, sse.Event{
		Id:    strconv.FormatInt(event.Sequence, 10),
		Event: "log",
		Data:  event.Message,
	})
	c.Writer.Flush()
}

func isBuilderRunTerminalStatus(status entities.BuilderRunStatus) bool {
	switch status {
	case entities.BuilderRunStatusSucceeded, entities.BuilderRunStatusFailed, entities.BuilderRunStatusCancelled, entities.BuilderRunStatusTimedOut:
		return true
	default:
		return false
	}
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
