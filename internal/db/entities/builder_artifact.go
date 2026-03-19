package entities

import "time"

type BuilderArtifact struct {
	ID           string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
	SessionID    string    `gorm:"type:varchar(36);index;not null"`
	WorkspaceID  string    `gorm:"type:varchar(36);index"`
	RunID        string    `gorm:"type:varchar(36);index"`
	Kind         string    `gorm:"type:varchar(64);not null"`
	Path         string    `gorm:"type:varchar(512);not null"`
	MetadataJSON string    `gorm:"type:text"`
}
