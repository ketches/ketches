package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListTemplates(c *gin.Context) {
	projectID := c.Param("projectID")
	templates, err := services.ListTemplates(projectID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	res := make([]models.TemplateResponse, 0, len(templates))
	for i := range templates {
		res = append(res, services.ToTemplateResponse(&templates[i]))
	}
	api.Success(c, res)
}

func CreateTemplate(c *gin.Context) {
	projectID := c.Param("projectID")
	var req models.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	tmpl, err := services.CreateTemplate(projectID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, services.ToTemplateResponse(tmpl))
}

func GetTemplate(c *gin.Context) {
	templateID := c.Param("templateID")
	tmpl, err := services.GetTemplate(templateID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	api.Success(c, services.ToTemplateResponse(tmpl))
}

func UpdateTemplate(c *gin.Context) {
	templateID := c.Param("templateID")
	var req models.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	tmpl, err := services.UpdateTemplate(templateID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, services.ToTemplateResponse(tmpl))
}

func DeleteTemplate(c *gin.Context) {
	templateID := c.Param("templateID")
	if err := services.DeleteTemplate(templateID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}
