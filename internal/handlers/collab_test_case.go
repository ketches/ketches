package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

func ListTestCases(c *gin.Context) {
	projectID := c.Param("projectID")

	var params models.CollabFilterParams
	if err := c.ShouldBindQuery(&params); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	total, items, err := services.ListTestCases(projectID, &params)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	resp := make([]models.TestCaseResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, toTestCaseResponse(&item))
	}

	api.Success(c, models.ListTestCaseResponse{
		Items:      resp,
		Pagination: models.BuildPaginationResponse(total, params.Page, params.PageSize),
	})
}

func GetTestCase(c *gin.Context) {
	projectID := c.Param("projectID")
	testCaseID := c.Param("testCaseID")

	testCase, err := services.GetTestCase(projectID, testCaseID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toTestCaseResponse(testCase))
}

func CreateTestCase(c *gin.Context) {
	projectID := c.Param("projectID")
	claims := api.GetClaims(c)

	var req models.CreateTestCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	testCase, err := services.CreateTestCase(projectID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	api.Created(c, toTestCaseResponse(testCase))
}

func UpdateTestCase(c *gin.Context) {
	projectID := c.Param("projectID")
	testCaseID := c.Param("testCaseID")
	claims := api.GetClaims(c)

	var req models.UpdateTestCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	testCase, err := services.UpdateTestCase(projectID, testCaseID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	api.Success(c, toTestCaseResponse(testCase))
}

func DeleteTestCase(c *gin.Context) {
	projectID := c.Param("projectID")
	testCaseID := c.Param("testCaseID")

	if err := services.DeleteTestCase(projectID, testCaseID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func CreateTestRun(c *gin.Context) {
	projectID := c.Param("projectID")
	testCaseID := c.Param("testCaseID")
	claims := api.GetClaims(c)

	var req models.CreateTestRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	testRun, err := services.CreateTestRun(projectID, testCaseID, claims.UserID, &req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	api.Created(c, toTestRunResponse(testRun))
}

func toTestCaseResponse(tc *entities.CollabTestCase) models.TestCaseResponse {
	return models.TestCaseResponse{
		ID:             tc.ID,
		ProjectID:      tc.ProjectID,
		RequirementID:  tc.RequirementID,
		TaskID:         tc.TaskID,
		Title:          tc.Title,
		Precondition:   tc.Precondition,
		Steps:          tc.Steps,
		ExpectedResult: tc.ExpectedResult,
		CreatedBy:      tc.CreatedBy,
		UpdatedBy:      tc.UpdatedBy,
		CreatedAt:      tc.CreatedAt,
		UpdatedAt:      tc.UpdatedAt,
	}
}

func toTestRunResponse(tr *entities.CollabTestRun) models.TestRunResponse {
	return models.TestRunResponse{
		ID:         tr.ID,
		ProjectID:  tr.ProjectID,
		TestCaseID: tr.TestCaseID,
		Status:     tr.Status,
		ExecutedBy: tr.ExecutedBy,
		ExecutedAt: tr.ExecutedAt,
		Comment:    tr.Comment,
		CreatedAt:  tr.CreatedAt,
		UpdatedAt:  tr.UpdatedAt,
	}
}
