package handlers

import (
	"net/http"

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

	req := models.DefaultPaginationRequest()
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, apps, err := services.ListSpecificGroupedApps(c.Request.Context(), groupID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ListAppResponse{
		Items:      apps,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
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
