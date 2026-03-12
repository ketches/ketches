package entities

import "time"

// SprintStatus represents the lifecycle status of a sprint.
type SprintStatus = string

const (
	SprintStatusPlanned SprintStatus = "planned"
	SprintStatusActive  SprintStatus = "active"
	SprintStatusClosed  SprintStatus = "closed"
)

// CollabSprint represents a time-boxed iteration within a project.
type CollabSprint struct {
	Base

	ProjectID string     `gorm:"type:varchar(36);not null;index:idx_collab_sprints_project_status"`
	Name      string     `gorm:"type:varchar(64);not null"`
	Goal      string     `gorm:"type:text"`
	Status    string     `gorm:"type:varchar(16);not null;index:idx_collab_sprints_project_status"`
	StartDate *time.Time `gorm:"type:date;not null"`
	EndDate   *time.Time `gorm:"type:date;not null"`
	CreatedBy string     `gorm:"type:varchar(36);not null;index"`
	UpdatedBy string     `gorm:"type:varchar(36);index"`
}

func (CollabSprint) TableName() string {
	return "collab_sprints"
}
