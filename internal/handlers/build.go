package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListAppBuilds(c *gin.Context) {
	appID := c.Param("appID")
	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, builds, err := services.ListAppBuilds(appID, req.Page, req.PageSize)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, gin.H{
		"items":      builds,
		"pagination": models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}
