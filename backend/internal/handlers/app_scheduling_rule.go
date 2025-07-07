package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// @Summary Get App Scheduling Rule
// @Description Get the scheduling rule of an app
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Success 200 {object} api.Response{data=models.AppSchedulingRuleModel}
// @Router /api/v1/apps/{appID}/scheduling-rule [get]
func GetAppSchedulingRule(c *gin.Context) {
	var req models.GetAppSchedulingRuleRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	s := services.NewAppSchedulingRuleService()
	rule, err := s.GetAppSchedulingRule(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	if rule == nil {
		api.Success(c, nil)
		return
	}

	api.Success(c, rule)
}

// @Summary Set App Scheduling Rule
// @Description Set the scheduling rule of an app
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param schedulingRule body models.SetAppSchedulingRuleRequest true "Set app scheduling rule"
// @Success 200 {object} api.Response{data=models.AppSchedulingRuleModel}
// @Router /api/v1/apps/{appID}/scheduling-rule [put]
func SetAppSchedulingRule(c *gin.Context) {
	var req models.SetAppSchedulingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = c.Param("appID")

	s := services.NewAppSchedulingRuleService()
	rule, err := s.SetAppSchedulingRule(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, rule)
}

// @Summary Delete App Scheduling Rule
// @Description Delete the scheduling rule of an app
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Success 204 {object} api.Response{}
// @Router /api/v1/apps/{appID}/scheduling-rule [delete]
func DeleteAppSchedulingRule(c *gin.Context) {
	appID := c.Param("appID")
	if appID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "App ID is required"))
		return
	}

	s := services.NewAppSchedulingRuleService()
	if err := s.DeleteAppSchedulingRule(c, appID); err != nil {
		api.Error(c, err)
		return
	}

	api.NoContent(c)
}