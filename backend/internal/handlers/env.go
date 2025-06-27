package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// @Summary List Envs Under Project
// @Description List envs under a specific project
// @Tags Env
// @Accept json
// @Produce json
// @Param projectID path string true "Project ID"
// @Param query query models.ListEnvsRequest false "Query parameters for filtering and pagination"
// @Success 200 {object} api.Response{data=models.ListEnvsResponse}
// @Router /api/v1/projects/{projectID}/envs [get]
func ListEnvs(c *gin.Context) {
	var req models.ListEnvsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.ProjectID = c.Param("projectID")

	s := services.NewEnvService()
	resp, err := s.ListEnvs(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, resp)
}

// @Summary All Env Refs Under Project
// @Description Get all envs for refs under a specific project
// @Tags Env
// @Accept json
// @Produce json
// @Param projectID path string true "Project ID"
// @Param query query models.AllEnvRefsRequest false "Query parameters for filtering refs"
// @Success 200 {object} api.Response{data=[]models.EnvRef}
// @Router /api/v1/projects/{projectID}/envs/refs [get]
func AllEnvRefs(c *gin.Context) {
	s := services.NewEnvService()
	refs, err := s.AllEnvRefs(c, &models.AllEnvRefsRequest{
		ProjectID: c.Param("projectID"),
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, refs)
}

// @Summary Create Env Under Project
// @Description Create a new env under a specific project
// @Tags Env
// @Accept json
// @Produce json
// @Param projectID path string true "Project ID"
// @Param request body models.CreateEnvRequest true "Create Env Request"
// @Success 201 {object} api.Response{data=models.EnvModel}
// @Router /api/v1/projects/{projectID}/envs [post]
func CreateEnv(c *gin.Context) {
	var req models.CreateEnvRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.ProjectID = c.Param("projectID")

	s := services.NewEnvService()
	env, err := s.CreateEnv(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, env)
}

// @Summary Get Env
// @Description Get env by env ID
// @Tags Env
// @Accept json
// @Produce json
// @Param envID path string true "Env ID"
// @Success 200 {object} api.Response{data=models.EnvModel}
// @Router /api/v1/envs/{envID} [get]
func GetEnv(c *gin.Context) {
	envID := c.Param("envID")
	if envID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Env ID is required"))
		return
	}

	s := services.NewEnvService()
	env, err := s.GetEnv(c, &models.GetEnvRequest{
		EnvID: envID,
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, env)
}

// @Summary Get Env Ref
// @Description Get env ref by env ID
// @Tags Env
// @Accept json
// @Produce json
// @Param envID path string true "Env ID"
// @Success 200 {object} api.Response{data=models.EnvRef}
// @Router /api/v1/envs/{envID}/ref [get]
func GetEnvRef(c *gin.Context) {
	var req models.GetEnvRefRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	s := services.NewEnvService()
	ref, err := s.GetEnvRef(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, ref)
}

// @Summary Update Env
// @Description Update an existing env
// @Tags Env
// @Accept json
// @Produce json
// @Param envID path string true "Env ID"
// @Param request body models.UpdateEnvRequest true "Update Env Request"
// @Success 200 {object} api.Response{data=models.EnvModel}
// @Router /api/v1/envs/{envID} [put]
func UpdateEnv(c *gin.Context) {
	envID := c.Param("envID")
	if envID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Env ID is required"))
		return
	}

	var req models.UpdateEnvRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.EnvID = envID

	s := services.NewEnvService()
	env, err := s.UpdateEnv(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, env)
}

// @Summary Delete Env
// @Description Delete an env by env ID
// @Tags Env
// @Accept json
// @Produce json
// @Param envID path string true "Env ID"
// @Success 204 {object} api.Response
// @Router /api/v1/envs/{envID} [delete]
func DeleteEnv(c *gin.Context) {
	envID := c.Param("envID")
	if envID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Env ID is required"))
		return
	}

	s := services.NewEnvService()
	err := s.DeleteEnv(c, &models.DeleteEnvRequest{
		EnvID: envID,
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.NoContent(c)
}
