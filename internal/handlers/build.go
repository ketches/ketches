package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListBuilds(c *gin.Context) {
	appID := c.Param("appID")
	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, builds, err := services.ListBuilds(appID, req.Page, req.PageSize)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := make([]models.BuildResponse, 0, len(builds))
	for _, b := range builds {
		res = append(res, services.ToBuildResponse(c.Request.Context(), &b))
	}

	api.Success(c, gin.H{
		"items":      res,
		"pagination": models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func TriggerBuild(c *gin.Context) {
	appID := c.Param("appID")
	claims := api.GetClaims(c)
	userID := ""
	if claims != nil {
		userID = claims.UserID
	}

	var req models.TriggerBuildRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body for simple trigger
		req = models.TriggerBuildRequest{}
	}

	build, err := services.TriggerBuild(c.Request.Context(), appID, userID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, services.ToBuildResponse(c.Request.Context(), build))
}

func GetBuild(c *gin.Context) {
	buildID := c.Param("buildID")

	build, err := services.GetBuild(buildID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	api.Success(c, services.ToBuildResponse(c.Request.Context(), build))
}

func StreamBuildLogs(c *gin.Context) {
	buildID := c.Param("buildID")
	services.StreamBuildLogs(c, buildID)
}

func CancelBuild(c *gin.Context) {
	buildID := c.Param("buildID")

	build, err := services.CancelBuild(buildID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, services.ToBuildResponse(c.Request.Context(), build))
}

func DeployBuild(c *gin.Context) {
	buildID := c.Param("buildID")

	build, err := services.DeployBuild(c.Request.Context(), buildID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, services.ToBuildResponse(c.Request.Context(), build))
}

func RebuildBuild(c *gin.Context) {
	buildID := c.Param("buildID")
	claims := api.GetClaims(c)
	userID := ""
	if claims != nil {
		userID = claims.UserID
	}

	var req models.RebuildRequest
	_ = c.ShouldBindJSON(&req)

	build, err := services.RebuildBuild(c.Request.Context(), buildID, userID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, services.ToBuildResponse(c.Request.Context(), build))
}
