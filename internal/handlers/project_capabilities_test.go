package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/middlewares"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func projectCapabilitiesRouter(userID, systemRole string) *gin.Engine {
	router := gin.New()
	router.Use(projectClaimsMiddleware(userID, userID, systemRole))
	router.GET(
		"/api/v1/projects/:projectID/capabilities",
		middlewares.RequireProjectRole(app.ProjectRoleViewer),
		GetProjectCapabilities,
	)
	return router
}

func TestGetProjectCapabilitiesUsesExactMembershipLookup(t *testing.T) {
	setupProjectHandlerTestDB(t)
	gin.SetMode(gin.TestMode)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)

	for i := 0; i < 15; i++ {
		require.NoError(t, db.DB.Create(&entities.ProjectMember{
			ID:          fmt.Sprintf("member-%02d", i),
			ProjectID:   "project-1",
			UserID:      fmt.Sprintf("user-%02d", i),
			ProjectRole: app.ProjectRoleViewer,
		}).Error)
	}
	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID:          "target-member",
		ProjectID:   "project-1",
		UserID:      "target-user",
		ProjectRole: app.ProjectRoleDeveloper,
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/capabilities", nil)
	res := httptest.NewRecorder()
	projectCapabilitiesRouter("target-user", app.UserRoleUser).ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	var response struct {
		Data models.ProjectCapabilitiesResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &response))
	assert.Equal(t, app.ProjectRoleDeveloper, response.Data.ProjectRole)
	assert.Equal(t, models.ProjectCapabilities{Read: true, Write: true, Manage: false}, response.Data.Capabilities)
}

func TestGetProjectCapabilitiesMapsAdminToOwner(t *testing.T) {
	setupProjectHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/capabilities", nil)
	res := httptest.NewRecorder()
	projectCapabilitiesRouter("admin-1", app.UserRoleAdmin).ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	var response struct {
		Data models.ProjectCapabilitiesResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &response))
	assert.Equal(t, app.ProjectRoleOwner, response.Data.ProjectRole)
	assert.Equal(t, models.ProjectCapabilities{Read: true, Write: true, Manage: true}, response.Data.Capabilities)
}

func TestGetProjectCapabilitiesViewerIsReadOnly(t *testing.T) {
	setupProjectHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectMember{
		ID:          "member-1",
		ProjectID:   "project-1",
		UserID:      "viewer-1",
		ProjectRole: app.ProjectRoleViewer,
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/capabilities", nil)
	res := httptest.NewRecorder()
	projectCapabilitiesRouter("viewer-1", app.UserRoleUser).ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	var response struct {
		Data models.ProjectCapabilitiesResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &response))
	assert.Equal(t, app.ProjectRoleViewer, response.Data.ProjectRole)
	assert.Equal(t, models.ProjectCapabilities{Read: true, Write: false, Manage: false}, response.Data.Capabilities)
}

func TestGetProjectCapabilitiesRejectsNonMember(t *testing.T) {
	setupProjectHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/project-1/capabilities", nil)
	res := httptest.NewRecorder()
	projectCapabilitiesRouter("user-1", app.UserRoleUser).ServeHTTP(res, req)

	assert.Equal(t, http.StatusForbidden, res.Code)
}
