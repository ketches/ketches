package entities

import "time"

type BuilderRunStatus string

const (
	BuilderRunStatusQueued    BuilderRunStatus = "queued"
	BuilderRunStatusExecuting BuilderRunStatus = "executing"
	BuilderRunStatusSucceeded BuilderRunStatus = "succeeded"
	BuilderRunStatusFailed    BuilderRunStatus = "failed"
	BuilderRunStatusCancelled BuilderRunStatus = "cancelled"
)

type BuilderRun struct {
	ID                 string           `gorm:"type:varchar(36);primaryKey"`
	CreatedAt          time.Time        `gorm:"autoCreateTime"`
	UpdatedAt          time.Time        `gorm:"autoUpdateTime"`
	SessionID          string           `gorm:"type:varchar(36);index;not null"`
	TriggerMessageID   string           `gorm:"type:varchar(36);index;not null"`
	WorkspaceID        *string          `gorm:"type:varchar(36);index"`
	Status             BuilderRunStatus `gorm:"type:varchar(32);not null;default:'queued';index"`
	RequestedBy        string           `gorm:"type:varchar(36);index;not null"`
	InstructionSummary string           `gorm:"type:text"`
	ExecutionLog       string           `gorm:"type:text"`
	StartedAt          *time.Time
	CompletedAt        *time.Time
	ErrorMessage       string `gorm:"type:text"`
}
