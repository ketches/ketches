package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/middlewares"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func projectMemberManagementRouter(userID, systemRole string) *gin.Engine {
	router := gin.New()
	router.Use(projectClaimsMiddleware(userID, userID, systemRole))
	router.POST(
		"/api/v1/projects/:projectID/members",
		middlewares.RequireProjectRole(app.ProjectRoleOwner),
		InviteProjectMembers,
	)
	router.DELETE(
		"/api/v1/projects/:projectID/members",
		middlewares.RequireProjectRole(app.ProjectRoleOwner),
		RemoveProjectMember,
	)
	return router
}

func seedProjectMemberAuthorizationProject(t *testing.T) {
	t.Helper()
	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)
	for _, member := range []entities.ProjectMember{
		{ID: "owner-member", ProjectID: "project-1", UserID: "owner-1", ProjectRole: app.ProjectRoleOwner},
		{ID: "developer-member", ProjectID: "project-1", UserID: "developer-1", ProjectRole: app.ProjectRoleDeveloper},
		{ID: "viewer-member", ProjectID: "project-1", UserID: "viewer-1", ProjectRole: app.ProjectRoleViewer},
	} {
		require.NoError(t, db.DB.Create(&member).Error)
	}
}

func projectMemberRequest(t *testing.T, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

func TestProjectMemberManagementRejectsDeveloperSelfPromotion(t *testing.T) {
	setupProjectHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	seedProjectMemberAuthorizationProject(t)

	res := projectMemberRequest(
		t,
		projectMemberManagementRouter("developer-1", app.UserRoleUser),
		http.MethodPost,
		"/api/v1/projects/project-1/members",
		`{"user_ids":["developer-1"],"role":"owner"}`,
	)

	assert.Equal(t, http.StatusForbidden, res.Code)
	var developer entities.ProjectMember
	require.NoError(t, db.DB.First(&developer, "project_id = ? AND user_id = ?", "project-1", "developer-1").Error)
	assert.Equal(t, app.ProjectRoleDeveloper, developer.ProjectRole)
}

func TestProjectMemberManagementRejectsDeveloperRemoval(t *testing.T) {
	setupProjectHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	seedProjectMemberAuthorizationProject(t)

	res := projectMemberRequest(
		t,
		projectMemberManagementRouter("developer-1", app.UserRoleUser),
		http.MethodDelete,
		"/api/v1/projects/project-1/members?user_id=viewer-1",
		"",
	)

	assert.Equal(t, http.StatusForbidden, res.Code)
	var viewerCount int64
	require.NoError(t, db.DB.Model(&entities.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", "project-1", "viewer-1").Count(&viewerCount).Error)
	assert.Equal(t, int64(1), viewerCount)
}

func TestProjectMemberManagementAllowsOwner(t *testing.T) {
	setupProjectHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	seedProjectMemberAuthorizationProject(t)
	router := projectMemberManagementRouter("owner-1", app.UserRoleUser)

	inviteRes := projectMemberRequest(
		t,
		router,
		http.MethodPost,
		"/api/v1/projects/project-1/members",
		`{"user_ids":["invitee-1"],"role":"developer"}`,
	)
	require.Equal(t, http.StatusNoContent, inviteRes.Code)

	var invitation entities.Notification
	require.NoError(t, db.DB.First(&invitation, "recipient_id = ? AND resource_id = ?", "invitee-1", "project-1").Error)
	assert.Equal(t, "pending", invitation.Status)
	assert.JSONEq(t, `{"role":"developer"}`, invitation.ActionData)

	updateRes := projectMemberRequest(
		t,
		router,
		http.MethodPost,
		"/api/v1/projects/project-1/members",
		`{"user_ids":["developer-1"],"role":"viewer"}`,
	)
	require.Equal(t, http.StatusNoContent, updateRes.Code)

	var developer entities.ProjectMember
	require.NoError(t, db.DB.First(&developer, "project_id = ? AND user_id = ?", "project-1", "developer-1").Error)
	assert.Equal(t, app.ProjectRoleViewer, developer.ProjectRole)

	removeRes := projectMemberRequest(
		t,
		router,
		http.MethodDelete,
		"/api/v1/projects/project-1/members?user_id=viewer-1",
		"",
	)
	require.Equal(t, http.StatusNoContent, removeRes.Code)

	var viewerCount int64
	require.NoError(t, db.DB.Model(&entities.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", "project-1", "viewer-1").Count(&viewerCount).Error)
	assert.Zero(t, viewerCount)
}

func TestProjectMemberManagementAllowsSystemAdmin(t *testing.T) {
	setupProjectHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	seedProjectMemberAuthorizationProject(t)

	res := projectMemberRequest(
		t,
		projectMemberManagementRouter("admin-1", app.UserRoleAdmin),
		http.MethodPost,
		"/api/v1/projects/project-1/members",
		`{"user_ids":["viewer-1"],"role":"developer"}`,
	)

	require.Equal(t, http.StatusNoContent, res.Code)
	var viewer entities.ProjectMember
	require.NoError(t, db.DB.First(&viewer, "project_id = ? AND user_id = ?", "project-1", "viewer-1").Error)
	assert.Equal(t, app.ProjectRoleDeveloper, viewer.ProjectRole)
}

func TestInviteProjectMembersRejectsInvalidRole(t *testing.T) {
	setupProjectHandlerTestDB(t)
	gin.SetMode(gin.TestMode)
	seedProjectMemberAuthorizationProject(t)

	res := projectMemberRequest(
		t,
		projectMemberManagementRouter("owner-1", app.UserRoleUser),
		http.MethodPost,
		"/api/v1/projects/project-1/members",
		`{"user_ids":["viewer-1"],"role":"administrator"}`,
	)

	assert.Equal(t, http.StatusBadRequest, res.Code)
}
