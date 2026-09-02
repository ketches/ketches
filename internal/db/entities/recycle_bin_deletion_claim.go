package entities

import "time"

// RecycleBinDeletionClaim records that a soft-deleted resource is undergoing
// irreversible external cleanup and therefore cannot be restored.
type RecycleBinDeletionClaim struct {
	ID           string    `gorm:"type:varchar(36);primaryKey"`
	ResourceType string    `gorm:"type:varchar(32);not null;uniqueIndex:idx_recycle_bin_deletion_claim_resource,priority:1"`
	ResourceID   string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_recycle_bin_deletion_claim_resource,priority:2"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}
