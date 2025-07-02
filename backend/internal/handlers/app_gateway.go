package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

type AppGatewayHandler struct {
	service services.AppGatewayService
}

func NewAppGatewayHandler() *AppGatewayHandler {
	return &AppGatewayHandler{
		service: services.NewAppGatewayService(),
	}
}

// @Summary List App Gateways
// @Description List gateways for an app
// @Tags AppGateway
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Success 200 {object} api.Response{data=[]models.AppGatewayModel}
// @Router /api/v1/apps/{appID}/gateways [get]
func (h *AppGatewayHandler) ListAppGateways(c *gin.Context) {
	resp, err := h.service.ListAppGateways(c, &models.ListAppGatewaysRequest{
		AppID: c.Param("appID"),
	})
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, resp)
}

// @Summary Create App Gateway
// @Description Create a new gateway for an app
// @Tags AppGateway
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param gateway body models.CreateAppGatewayRequest true "Gateway"
// @Success 201 {object} api.Response{data=models.AppGatewayModel}
// @Router /api/v1/apps/{appID}/gateways [post]
func (h *AppGatewayHandler) CreateAppGateway(c *gin.Context) {
	req := &models.CreateAppGatewayRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")
	resp, err := h.service.CreateAppGateway(c, req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Created(c, resp)
}

// @Summary Update App Gateway
// @Description Update a gateway for an app
// @Tags AppGateway
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param gatewayID path string true "Gateway ID"
// @Param gateway body models.UpdateAppGatewayRequest true "Gateway"
// @Success 200 {object} api.Response{data=models.AppGatewayModel}
// @Router /api/v1/apps/{appID}/gateways/{gatewayID} [put]
func (h *AppGatewayHandler) UpdateAppGateway(c *gin.Context) {
	req := &models.UpdateAppGatewayRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")
	req.GatewayID = c.Param("gatewayID")
	resp, err := h.service.UpdateAppGateway(c, req)
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, resp)
}

// @Summary Delete App Gateway
// @Description Delete a gateway for an app
// @Tags AppGateway
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param gatewayIDs body models.DeleteAppGatewaysRequest true "Gateway IDs"
// @Success 204 {object} api.Response{}
// @Router /api/v1/apps/{appID}/gateways [delete]
func (h *AppGatewayHandler) DeleteAppGateways(c *gin.Context) {
	req := &models.DeleteAppGatewaysRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")
	if err := h.service.DeleteAppGateways(c, req); err != nil {
		api.Error(c, err)
		return
	}
	api.NoContent(c)
}
