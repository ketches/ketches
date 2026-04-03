package services

import (
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

// allowedTaskTransitions defines valid status transitions for tasks.
var allowedTaskTransitions = map[string][]string{
	entities.TaskStatusTodo:       {entities.TaskStatusInProgress, entities.TaskStatusCancelled},
	entities.TaskStatusInProgress: {entities.TaskStatusReview, entities.TaskStatusCancelled},
	entities.TaskStatusReview:     {entities.TaskStatusDone, entities.TaskStatusCancelled},
	entities.TaskStatusDone:       {entities.TaskStatusInProgress},
}

// ListTasks returns paginated tasks for a project.
func ListTasks(projectID string, params *models.CollabFilterParams) (int64, []entities.CollabTask, error) {
	params.Validate()

	q := db.DB.Where("project_id = ?", projectID)
	if params.Status != "" {
		q = q.Where("status = ?", params.Status)
	}
	if params.SprintID != "" {
		q = q.Where("sprint_id = ?", params.SprintID)
	}
	if params.AssigneeID != "" {
		q = q.Where("assignee_id = ?", params.AssigneeID)
	}
	if params.Priority != "" {
		q = q.Where("priority = ?", params.Priority)
	}
	if params.Search != "" {
		q = q.Where("title LIKE ?", "%"+params.Search+"%")
	}

	var total int64
	if err := q.Model(&entities.CollabTask{}).Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var items []entities.CollabTask
	if err := q.Order("created_at DESC").Offset(params.GetOffset()).Limit(params.PageSize).Find(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}

// GetTask returns a single task by ID scoped to the project.
func GetTask(projectID, taskID string) (*entities.CollabTask, error) {
	var task entities.CollabTask
	if err := db.DB.Where("id = ? AND project_id = ?", taskID, projectID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// CreateTask creates a new task within a project.
func CreateTask(projectID, userID string, req *models.CreateTaskRequest) (*entities.CollabTask, error) {
	depth := 0
	if req.ParentTaskID != "" {
		parent, err := GetTask(projectID, req.ParentTaskID)
		if err != nil {
			return nil, app.WrapErrorf(err, "parent task not found: %w", err)
		}
		if parent.Depth >= entities.MaxCollabDepth {
			return nil, app.NewErrorf("maximum nesting depth (%d) exceeded", entities.MaxCollabDepth)
		}
		depth = parent.Depth + 1
	}

	var dueDate *time.Time
	if req.DueDate != "" {
		d, err := parseDate(req.DueDate)
		if err != nil {
			return nil, err
		}
		dueDate = d
	}

	task := &entities.CollabTask{
		Base:          entities.Base{ID: uuid.New()},
		ProjectID:     projectID,
		SprintID:      req.SprintID,
		RequirementID: req.RequirementID,
		Title:         req.Title,
		Description:   req.Description,
		Status:        req.Status,
		Priority:      req.Priority,
		AssigneeID:    req.AssigneeID,
		DueDate:       dueDate,
		EstimateHours: req.EstimateHours,
		ParentTaskID:  req.ParentTaskID,
		Depth:         depth,
		CreatedBy:     userID,
	}
	if err := db.DB.Create(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

// UpdateTask updates an existing task.
func UpdateTask(projectID, taskID, userID string, req *models.UpdateTaskRequest) (*entities.CollabTask, error) {
	task, err := GetTask(projectID, taskID)
	if err != nil {
		return nil, err
	}

	var dueDate *time.Time
	if req.DueDate != "" {
		d, err := parseDate(req.DueDate)
		if err != nil {
			return nil, err
		}
		dueDate = d
	}

	task.Title = req.Title
	task.Description = req.Description
	task.Status = req.Status
	task.Priority = req.Priority
	task.AssigneeID = req.AssigneeID
	task.SprintID = req.SprintID
	task.RequirementID = req.RequirementID
	task.DueDate = dueDate
	task.EstimateHours = req.EstimateHours
	task.UpdatedBy = userID

	if err := db.DB.Save(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

// DeleteTask soft-deletes a task by ID scoped to the project.
func DeleteTask(projectID, taskID string) error {
	return db.DB.Where("id = ? AND project_id = ?", taskID, projectID).Delete(&entities.CollabTask{}).Error
}

// TransitionTask validates and applies a status transition for a task.
func TransitionTask(projectID, taskID, userID string, req *models.TaskTransitionRequest) (*entities.CollabTask, error) {
	task, err := GetTask(projectID, taskID)
	if err != nil {
		return nil, err
	}

	targets, ok := allowedTaskTransitions[task.Status]
	if !ok {
		return nil, app.NewErrorf("no transitions allowed from status %q", task.Status)
	}

	allowed := false
	for _, t := range targets {
		if t == req.Status {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, app.NewErrorf("transition from %q to %q is not allowed", task.Status, req.Status)
	}

	task.Status = req.Status
	task.UpdatedBy = userID
	if err := db.DB.Save(task).Error; err != nil {
		return nil, err
	}

	// Sync all linked defect statuses.
	targetDefectStatus := taskStatusToDefectStatus(req.Status)
	if targetDefectStatus != "" {
		db.DB.Model(&entities.CollabDefect{}).
			Where("task_id = ? AND project_id = ?", taskID, projectID).
			Updates(map[string]any{"status": targetDefectStatus, "updated_by": userID})
	}

	return task, nil
}

// taskStatusToDefectStatus maps a task status to its corresponding defect status.
func taskStatusToDefectStatus(taskStatus string) string {
	switch taskStatus {
	case entities.TaskStatusTodo:
		return entities.DefectStatusNew
	case entities.TaskStatusInProgress:
		return entities.DefectStatusProcessing
	case entities.TaskStatusReview:
		return entities.DefectStatusPendingVerify
	case entities.TaskStatusDone:
		return entities.DefectStatusPendingVerify
	case entities.TaskStatusCancelled:
		return entities.DefectStatusRejected
	default:
		return ""
	}
}
