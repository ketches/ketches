package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/services"
)

// @Summary Get Platform Statistics
// @Description Get platform statistics including total clusters, projects, users, environments, apps, and
// @Tags Platform
// @Accept json
// @Produce json
// @Success 200 {object} api.Response{data=models.PlatformStatisticsModel}
// @Router /api/v1/statistics [get]
func GetPlatformStatistics(c *gin.Context) {
	s := services.NewPlatformService()
	resp, err := s.GetStatistics(c)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, resp)
}
