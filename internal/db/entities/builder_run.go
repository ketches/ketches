package entities

import "time"

type BuilderRunStatus string

const (
	BuilderRunStatusQueued    BuilderRunStatus = "queued"
	BuilderRunStatusExecuting BuilderRunStatus = "executing"
	BuilderRunStatusSucceeded BuilderRunStatus = "succeeded"
	BuilderRunStatusFailed    BuilderRunStatus = "failed"
	BuilderRunStatusCancelled BuilderRunStatus = "cancelled"
	BuilderRunStatusTimedOut  BuilderRunStatus = "timed_out"
)

type BuilderRunPhase string

const (
	BuilderRunPhaseQueued             BuilderRunPhase = "queued"
	BuilderRunPhaseClaiming           BuilderRunPhase = "claiming"
	BuilderRunPhasePreparingExecutor  BuilderRunPhase = "preparing_executor"
	BuilderRunPhaseGenerating         BuilderRunPhase = "generating"
	BuilderRunPhaseMaterializingFiles BuilderRunPhase = "materializing_files"
	BuilderRunPhaseBuilding           BuilderRunPhase = "building"
	BuilderRunPhaseTesting            BuilderRunPhase = "testing"
	BuilderRunPhasePublishing         BuilderRunPhase = "publishing"
	BuilderRunPhaseFinalizing         BuilderRunPhase = "finalizing"
)

type BuilderRun struct {
	ID                       string           `gorm:"type:varchar(36);primaryKey"`
	CreatedAt                time.Time        `gorm:"autoCreateTime;index:idx_builder_runs_status_phase_created,priority:3;index:idx_builder_runs_status_timeout_heartbeat_created,priority:4"`
	UpdatedAt                time.Time        `gorm:"autoUpdateTime"`
	SessionID                string           `gorm:"type:varchar(36);index;index:idx_builder_runs_session_status,priority:1;not null"`
	TriggerMessageID         string           `gorm:"type:varchar(36);index;not null"`
	WorkspaceID              *string          `gorm:"type:varchar(36);index"`
	Status                   BuilderRunStatus `gorm:"type:varchar(32);not null;default:'queued';index;index:idx_builder_runs_session_status,priority:2;index:idx_builder_runs_status_phase_created,priority:1;index:idx_builder_runs_status_timeout_heartbeat_created,priority:1"`
	Phase                    *BuilderRunPhase `gorm:"type:varchar(64);default:'queued';index;index:idx_builder_runs_status_phase_created,priority:2"`
	AttemptCount             int              `gorm:"not null;default:0"`
	MaxAttempts              int              `gorm:"not null;default:3"`
	ClaimToken               *string          `gorm:"type:varchar(64);index"`
	ClaimedAt                *time.Time       `gorm:"index"`
	HeartbeatAt              *time.Time       `gorm:"index;index:idx_builder_runs_status_timeout_heartbeat_created,priority:3"`
	TimeoutAt                *time.Time       `gorm:"index;index:idx_builder_runs_status_timeout_heartbeat_created,priority:2"`
	CancelRequestedAt        *time.Time       `gorm:"index"`
	ProviderScope            *string          `gorm:"type:varchar(32);index"`
	ProviderKey              *string          `gorm:"type:varchar(128);index"`
	ModelProfileKey          *string          `gorm:"type:varchar(128);index"`
	PlannedProjectKind       *string          `gorm:"type:varchar(128);index"`
	PlannedProjectSummary    string           `gorm:"type:text"`
	PlannedExecutorPolicyKey *string          `gorm:"type:varchar(128);index"`
	PlannedImageProfileKey   *string          `gorm:"type:varchar(128);index"`
	ExecutorPolicyKey        *string          `gorm:"type:varchar(128);index"`
	ExecutionImageProfileKey *string          `gorm:"type:varchar(128);index"`
	ExecutionImageRef        *string          `gorm:"type:varchar(512)"`
	ExecutorHandleID         *string          `gorm:"type:varchar(36);index"`
	RequestedBy              string           `gorm:"type:varchar(36);index;not null"`
	InstructionSummary       string           `gorm:"type:text"`
	ExecutionLog             string           `gorm:"type:text"`
	StartedAt                *time.Time
	CompletedAt              *time.Time
	ErrorCode                *string `gorm:"type:varchar(128)"`
	ErrorClass               *string `gorm:"type:varchar(64);index"`
	ErrorMessage             string  `gorm:"type:text"`
}
