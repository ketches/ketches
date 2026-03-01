package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListProjects(c *gin.Context) {
	claims := api.GetClaims(c)

	req := models.DefaultPaginationRequest()
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, projects, err := services.ListProjects(claims.UserID, claims.Role, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := []models.ProjectResponse{}
	for _, p := range projects {
		// Find the owner member to populate owner_name
		ownerName := ""
		for _, m := range p.Members {
			if m.ProjectRole == "owner" {
				if m.User.Fullname != "" {
					ownerName = m.User.Fullname
				} else {
					ownerName = m.User.Username
				}
				break
			}
		}
		res = append(res, models.ProjectResponse{
			ID:          p.ID,
			Slug:        p.Slug,
			Name:        p.Name,
			Description: p.Description,
			OwnerName:   ownerName,
		})
	}

	api.Success(c, models.ListProjectResponse{
		Items:      res,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func ListProjectsSimple(c *gin.Context) {
	claims := api.GetClaims(c)
	projects, err := services.ListProjectsSimple(claims.UserID, claims.Role)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := []models.SimpleResponse{}
	for _, p := range projects {
		res = append(res, models.SimpleResponse{
			ID:          p.ID,
			Slug:        p.Slug,
			Name:        p.Name,
			Description: p.Description,
		})
	}

	api.Success(c, res)
}

func CreateProject(c *gin.Context) {
	var req models.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	claims := api.GetClaims(c)
	project, err := services.CreateProject(&req, claims.UserID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, models.ProjectResponse{
		ID:          project.ID,
		Slug:        project.Slug,
		Name:        project.Name,
		Description: project.Description,
	})
}

func GetProject(c *gin.Context) {
	projectID := c.Param("projectID")
	project, err := services.GetProject(projectID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	api.Success(c, models.ProjectResponse{
		ID:          project.ID,
		Slug:        project.Slug,
		Name:        project.Name,
		Description: project.Description,
	})
}

func UpdateProject(c *gin.Context) {
	projectID := c.Param("projectID")
	var req models.UpdateBasicInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	project, err := services.UpdateProject(projectID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ProjectResponse{
		ID:          project.ID,
		Slug:        project.Slug,
		Name:        project.Name,
		Description: project.Description,
	})
}

func DeleteProject(c *gin.Context) {
	projectID := c.Param("projectID")
	if err := services.DeleteProject(projectID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func ListProjectMembers(c *gin.Context) {
	projectID := c.Param("projectID")

	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, members, err := services.ListProjectMembers(projectID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	var res []models.ProjectMemberResponse
	for _, m := range members {
		res = append(res, models.ProjectMemberResponse{
			UserID:      m.UserID,
			Username:    m.User.Username,
			Email:       m.User.Email,
			ProjectRole: m.ProjectRole,
			JoinedAt:    m.CreatedAt,
		})
	}
	api.Success(c, models.ListProjectMemberResponse{
		Items:      res,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func AddProjectMember(c *gin.Context) {
	projectID := c.Param("projectID")
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Role   string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.AddProjectMember(projectID, req.UserID, req.Role); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func RemoveProjectMember(c *gin.Context) {
	projectID := c.Param("projectID")
	userID := c.Query("user_id")
	if err := services.RemoveProjectMember(projectID, userID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}
