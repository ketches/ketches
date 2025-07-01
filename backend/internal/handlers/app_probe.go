package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

type AppProbeHandler struct {
	svc services.AppProbeService
}

func NewAppProbeHandler() *AppProbeHandler {
	return &AppProbeHandler{
		svc: services.NewAppProbeService(),
	}
}

// @Summary List App Probes
// @Description List probes for an app
// @Tags AppProbe
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Success 200 {object} api.Response{data=[]models.AppProbeModel}
// @Router /api/v1/apps/{appID}/probes [get]
func (h *AppProbeHandler) ListAppProbes(c *gin.Context) {
	var req models.ListAppProbesRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	probes, err := h.svc.ListAppProbes(c.Request.Context(), &req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, probes)
}

// @Summary Create App Probe
// @Description Create a new probe for an app
// @Tags AppProbe
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param probe body models.CreateAppProbeRequest true "Probe"
// @Success 201 {object} api.Response{data=models.AppProbeModel}
// @Router /api/v1/apps/{appID}/probes [post]
func (h *AppProbeHandler) CreateAppProbe(c *gin.Context) {
	var req models.CreateAppProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")
	probe, err := h.svc.CreateAppProbe(c.Request.Context(), &req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Created(c, probe)
}

// @Summary Update App Probe
// @Description Update a probe for an app
// @Tags AppProbe
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param probeID path string true "Probe ID"
// @Param probe body models.UpdateAppProbeRequest true "Probe"
// @Success 200 {object} api.Response{data=models.AppProbeModel}
// @Router /api/v1/apps/{appID}/probes/{probeID} [put]
func (h *AppProbeHandler) UpdateAppProbe(c *gin.Context) {
	var req models.UpdateAppProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")
	req.ProbeID = c.Param("probeID")
	probe, err := h.svc.UpdateAppProbe(c.Request.Context(), &req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, probe)
}

// @Summary Delete App Probe
// @Description Delete a probe for an app
// @Tags AppProbe
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param probeID path string true "Probe ID"
// @Success 204 {object} api.Response{}
// @Router /api/v1/apps/{appID}/probes/{probeID} [delete]
func (h *AppProbeHandler) DeleteAppProbe(c *gin.Context) {
	var req models.DeleteAppProbeRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	if err := h.svc.DeleteAppProbe(c.Request.Context(), &req); err != nil {
		api.Error(c, err)
		return
	}
	api.NoContent(c)
}
