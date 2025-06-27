package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// @Summary List App Volumes
// @Description List storage volumes for an app
// @Tags AppVolume
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Success 200 {object} api.Response{data=[]models.AppVolumeModel}
// @Router /api/v1/apps/{appID}/volumes [get]
func ListAppVolumes(c *gin.Context) {
	var req models.ListAppVolumesRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	s := services.NewAppVolumeService()
	resp, err := s.ListAppVolumes(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, resp)
}

// @Summary Create App Volume
// @Description Create a new storage volume for an app
// @Tags AppVolume
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param volume body models.CreateAppVolumeRequest true "Volume"
// @Success 201 {object} api.Response{data=models.AppVolumeModel}
// @Router /api/v1/apps/{appID}/volumes [post]
func CreateAppVolume(c *gin.Context) {
	var req models.CreateAppVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")
	s := services.NewAppVolumeService()
	volume, err := s.CreateAppVolume(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Created(c, volume)
}

// @Summary Update App Volume
// @Description Update a storage volume for an app
// @Tags AppVolume
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param volume body models.UpdateAppVolumeRequest true "Volume"
// @Success 200 {object} api.Response{data=models.AppVolumeModel}
// @Router /api/v1/apps/{appID}/volumes/{volumeID} [put]
func UpdateAppVolume(c *gin.Context) {
	var req models.UpdateAppVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")
	req.VolumeID = c.Param("volumeID")

	s := services.NewAppVolumeService()
	volume, err := s.UpdateAppVolume(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, volume)
}

// @Summary Delete App Volumes
// @Description Delete storage volumes for an app
// @Tags AppVolume
// @Accept json
// @Produce json
// @Param req body models.DeleteAppVolumesRequest true "Volume IDs"
// @Success 204 {object} api.Response{}
// @Router /api/v1/apps/volumes [delete]
func DeleteAppVolumes(c *gin.Context) {
	var req models.DeleteAppVolumesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")

	s := services.NewAppVolumeService()
	if err := s.DeleteAppVolumes(c, &req); err != nil {
		api.Error(c, err)
		return
	}
	api.NoContent(c)
}
