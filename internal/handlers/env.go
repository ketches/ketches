package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListEnvs(c *gin.Context) {
	projectID := c.Param("projectID")

	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, envs, err := services.ListEnvs(projectID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := []models.EnvResponse{}
	for _, e := range envs {
		res = append(res, models.EnvResponse{
			ID:               e.ID,
			Slug:             e.Slug,
			Name:             e.Name,
			Description:      e.Description,
			ProjectID:        e.ProjectID,
			ClusterID:        e.ClusterID,
			ClusterNamespace: e.ClusterNamespace,
			IsBuildEnv:       e.IsBuildEnv,
			CreatedAt:        e.CreatedAt,
		})
	}

	api.Success(c, models.ListEnvResponse{
		Items:      res,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func ListEnvsSimple(c *gin.Context) {
	projectID := c.Param("projectID")
	envs, err := services.ListEnvsSimple(projectID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, envs)
}

func CreateEnv(c *gin.Context) {
	projectID := c.Param("projectID")
	var req models.CreateEnvRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	env, err := services.CreateEnv(projectID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, models.EnvResponse{
		ID:               env.ID,
		Slug:             env.Slug,
		Name:             env.Name,
		Description:      env.Description,
		ProjectID:        env.ProjectID,
		ClusterID:        env.ClusterID,
		ClusterNamespace: env.ClusterNamespace,
		IsBuildEnv:       env.IsBuildEnv,
	})
}

func GetEnv(c *gin.Context) {
	envID := c.Param("envID")
	env, err := services.GetEnv(envID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	api.Success(c, models.EnvResponse{
		ID:               env.ID,
		Slug:             env.Slug,
		Name:             env.Name,
		Description:      env.Description,
		ProjectID:        env.ProjectID,
		ClusterID:        env.ClusterID,
		ClusterNamespace: env.ClusterNamespace,
		IsBuildEnv:       env.IsBuildEnv,
	})
}

func UpdateEnv(c *gin.Context) {
	envID := c.Param("envID")
	var req models.CreateEnvRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	env, err := services.UpdateEnv(envID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.EnvResponse{
		ID:               env.ID,
		Slug:             env.Slug,
		Name:             env.Name,
		Description:      env.Description,
		ProjectID:        env.ProjectID,
		ClusterID:        env.ClusterID,
		ClusterNamespace: env.ClusterNamespace,
		IsBuildEnv:       env.IsBuildEnv,
	})
}

func UpdateEnvBasic(c *gin.Context) {
	envID := c.Param("envID")
	var req models.UpdateBasicInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	env, err := services.UpdateEnvBasic(envID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.EnvResponse{
		ID:               env.ID,
		Slug:             env.Slug,
		Name:             env.Name,
		Description:      env.Description,
		ProjectID:        env.ProjectID,
		ClusterID:        env.ClusterID,
		ClusterNamespace: env.ClusterNamespace,
		IsBuildEnv:       env.IsBuildEnv,
	})
}

func SetBuildEnv(c *gin.Context) {
	envID := c.Param("envID")

	env, err := services.SetBuildEnv(envID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.EnvResponse{
		ID:               env.ID,
		Slug:             env.Slug,
		Name:             env.Name,
		Description:      env.Description,
		ProjectID:        env.ProjectID,
		ClusterID:        env.ClusterID,
		ClusterNamespace: env.ClusterNamespace,
		IsBuildEnv:       env.IsBuildEnv,
	})
}

func UnsetBuildEnv(c *gin.Context) {
	envID := c.Param("envID")

	env, err := services.UnsetBuildEnv(envID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.EnvResponse{
		ID:               env.ID,
		Slug:             env.Slug,
		Name:             env.Name,
		Description:      env.Description,
		ProjectID:        env.ProjectID,
		ClusterID:        env.ClusterID,
		ClusterNamespace: env.ClusterNamespace,
		IsBuildEnv:       env.IsBuildEnv,
	})
}

func DeleteEnv(c *gin.Context) {
	envID := c.Param("envID")
	if err := services.DeleteEnv(envID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}
