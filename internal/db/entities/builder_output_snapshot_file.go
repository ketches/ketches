package entities

import "time"

type BuilderOutputSnapshotFile struct {
	ID             string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
	SnapshotID     string    `gorm:"type:varchar(36);not null;index:idx_builder_output_snapshot_files_snapshot_path,unique,priority:1"`
	RelativePath   string    `gorm:"type:varchar(255);not null;index:idx_builder_output_snapshot_files_snapshot_path,unique,priority:2"`
	StoragePath    string    `gorm:"type:varchar(1024);not null"`
	SizeBytes      int64     `gorm:"not null;default:0"`
	ContentType    string    `gorm:"type:varchar(255)"`
	IsDefaultEntry bool      `gorm:"not null;default:false"`
}
