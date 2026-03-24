package entities

import "time"

type BuilderOutputSnapshotStatus string

const (
	BuilderOutputSnapshotStatusDeliveryOnly BuilderOutputSnapshotStatus = "delivery_only"
	BuilderOutputSnapshotStatusPreviewable  BuilderOutputSnapshotStatus = "previewable"
)

type BuilderOutputSnapshot struct {
	ID               string                      `gorm:"type:varchar(36);primaryKey"`
	CreatedAt        time.Time                   `gorm:"autoCreateTime"`
	UpdatedAt        time.Time                   `gorm:"autoUpdateTime"`
	SessionID        string                      `gorm:"type:varchar(36);index;not null"`
	RunID            string                      `gorm:"type:varchar(36);uniqueIndex;not null"`
	WorkspaceID      string                      `gorm:"type:varchar(36);index;not null"`
	Status           BuilderOutputSnapshotStatus `gorm:"type:varchar(32);not null;index"`
	OutputRoot       string                      `gorm:"type:varchar(255);not null"`
	DefaultEntryPath string                      `gorm:"type:varchar(512)"`
	StoragePath      string                      `gorm:"type:varchar(1024);not null"`
	FileCount        int                         `gorm:"not null;default:0"`
	TotalSizeBytes   int64                       `gorm:"not null;default:0"`
	PublishedAt      time.Time                   `gorm:"index;not null"`
}
