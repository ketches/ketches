package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// ListAppGroups returns all app groups (with their apps) for an environment.
func ListAppGroups(c *gin.Context) {
	envID := c.Param("envID")
	groups, err := services.ListGroupedApps(envID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, groups)
}

// ListSpecificGroupedApps returns paginated apps for a specific group.
func ListSpecificGroupedApps(c *gin.Context) {
	groupID := c.Param("groupID")

	page := 1
	pageSize := 10
	search := c.Query("search")

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}

	total, apps, err := services.ListSpecificGroupedApps(c.Request.Context(), groupID, page, pageSize, search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ListAppResponse{
		Items:      apps,
		Pagination: models.BuildPaginationResponse(total, page, pageSize),
	})
}

// CreateAppGroup creates a new app group for an environment.
func CreateAppGroup(c *gin.Context) {
	envID := c.Param("envID")

	var req models.CreateAppGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	group, err := services.CreateAppGroup(envID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, group)
}

// UpdateAppGroup updates an app group's name and description.
func UpdateAppGroup(c *gin.Context) {
	groupID := c.Param("groupID")

	var req models.UpdateAppGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	group, err := services.UpdateAppGroup(groupID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, group)
}

// GetAppGroup returns an app group by ID.
func GetAppGroup(c *gin.Context) {
	groupID := c.Param("groupID")

	group, err := services.GetAppGroup(groupID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, group)
}

// DeleteAppGroup deletes an app group and its memberships.
func DeleteAppGroup(c *gin.Context) {
	groupID := c.Param("groupID")
	if err := services.DeleteAppGroup(groupID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

// AddAppToGroup adds an app to a group.
func AddAppToGroup(c *gin.Context) {
	groupID := c.Param("groupID")
	appID := c.Param("appID")
	if err := services.AddAppToGroup(groupID, appID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

// RemoveAppFromGroup removes an app from a group.
func RemoveAppFromGroup(c *gin.Context) {
	groupID := c.Param("groupID")
	appID := c.Param("appID")
	if err := services.RemoveAppFromGroup(groupID, appID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}
