package entities

import "time"

// Extension represents a platform-level OCI-based Helm chart extension.
// It is admin-managed and can be installed on any cluster.
type Extension struct {
	ID           string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
	Slug         string    `gorm:"type:varchar(128);uniqueIndex;not null"`
	Name         string    `gorm:"type:varchar(256)"`
	Description  string    `gorm:"type:text"`
	Capabilities string    `gorm:"type:text"`
	Metadata     JSONBlob
	OCIUrl       string  `gorm:"type:varchar(512);not null"`
	IconURL      string  `gorm:"type:varchar(512)"`
	Builtin      bool    `gorm:"type:bool;default:false"`
	CreatedBy    *string `gorm:"type:varchar(36)"`
}
