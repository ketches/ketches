package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListCodeRepositories(c *gin.Context) {
	projectID := c.Param("projectID")

	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, repos, err := services.ListCodeRepositories(projectID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	baseURL := getBaseURL(c)
	res := make([]models.CodeRepositoryResponse, 0, len(repos))
	for i := range repos {
		res = append(res, services.ToCodeRepositoryResponse(&repos[i], baseURL))
	}

	api.Success(c, models.ListCodeRepositoryResponse{
		Items:      res,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func ListCodeRepositoriesSimple(c *gin.Context) {
	projectID := c.Param("projectID")
	repos, err := services.ListCodeRepositoriesSimple(projectID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := []models.SimpleResponse{}
	for _, repo := range repos {
		res = append(res, models.SimpleResponse{
			ID:   repo.ID,
			Slug: repo.Slug,
			Name: repo.Name,
		})
	}

	api.Success(c, res)
}

func CreateCodeRepository(c *gin.Context) {
	projectID := c.Param("projectID")
	var req models.CreateCodeRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	repo, err := services.CreateCodeRepository(projectID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, services.ToCodeRepositoryResponse(repo, getBaseURL(c)))
}

func GetCodeRepository(c *gin.Context) {
	repoID := c.Param("repoID")
	repo, err := services.GetCodeRepository(repoID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	api.Success(c, services.ToCodeRepositoryResponse(repo, getBaseURL(c)))
}

func UpdateCodeRepository(c *gin.Context) {
	repoID := c.Param("repoID")
	var req models.UpdateCodeRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	repo, err := services.UpdateCodeRepository(repoID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, services.ToCodeRepositoryResponse(repo, getBaseURL(c)))
}

func DeleteCodeRepository(c *gin.Context) {
	repoID := c.Param("repoID")
	if err := services.DeleteCodeRepository(repoID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func ListCodeRepositoryBuilds(c *gin.Context) {
	repoID := c.Param("repoID")
	builds, err := services.ListBuildsByCodeRepository(repoID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	res := make([]models.BuildResponse, 0, len(builds))
	for i := range builds {
		res = append(res, services.ToBuildResponse(c.Request.Context(), &builds[i]))
	}
	api.Success(c, res)
}

func ListCodeRepositoryDeployments(c *gin.Context) {
	repoID := c.Param("repoID")
	builds, err := services.ListDeploymentsByCodeRepository(repoID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	res := make([]models.BuildResponse, 0, len(builds))
	for i := range builds {
		res = append(res, services.ToBuildResponse(c.Request.Context(), &builds[i]))
	}
	api.Success(c, res)
}

func TriggerCodeRepositoryBuild(c *gin.Context) {
	repoID := c.Param("repoID")
	userID := ""
	if claims := api.GetClaims(c); claims != nil {
		userID = claims.UserID
	}
	var req models.TriggerCodeRepositoryBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	build, err := services.TriggerCodeRepositoryBuild(repoID, userID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, services.ToBuildResponse(c.Request.Context(), build))
}

func GetCodeRepositoryBuild(c *gin.Context) {
	repoID := c.Param("repoID")
	buildID := c.Param("buildID")
	build, err := services.GetBuildByCodeRepository(repoID, buildID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	api.Success(c, services.ToBuildResponse(c.Request.Context(), build))
}

func StreamCodeRepositoryBuildLogs(c *gin.Context) {
	buildID := c.Param("buildID")
	services.StreamBuildLogs(c, buildID)
}

func CancelCodeRepositoryBuild(c *gin.Context) {
	buildID := c.Param("buildID")
	build, err := services.CancelBuild(buildID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, services.ToBuildResponse(c.Request.Context(), build))
}

func DeployCodeRepositoryBuild(c *gin.Context) {
	repoID := c.Param("repoID")
	buildID := c.Param("buildID")
	var req models.DeployCodeRepositoryBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	build, app, err := services.DeployCodeRepositoryBuild(repoID, buildID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, gin.H{
		"build": services.ToBuildResponse(c.Request.Context(), build),
		"app":   app,
	})
}

func ListCodeRepositoryContainerRegistries(c *gin.Context) {
	repoID := c.Param("repoID")
	registries, err := services.ListAvailableContainerRegistriesForCodeRepository(repoID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	res := make([]models.ContainerRegistryResponse, 0, len(registries))
	for i := range registries {
		res = append(res, services.ToContainerRegistryResponse(&registries[i]))
	}
	api.Success(c, res)
}

// Build configs under code repository

func ListCodeRepositoryBuildConfigs(c *gin.Context) {
	repoID := c.Param("repoID")
	configs, err := services.ListCodeRepositoryBuildConfigs(repoID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	res := make([]models.CodeRepositoryBuildConfigResponse, 0, len(configs))
	for i := range configs {
		res = append(res, services.ToCodeRepositoryBuildConfigResponse(&configs[i]))
	}
	api.Success(c, res)
}

func CreateCodeRepositoryBuildConfig(c *gin.Context) {
	repoID := c.Param("repoID")
	var req models.CreateCodeRepositoryBuildConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	cfg, err := services.CreateCodeRepositoryBuildConfig(repoID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, services.ToCodeRepositoryBuildConfigResponse(cfg))
}

func GetCodeRepositoryBuildConfig(c *gin.Context) {
	repoID := c.Param("repoID")
	configID := c.Param("configID")
	cfg, err := services.GetCodeRepositoryBuildConfig(configID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	if cfg.CodeRepositoryID != repoID {
		api.Error(c, http.StatusNotFound, fmt.Errorf("build config not found"))
		return
	}
	api.Success(c, services.ToCodeRepositoryBuildConfigResponse(cfg))
}

func UpdateCodeRepositoryBuildConfig(c *gin.Context) {
	repoID := c.Param("repoID")
	configID := c.Param("configID")
	cfg, err := services.GetCodeRepositoryBuildConfig(configID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	if cfg.CodeRepositoryID != repoID {
		api.Error(c, http.StatusNotFound, fmt.Errorf("build config not found"))
		return
	}
	var req models.UpdateCodeRepositoryBuildConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	updated, err := services.UpdateCodeRepositoryBuildConfig(configID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, services.ToCodeRepositoryBuildConfigResponse(updated))
}

func DeleteCodeRepositoryBuildConfig(c *gin.Context) {
	repoID := c.Param("repoID")
	configID := c.Param("configID")
	cfg, err := services.GetCodeRepositoryBuildConfig(configID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	if cfg.CodeRepositoryID != repoID {
		api.Error(c, http.StatusNotFound, fmt.Errorf("build config not found"))
		return
	}
	if err := services.DeleteCodeRepositoryBuildConfig(configID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func TestCodeRepositoryGit(c *gin.Context) {
	var req models.TestGitConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	result := services.TestGitConnection(&req)
	api.Success(c, result)
}

func GetCodeRepositoryTopology(c *gin.Context) {
	repoID := c.Param("repoID")
	topology, err := services.GetCodeRepositoryTopology(repoID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, topology)
}

func ListCodeRepositoryRefs(c *gin.Context) {
	repoID := c.Param("repoID")
	refs, err := services.ListCodeRepositoryRefs(repoID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, models.ListGitRefsResponse{Refs: refs})
}

func getBaseURL(c *gin.Context) string {
	// Prefer X-Forwarded-Proto/Host for webhook URL
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "https"
		if c.Request.TLS == nil {
			scheme = "http"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return scheme + "://" + host
}
