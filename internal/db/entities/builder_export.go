package entities

import "time"

type BuilderExportStatus string

const (
	BuilderExportStatusReady  BuilderExportStatus = "ready"
	BuilderExportStatusFailed BuilderExportStatus = "failed"
)

type BuilderExport struct {
	ID           string              `gorm:"type:varchar(36);primaryKey"`
	CreatedAt    time.Time           `gorm:"autoCreateTime"`
	UpdatedAt    time.Time           `gorm:"autoUpdateTime"`
	SessionID    string              `gorm:"type:varchar(36);index;not null"`
	RunID        *string             `gorm:"type:varchar(36);index"`
	WorkspaceID  *string             `gorm:"type:varchar(36);index"`
	SnapshotID   *string             `gorm:"type:varchar(36);index"`
	Kind         string              `gorm:"type:varchar(64);index;not null"`
	Status       BuilderExportStatus `gorm:"type:varchar(32);index;not null"`
	FileName     string              `gorm:"type:varchar(255);not null"`
	StoragePath  string              `gorm:"type:varchar(1024);not null"`
	SourceRoot   string              `gorm:"type:varchar(512)"`
	FileCount    int                 `gorm:"not null;default:0"`
	SizeBytes    int64               `gorm:"not null;default:0"`
	MetadataJSON string              `gorm:"type:text"`
	ErrorMessage string              `gorm:"type:text"`
	CreatedBy    string              `gorm:"type:varchar(36);index;not null"`
}
