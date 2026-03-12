package entities

import "time"

// TaskStatus represents the workflow status of a task.
type TaskStatus = string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusReview     TaskStatus = "review"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// CollabTask represents an executable work item within a project.
type CollabTask struct {
	Base

	ProjectID     string     `gorm:"type:varchar(36);not null;index:idx_collab_task_project_status;index:idx_collab_task_project_sprint;index:idx_collab_task_project_req"`
	SprintID      string     `gorm:"type:varchar(36);index:idx_collab_task_project_sprint"`
	RequirementID string     `gorm:"type:varchar(36);index:idx_collab_task_project_req"`
	Title         string     `gorm:"type:varchar(200);not null"`
	Description   string     `gorm:"type:text"`
	Status        string     `gorm:"type:varchar(24);not null;index:idx_collab_task_project_status"`
	Priority      string     `gorm:"type:varchar(8);not null;index"`
	AssigneeID    string     `gorm:"type:varchar(36);index"`
	DueDate       *time.Time `gorm:"type:date"`
	EstimateHours *float64   `gorm:"type:decimal(8,2)"`
	ParentTaskID  string     `gorm:"type:varchar(36);index:idx_collab_task_parent"`
	Depth         int        `gorm:"type:int;not null;index:idx_collab_task_parent"`
	CreatedBy     string     `gorm:"type:varchar(36);not null;index"`
	UpdatedBy     string     `gorm:"type:varchar(36);index"`
}

func (CollabTask) TableName() string {
	return "collab_tasks"
}
