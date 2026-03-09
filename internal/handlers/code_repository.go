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
	res := make([]models.CodeRepositoryResponse, 0, len(repos))
	for i := range repos {
		res = append(res, services.ToCodeRepositoryResponse(&repos[i]))
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

	api.Success(c, repos)
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
	api.Created(c, services.ToCodeRepositoryRowResponse(repo))
}

func GetCodeRepository(c *gin.Context) {
	repoID := c.Param("repoID")
	repo, err := services.GetCodeRepository(repoID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	api.Success(c, services.ToCodeRepositoryRowResponse(repo))
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
	api.Success(c, services.ToCodeRepositoryRowResponse(repo))
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

func ListBuildDeployments(c *gin.Context) {
	repoID := c.Param("repoID")
	deployments, err := services.ListDeployments(repoID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, deployments)
}

func ListDeployedAppsByEnvironment(c *gin.Context) {
	repoID := c.Param("repoID")
	buildSettingID := c.Param("settingID")
	envID := c.Query("env_id")
	if envID == "" {
		api.Error(c, http.StatusBadRequest, fmt.Errorf("env_id is required"))
		return
	}

	setting, err := services.GetBuildSetting(buildSettingID)
	if err != nil {
		api.Error(c, http.StatusNotFound, fmt.Errorf("build setting not found"))
		return
	}
	if setting.CodeRepositoryID == nil || *setting.CodeRepositoryID != repoID {
		api.Error(c, http.StatusForbidden, fmt.Errorf("build setting does not belong to this code repository"))
		return
	}

	apps, err := services.ListDeployedAppsByEnvironmentAndBuildSetting(envID, buildSettingID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, apps)
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
	build, appCtx, err := services.DeployCodeRepositoryBuild(c.Request.Context(), repoID, buildID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, gin.H{
		"build": services.ToBuildResponse(c.Request.Context(), build),
		"app":   services.ToAppResponse(c.Request.Context(), appCtx),
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

func ListRepoBuildSettings(c *gin.Context) {
	repoID := c.Param("repoID")
	settings, err := services.ListRepoBuildSettings(repoID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	res := make([]models.BuildSettingResponse, 0, len(settings))
	for i := range settings {
		res = append(res, services.ToBuildSettingResponse(&settings[i]))
	}
	api.Success(c, res)
}

func CreateRepoBuildSetting(c *gin.Context) {
	repoID := c.Param("repoID")
	var req models.CreateBuildSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	s, err := services.CreateRepoBuildSetting(repoID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, services.ToBuildSettingResponse(s))
}

func GetRepoBuildSetting(c *gin.Context) {
	settingID := c.Param("settingID")
	s, err := services.GetBuildSetting(settingID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	api.Success(c, services.ToBuildSettingResponse(s))
}

func UpdateRepoBuildSetting(c *gin.Context) {
	settingID := c.Param("settingID")
	var req models.UpdateRepoBuildSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	s, err := services.UpdateRepoBuildSetting(settingID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, services.ToBuildSettingResponse(s))
}

func DeleteRepoBuildSetting(c *gin.Context) {
	repoID := c.Param("repoID")
	settingID := c.Param("settingID")
	s, err := services.GetBuildSetting(settingID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	if s.CodeRepositoryID == nil || *s.CodeRepositoryID != repoID {
		api.Error(c, http.StatusForbidden, fmt.Errorf("setting does not belong to this repository"))
		return
	}
	if err := services.DeleteRepoBuildSetting(settingID); err != nil {
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
