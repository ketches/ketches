package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListDefects(c *gin.Context) {
	projectID := c.Param("projectID")

	var params models.CollabFilterParams
	if err := c.ShouldBindQuery(&params); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	total, items, err := services.ListDefects(projectID, &params)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	resp := make([]models.DefectResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, toDefectResponse(&item))
	}

	api.Success(c, models.ListDefectResponse{
		Items:      resp,
		Pagination: models.BuildPaginationResponse(total, params.Page, params.PageSize),
	})
}

func GetDefect(c *gin.Context) {
	projectID := c.Param("projectID")
	defectID := c.Param("defectID")

	defect, err := services.GetDefect(projectID, defectID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toDefectResponse(defect))
}

func CreateDefect(c *gin.Context) {
	projectID := c.Param("projectID")
	claims := api.GetClaims(c)

	var req models.CreateDefectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	defect, err := services.CreateDefect(projectID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	api.Created(c, toDefectResponse(defect))
}

func UpdateDefect(c *gin.Context) {
	projectID := c.Param("projectID")
	defectID := c.Param("defectID")
	claims := api.GetClaims(c)

	var req models.UpdateDefectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	defect, err := services.UpdateDefect(projectID, defectID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	api.Success(c, toDefectResponse(defect))
}

func DeleteDefect(c *gin.Context) {
	projectID := c.Param("projectID")
	defectID := c.Param("defectID")

	if err := services.DeleteDefect(projectID, defectID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func TransitionDefect(c *gin.Context) {
	projectID := c.Param("projectID")
	defectID := c.Param("defectID")
	claims := api.GetClaims(c)

	var req models.DefectTransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	defect, err := services.TransitionDefect(projectID, defectID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	api.Success(c, toDefectResponse(defect))
}

func toDefectResponse(d *entities.CollabDefect) models.DefectResponse {
	return models.DefectResponse{
		ID:                 d.ID,
		ProjectID:          d.ProjectID,
		RequirementID:      d.RequirementID,
		TaskID:             d.TaskID,
		TestCaseID:         d.TestCaseID,
		TestRunID:          d.TestRunID,
		Title:              d.Title,
		Description:        d.Description,
		Severity:           d.Severity,
		Status:             d.Status,
		AssigneeID:         d.AssigneeID,
		ReproductionSteps:  d.ReproductionSteps,
		FixNote:            d.FixNote,
		RuntimeContextJSON: d.RuntimeContextJSON,
		CreatedBy:          d.CreatedBy,
		UpdatedBy:          d.UpdatedBy,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}
