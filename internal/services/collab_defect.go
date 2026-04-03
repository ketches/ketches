package services

import (
	"github.com/ketches/ketches/internal/app"
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

	// Auto-create a linked task when no task_id is provided.
	if req.TaskID == "" {
		autoTask, err := CreateTask(projectID, userID, &models.CreateTaskRequest{
			Title:       "[Defect] " + req.Title,
			Description: req.Description,
			Status:      models.TaskStatusTodo,
			Priority:    severityToPriority(req.Severity),
			AssigneeID:  req.AssigneeID,
			SprintID:    req.SprintID,
		})
		if err != nil {
			return nil, app.WrapErrorf(err, "failed to auto-create task for defect: %w", err)
		}
		req.TaskID = autoTask.ID
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
		return nil, app.NewErrorf("no transitions allowed from status %q", defect.Status)
	}

	allowed := false
	for _, t := range targets {
		if t == req.Status {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, app.NewErrorf("transition from %q to %q is not allowed", defect.Status, req.Status)
	}

	defect.Status = req.Status
	defect.UpdatedBy = userID
	if err := db.DB.Save(defect).Error; err != nil {
		return nil, err
	}

	// Sync linked task status.
	if defect.TaskID != "" {
		targetTaskStatus := defectStatusToTaskStatus(req.Status)
		if targetTaskStatus != "" {
			db.DB.Model(&entities.CollabTask{}).
				Where("id = ? AND project_id = ?", defect.TaskID, projectID).
				Updates(map[string]any{"status": targetTaskStatus, "updated_by": userID})
		}
	}

	return defect, nil
}

func validateDefectLinks(projectID, requirementID, taskID, testCaseID, testRunID string) error {
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
	if testCaseID != "" {
		if _, err := GetTestCase(projectID, testCaseID); err != nil {
			return app.NewErrorf("cross-project link is not allowed for test_case_id")
		}
	}
	if testRunID != "" {
		var tr entities.CollabTestRun
		if err := db.DB.Where("id = ? AND project_id = ?", testRunID, projectID).First(&tr).Error; err != nil {
			return app.NewErrorf("cross-project link is not allowed for test_run_id")
		}
	}

	return nil
}

// severityToPriority maps defect severity to task priority.
func severityToPriority(severity string) string {
	switch severity {
	case entities.DefectSeverityCritical:
		return models.CollabPriorityP0
	case entities.DefectSeverityHigh:
		return models.CollabPriorityP1
	case entities.DefectSeverityMedium:
		return models.CollabPriorityP2
	default:
		return models.CollabPriorityP3
	}
}

// defectStatusToTaskStatus maps a defect status to its corresponding task status.
func defectStatusToTaskStatus(defectStatus string) string {
	switch defectStatus {
	case entities.DefectStatusNew:
		return entities.TaskStatusTodo
	case entities.DefectStatusProcessing:
		return entities.TaskStatusInProgress
	case entities.DefectStatusPendingVerify:
		return entities.TaskStatusReview
	case entities.DefectStatusClosed:
		return entities.TaskStatusDone
	case entities.DefectStatusRejected:
		return entities.TaskStatusCancelled
	default:
		return ""
	}
}
