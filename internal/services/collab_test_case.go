package services

import (
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

func ListTestCases(projectID string, params *models.CollabFilterParams) (int64, []entities.CollabTestCase, error) {
	params.Validate()

	q := db.DB.Where("project_id = ?", projectID)
	if params.Search != "" {
		q = q.Where("title LIKE ?", "%"+params.Search+"%")
	}
	if params.SprintID != "" {
		q = q.Where("sprint_id = ?", params.SprintID)
	}

	var total int64
	if err := q.Model(&entities.CollabTestCase{}).Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var items []entities.CollabTestCase
	if err := q.Order("created_at DESC").Offset(params.GetOffset()).Limit(params.PageSize).Find(&items).Error; err != nil {
		return 0, nil, err
	}

	return total, items, nil
}

func GetTestCase(projectID, testCaseID string) (*entities.CollabTestCase, error) {
	var tc entities.CollabTestCase
	if err := db.DB.Where("id = ? AND project_id = ?", testCaseID, projectID).First(&tc).Error; err != nil {
		return nil, err
	}
	return &tc, nil
}

func CreateTestCase(projectID, userID string, req *models.CreateTestCaseRequest) (*entities.CollabTestCase, error) {
	if err := validateTestCaseLinks(projectID, req.RequirementID, req.TaskID); err != nil {
		return nil, err
	}

	tc := &entities.CollabTestCase{
		Base:           entities.Base{ID: uuid.New()},
		ProjectID:      projectID,
		SprintID:       req.SprintID,
		RequirementID:  req.RequirementID,
		TaskID:         req.TaskID,
		Title:          req.Title,
		Precondition:   req.Precondition,
		Steps:          req.Steps,
		ExpectedResult: req.ExpectedResult,
		CreatedBy:      userID,
	}

	if err := db.DB.Create(tc).Error; err != nil {
		return nil, err
	}

	return tc, nil
}

func UpdateTestCase(projectID, testCaseID, userID string, req *models.UpdateTestCaseRequest) (*entities.CollabTestCase, error) {
	tc, err := GetTestCase(projectID, testCaseID)
	if err != nil {
		return nil, err
	}

	if err := validateTestCaseLinks(projectID, req.RequirementID, req.TaskID); err != nil {
		return nil, err
	}

	tc.RequirementID = req.RequirementID
	tc.TaskID = req.TaskID
	tc.Title = req.Title
	tc.SprintID = req.SprintID
	tc.Precondition = req.Precondition
	tc.Steps = req.Steps
	tc.ExpectedResult = req.ExpectedResult
	tc.UpdatedBy = userID

	if err := db.DB.Save(tc).Error; err != nil {
		return nil, err
	}

	return tc, nil
}

func DeleteTestCase(projectID, testCaseID string) error {
	return db.DB.Where("id = ? AND project_id = ?", testCaseID, projectID).Delete(&entities.CollabTestCase{}).Error
}

func CreateTestRun(projectID, testCaseID, userID string, req *models.CreateTestRunRequest) (*entities.CollabTestRun, error) {
	tc, err := GetTestCase(projectID, testCaseID)
	if err != nil {
		return nil, app.NewErrorf("cross-project link is not allowed for test_case_id")
	}

	tr := &entities.CollabTestRun{
		Base:       entities.Base{ID: uuid.New()},
		ProjectID:  projectID,
		TestCaseID: tc.ID,
		Status:     req.Status,
		ExecutedBy: userID,
		ExecutedAt: time.Now(),
		Comment:    req.Comment,
	}

	if err := db.DB.Create(tr).Error; err != nil {
		return nil, err
	}

	return tr, nil
}

func validateTestCaseLinks(projectID, requirementID, taskID string) error {
	if requirementID != "" {
		if _, err := GetRequirement(projectID, requirementID); err != nil {
			return app.NewErrorf("cross-project link is not allowed for requirement_id")
		}
	}
	if taskID != "" {
		if _, err := GetTask(projectID, taskID); err != nil {
			return app.NewErrorf("cross-project link is not allowed for task_id")
		}
	}
	return nil
}
