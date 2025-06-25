package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

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

// @Summary List Apps Under Env
// @Description List apps under a specific env
// @Tags Env
// @Accept json
// @Produce json
// @Param envID path string true "Env ID"
// @Param query query models.ListAppsRequest false "Query parameters for filtering and pagination"
// @Success 200 {object} api.Response{data=models.ListAppsResponse}
// @Router /api/v1/envs/{envID}/apps [get]
func ListApps(c *gin.Context) {
	var req models.ListAppsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.EnvID = c.Param("envID")

	s := services.NewEnvService()
	resp, err := s.ListApps(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, resp)
}

// @Summary All App Refs Under Env
// @Description Get all apps for refs under a specific env
// @Tags App
// @Accept json
// @Produce json
// @Param envID path string true "Env ID"
// @Param query query models.AllAppRefsRequest false "Query parameters for filtering refs"
// @Success 200 {object} api.Response{data=[]models.AppRef}
// @Router /api/v1/envs/{envID}/apps/refs [get]
func AllAppRefs(c *gin.Context) {
	s := services.NewEnvService()
	refs, err := s.AllAppRefs(c, &models.AllAppRefsRequest{
		EnvID: c.Param("envID"),
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, refs)
}

// @Summary Create App Under Env
// @Description Create a new app under a specific env
// @Tags App
// @Accept json
// @Produce json
// @Param envID path string true "Env ID"
// @Param app body models.CreateAppRequest true "App data"
// @Success 201 {object} api.Response{data=models.AppModel}
// @Router /api/v1/envs/{envID}/apps [post]
func CreateApp(c *gin.Context) {
	var req models.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.EnvID = c.Param("envID")

	s := services.NewEnvService()
	app, err := s.CreateApp(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Created(c, app)
}
