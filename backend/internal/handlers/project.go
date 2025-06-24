package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// @Summary List Projects
// @Description List projects
// @Tags Project
// @Accept json
// @Produce json
// @Param query query model.ListProjectsRequest false "Query parameters for filtering and pagination"
// @Success 200 {object} api.Response{data=model.ListProjectResponse}
// @Router /api/v1/projects [get]
// @Security BearerAuth
func ListProjects(c *gin.Context) {
	var req models.ListProjectsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	ps := services.NewProjectService()
	resp, err := ps.ListProjects(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, resp)
}

// @Summary All Project Refs
// @Description Get all projects for refs
// @Tags Project
// @Accept json
// @Produce json
// @Success 200 {object} api.Response{data=[]model.ProjectRef}
// @Router /api/v1/projects/refs [get]
func AllProjectRefs(c *gin.Context) {
	ps := services.NewProjectService()
	refs, err := ps.AllProjectRefs(c)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, refs)
}

// @Summary Get Project
// @Description Get project by project ID
// @Tags Project
// @Accept json
// @Produce json
// @Param projectID path string true "Project ID"
// @Success 200 {object} api.Response{data=model.ProjectModel}
// @Router /api/v1/projects/{projectID} [get]
func GetProject(c *gin.Context) {
	projectID := c.Param("projectID")
	if projectID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Project ID is required"))
		return
	}

	ps := services.NewProjectService()
	project, err := ps.GetProject(c, &models.GetProjectRequest{
		ProjectID: projectID,
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, project)
}

// @Summary Get Project Ref
// @Description Get project ref by project ID
// @Tags Project
// @Accept json
// @Produce json
// @Param projectID path string true "Project ID"
// @Success 200 {object} api.Response{data=model.ProjectRef}
// @Router /api/v1/projects/{projectID}/ref [get]
func GetProjectRef(c *gin.Context) {
	var req models.GetProjectRefRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	as := services.NewProjectService()
	ref, err := as.GetProjectRef(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, ref)
}

// @Summary Create Project
// @Description Create a new project
// @Tags Project
// @Accept json
// @Produce json
// @Param project body model.CreateProjectRequest true "Project data"
// @Success 201 {object} api.Response{data=model.ProjectModel}
// @Router /api/v1/projects [post]
func CreateProject(c *gin.Context) {
	var req models.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	ps := services.NewProjectService()
	project, err := ps.CreateProject(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, project)
}

// @Summary Update Project
// @Description Update an existing project
// @Tags Project
// @Accept json
// @Produce json
// @Param projectID path string true "Project ID"
// @Param project body model.UpdateProjectRequest true "Project data"
// @Success 200 {object} api.Response{data=model.ProjectModel}
// @Router /api/v1/projects/{projectID} [put]
func UpdateProject(c *gin.Context) {
	projectID := c.Param("projectID")
	if projectID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Project ID is required"))
		return
	}

	var req models.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.ProjectID = projectID

	ps := services.NewProjectService()
	project, err := ps.UpdateProject(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, project)
}

// @Summary Delete Project
// @Description Delete a project by project ID
// @Tags Project
// @Accept json
// @Produce json
// @Param projectID path string true "Project ID"
// @Success 204
// @Router /api/v1/projects/{projectID} [delete]
func DeleteProject(c *gin.Context) {
	projectID := c.Param("projectID")
	if projectID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Project ID is required"))
		return
	}

	ps := services.NewProjectService()
	err := ps.DeleteProject(c, &models.DeleteProjectRequest{
		ProjectID: projectID,
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.NoContent(c)
}

// @Summary List Project Members
// @Description List members of a project
// @Tags Project
// @Accept json
// @Produce json
// @Param projectID path string true "Project ID"
// @Param query query model.ListClustersRequest false "Query parameters for filtering and pagination"
// @Success 200 {object} api.Response{data=model.ListProjectMembersResponse}
// @Router /api/v1/projects/{projectID}/members [get]
func ListProjectMembers(c *gin.Context) {
	projectID := c.Param("projectID")
	if projectID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Project ID is required"))
		return
	}

	var req models.ListProjectMembersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.ProjectID = projectID

	ps := services.NewProjectService()
	resp, err := ps.ListProjectMembers(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, resp)
}

// @Summary List Addable Project Members
// @Description List users that can be added to a project
// @Tags Project
// @Accept json
// @Produce json
// @Param projectID path string true "Project ID"
// @Success 200 {object} api.Response{data=[]model.ProjectMemberModel}
// @Router /api/v1/projects/{projectID}/members/addable [get]
func ListAddableProjectMembers(c *gin.Context) {
	projectID := c.Param("projectID")
	if projectID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Project ID is required"))
		return
	}

	ps := services.NewProjectService()
	members, err := ps.ListAddableProjectMembers(c, projectID)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, members)
}

// @Summary Add Project Members
// @Description Add members to a project
// @Tags Project
// @Accept json
// @Produce json
// @Param projectID path string true "Project ID"
// @Param members body model.AddProjectMembersRequest true "Project members data"
// @Success 201 {object} api.Response{data=[]model.ProjectMemberModel}
// @Router /api/v1/projects/{projectID}/members [post]
func AddProjectMembers(c *gin.Context) {
	projectID := c.Param("projectID")
	if projectID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Project ID is required"))
		return
	}

	var req models.AddProjectMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.ProjectID = projectID

	ps := services.NewProjectService()
	err := ps.AddProjectMembers(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, nil)
}

// @Summary Update Project Member
// @Description Update a project member's role
// @Tags Project
// @Accept json
// @Produce json
// @Param projectID path string true "Project ID"
// @Param userID path string true "User ID"
// @Param member body model.UpdateProjectMemberRequest true "Project member data"
// @Success 200 {object} api.Response{data=model.ProjectMemberModel}
// @Router /api/v1/projects/{projectID}/members/{userID} [put]
func UpdateProjectMember(c *gin.Context) {
	projectID := c.Param("projectID")
	if projectID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Project ID is required"))
		return
	}
	userID := c.Param("userID")
	if userID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "User ID is required"))
		return
	}

	var req models.UpdateProjectMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.ProjectID = projectID
	req.UserID = userID

	ps := services.NewProjectService()
	member, err := ps.UpdateProjectMember(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, member)
}

// @Summary Remove Project Member
// @Description Remove a member from a project
// @Tags Project
// @Accept json
// @Produce json
// @Param projectID path string true "Project ID"
// @Success 204
// @Router /api/v1/projects/{projectID}/members [delete]
func RemoveProjectMember(c *gin.Context) {
	projectID := c.Param("projectID")
	if projectID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Project ID is required"))
		return
	}

	var req models.RemoveProjectMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	ps := services.NewProjectService()
	err := ps.RemoveProjectMembers(c, &models.RemoveProjectMembersRequest{
		ProjectID: projectID,
		UserIDs:   req.UserIDs,
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.NoContent(c)
}
