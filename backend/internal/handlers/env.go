package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// @Summary List Envs
// @Description List envs
// @Tags Env
// @Accept json
// @Produce json
// @Param query query model.ListEnvsRequest false "Query parameters for filtering and pagination"
// @Success 200 {object} api.Response{data=model.ListEnvsResponse}
// @Router /api/v1/envs [get]
func ListEnvs(c *gin.Context) {
	var req models.ListEnvsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	es := services.NewEnvService()
	resp, err := es.ListEnvs(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, resp)
}

// @Summary All Env Refs Under Cluster
// @Description Get all envs for refs under a specific cluster
// @Tags Env
// @Accept json
// @Produce json
// @Param query query model.AllEnvRefsRequest false "Query parameters for filtering refs"
// @Success 200 {object} api.Response{data=[]model.EnvRef}
// @Router /api/v1/envs/refs [get]
func AllEnvRefs(c *gin.Context) {
	var req models.AllEnvRefsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	as := services.NewEnvService()
	refs, err := as.AllEnvRefs(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, refs)
}

// @Summary Get Env
// @Description Get env by env ID
// @Tags Env
// @Accept json
// @Produce json
// @Param envID path string true "Env ID"
// @Success 200 {object} api.Response{data=model.EnvModel}
// @Router /api/v1/envs/{envID} [get]
func GetEnv(c *gin.Context) {
	envID := c.Param("envID")
	if envID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Env ID is required"))
		return
	}

	es := services.NewEnvService()
	env, err := es.GetEnv(c, &models.GetEnvRequest{
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
// @Success 200 {object} api.Response{data=model.EnvRef}
// @Router /api/v1/envs/{envID}/ref [get]
func GetEnvRef(c *gin.Context) {
	var req models.GetEnvRefRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	as := services.NewEnvService()
	ref, err := as.GetEnvRef(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, ref)
}

// @Summary Create Env
// @Description Create a new env
// @Tags Env
// @Accept json
// @Produce json
// @Param request body model.CreateEnvRequest true "Create Env Request"
// @Success 201 {object} api.Response{data=model.EnvModel}
// @Router /api/v1/envs [post]
func CreateEnv(c *gin.Context) {
	var req models.CreateEnvRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	es := services.NewEnvService()
	env, err := es.CreateEnv(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, env)
}

// @Summary Update Env
// @Description Update an existing env
// @Tags Env
// @Accept json
// @Produce json
// @Param envID path string true "Env ID"
// @Param request body model.UpdateEnvRequest true "Update Env Request"
// @Success 200 {object} api.Response{data=model.EnvModel}
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

	es := services.NewEnvService()
	env, err := es.UpdateEnv(c, &req)
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

	es := services.NewEnvService()
	err := es.DeleteEnv(c, &models.DeleteEnvRequest{
		EnvID: envID,
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.NoContent(c)
}
