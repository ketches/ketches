package handlers

import (
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListActivities(c *gin.Context) {
	var req models.OperationLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	total, items, err := services.ListActivities(req, claims.UserID, claims.Role == app.UserRoleAdmin)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, models.OperationLogListResponse{Items: items, Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize)})
}

func ListAppOperationLogs(c *gin.Context) {
	appID := c.Param("appID")
	var req models.OperationLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()
	total, items, err := services.ListAppOperationLogs(appID, req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, models.OperationLogListResponse{Items: items, Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize)})
}

func ListCodeRepositoryOperationLogs(c *gin.Context) {
	repoID := c.Param("repoID")
	var req models.OperationLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()
	total, items, err := services.ListCodeRepositoryOperationLogs(repoID, req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, models.OperationLogListResponse{Items: items, Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize)})
}

func ListOperationLogs(c *gin.Context) {
	var req models.OperationLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()
	total, items, err := services.ListOperationLogs(req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	if req.Export {
		exportOperationLogsCSV(c, items)
		return
	}
	api.Success(c, models.OperationLogListResponse{Items: items, Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize)})
}

func ListPlatformAuditLogs(c *gin.Context) {
	var req models.OperationLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()
	total, items, err := services.ListPlatformOperationLogs(req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, models.OperationLogListResponse{Items: items, Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize)})
}

func GetOperationLogSettings(c *gin.Context) {
	days, err := services.GetOperationLogRetentionDays()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, models.OperationLogSettingsResponse{RetentionDays: days})
}

func UpdateOperationLogSettings(c *gin.Context) {
	var req models.UpdateOperationLogSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	if err := services.UpdateOperationLogRetentionDays(req.RetentionDays); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	api.NoContent(c)
}

func exportOperationLogsCSV(c *gin.Context, items []models.OperationLogItem) {
	filename := "operation-logs-" + time.Now().Format("20060102-150405") + ".csv"
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)

	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"id", "created_at", "user_id", "username", "action", "resource_type", "resource_id", "project_id", "env_id", "app_id", "repo_id", "status", "status_code", "sensitivity", "request_summary", "client_ip"})
	for i := range items {
		item := items[i]
		_ = w.Write([]string{
			item.ID,
			item.CreatedAt.Format(time.RFC3339),
			item.UserID,
			item.Username,
			item.Action,
			item.ResourceType,
			item.ResourceID,
			item.ProjectID,
			item.EnvID,
			item.AppID,
			item.RepoID,
			item.Status,
			strconv.Itoa(item.StatusCode),
			item.Sensitivity,
			item.RequestSummary,
			item.ClientIP,
		})
	}
	w.Flush()
}
