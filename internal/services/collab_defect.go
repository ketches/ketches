package services

import (
	"fmt"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

var allowedDefectTransitions = map[string][]string{
	entities.DefectStatusNew:           {entities.DefectStatusProcessing, entities.DefectStatusRejected},
	entities.DefectStatusProcessing:    {entities.DefectStatusPendingVerify, entities.DefectStatusRejected},
	entities.DefectStatusPendingVerify: {entities.DefectStatusClosed},
	entities.DefectStatusClosed:        {entities.DefectStatusProcessing},
}

func ListDefects(projectID string, params *models.CollabFilterParams) (int64, []entities.CollabDefect, error) {
	params.Validate()

	q := db.DB.Where("project_id = ?", projectID)
	if params.Status != "" {
		q = q.Where("status = ?", params.Status)
	}
	if params.AssigneeID != "" {
		q = q.Where("assignee_id = ?", params.AssigneeID)
	}
	if params.SprintID != "" {
		q = q.Where("sprint_id = ?", params.SprintID)
	}
	if params.Search != "" {
		q = q.Where("title LIKE ?", "%"+params.Search+"%")
	}

	var total int64
	if err := q.Model(&entities.CollabDefect{}).Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var items []entities.CollabDefect
	if err := q.Order("created_at DESC").Offset(params.GetOffset()).Limit(params.PageSize).Find(&items).Error; err != nil {
		return 0, nil, err
	}

	return total, items, nil
}

func GetDefect(projectID, defectID string) (*entities.CollabDefect, error) {
	var defect entities.CollabDefect
	if err := db.DB.Where("id = ? AND project_id = ?", defectID, projectID).First(&defect).Error; err != nil {
		return nil, err
	}
	return &defect, nil
}

func CreateDefect(projectID, userID string, req *models.CreateDefectRequest) (*entities.CollabDefect, error) {
	if err := validateDefectLinks(projectID, req.RequirementID, req.TaskID, req.TestCaseID, req.TestRunID); err != nil {
		return nil, err
	}

	defect := &entities.CollabDefect{
		Base:               entities.Base{ID: uuid.New()},
		ProjectID:          projectID,
		SprintID:           req.SprintID,
		RequirementID:      req.RequirementID,
		TaskID:             req.TaskID,
		TestCaseID:         req.TestCaseID,
		TestRunID:          req.TestRunID,
		Title:              req.Title,
		Description:        req.Description,
		Severity:           req.Severity,
		Status:             req.Status,
		AssigneeID:         req.AssigneeID,
		ReproductionSteps:  req.ReproductionSteps,
		RuntimeContextJSON: "",
		CreatedBy:          userID,
	}

	if err := db.DB.Create(defect).Error; err != nil {
		return nil, err
	}

	return defect, nil
}

func UpdateDefect(projectID, defectID, userID string, req *models.UpdateDefectRequest) (*entities.CollabDefect, error) {
	defect, err := GetDefect(projectID, defectID)
	if err != nil {
		return nil, err
	}

	if err := validateDefectLinks(projectID, req.RequirementID, req.TaskID, req.TestCaseID, req.TestRunID); err != nil {
		return nil, err
	}

	defect.RequirementID = req.RequirementID
	defect.TaskID = req.TaskID
	defect.TestCaseID = req.TestCaseID
	defect.TestRunID = req.TestRunID
	defect.Title = req.Title
	defect.Description = req.Description
	defect.Severity = req.Severity
	defect.Status = req.Status
	defect.AssigneeID = req.AssigneeID
	defect.SprintID = req.SprintID
	defect.ReproductionSteps = req.ReproductionSteps
	defect.FixNote = req.FixNote
	defect.RuntimeContextJSON = req.RuntimeContextJSON
	defect.UpdatedBy = userID

	if err := db.DB.Save(defect).Error; err != nil {
		return nil, err
	}

	return defect, nil
}

func DeleteDefect(projectID, defectID string) error {
	return db.DB.Where("id = ? AND project_id = ?", defectID, projectID).Delete(&entities.CollabDefect{}).Error
}

func TransitionDefect(projectID, defectID, userID string, req *models.DefectTransitionRequest) (*entities.CollabDefect, error) {
	defect, err := GetDefect(projectID, defectID)
	if err != nil {
		return nil, err
	}

	targets, ok := allowedDefectTransitions[defect.Status]
	if !ok {
		return nil, fmt.Errorf("no transitions allowed from status %q", defect.Status)
	}

	allowed := false
	for _, t := range targets {
		if t == req.Status {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("transition from %q to %q is not allowed", defect.Status, req.Status)
	}

	defect.Status = req.Status
	defect.UpdatedBy = userID
	if err := db.DB.Save(defect).Error; err != nil {
		return nil, err
	}

	return defect, nil
}

func validateDefectLinks(projectID, requirementID, taskID, testCaseID, testRunID string) error {
	if requirementID == "" && taskID == "" && testCaseID == "" && testRunID == "" {
		return fmt.Errorf("defect must include at least one upstream link")
	}

	if requirementID != "" {
		if _, err := GetRequirement(projectID, requirementID); err != nil {
			return fmt.Errorf("cross-project link is not allowed for requirement_id")
		}
	}
	if taskID != "" {
		if _, err := GetTask(projectID, taskID); err != nil {
			return fmt.Errorf("cross-project link is not allowed for task_id")
		}
	}
	if testCaseID != "" {
		if _, err := GetTestCase(projectID, testCaseID); err != nil {
			return fmt.Errorf("cross-project link is not allowed for test_case_id")
		}
	}
	if testRunID != "" {
		var tr entities.CollabTestRun
		if err := db.DB.Where("id = ? AND project_id = ?", testRunID, projectID).First(&tr).Error; err != nil {
			return fmt.Errorf("cross-project link is not allowed for test_run_id")
		}
	}

	return nil
}
