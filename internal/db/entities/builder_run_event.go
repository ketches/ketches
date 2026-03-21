package entities

import "time"

type BuilderRunEventLevel string

const (
	BuilderRunEventLevelInfo  BuilderRunEventLevel = "info"
	BuilderRunEventLevelWarn  BuilderRunEventLevel = "warn"
	BuilderRunEventLevelError BuilderRunEventLevel = "error"
)

type BuilderRunEventKind string

const (
	BuilderRunEventKindStatus   BuilderRunEventKind = "status"
	BuilderRunEventKindLog      BuilderRunEventKind = "log"
	BuilderRunEventKindArtifact BuilderRunEventKind = "artifact"
	BuilderRunEventKindPreview  BuilderRunEventKind = "preview"
	BuilderRunEventKindSystem   BuilderRunEventKind = "system"
)

type BuilderRunEvent struct {
	ID          string               `gorm:"type:varchar(36);primaryKey"`
	CreatedAt   time.Time            `gorm:"autoCreateTime;index"`
	RunID       string               `gorm:"type:varchar(36);not null;index:idx_builder_run_events_run_sequence,unique,priority:1"`
	Sequence    int64                `gorm:"not null;index:idx_builder_run_events_run_sequence,unique,priority:2"`
	Level       BuilderRunEventLevel `gorm:"type:varchar(16);not null;default:'info'"`
	Kind        BuilderRunEventKind  `gorm:"type:varchar(32);not null;index"`
	Phase       *BuilderRunPhase     `gorm:"type:varchar(64);index"`
	Message     string               `gorm:"type:text"`
	PayloadJSON string               `gorm:"type:text"`
}
