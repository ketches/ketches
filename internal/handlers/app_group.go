package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// ListAppGroups returns all app groups for a project.
func ListAppGroups(c *gin.Context) {
	projectID := c.Param("projectID")
	groups, err := services.ListAppGroups(projectID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, groups)
}

// ListGroupedApps returns groups with their apps for a given project+env.
func ListGroupedApps(c *gin.Context) {
	projectID := c.Param("projectID")
	envID := c.Param("envID")
	result, err := services.ListGroupedApps(projectID, envID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, result)
}

// CreateAppGroup creates a new app group.
func CreateAppGroup(c *gin.Context) {
	projectID := c.Param("projectID")
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, nil)
		return
	}

	var req models.CreateAppGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	group, err := services.CreateAppGroup(projectID, claims.UserID, &req)
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
