package entities

import "time"

// ClusterExtensionStatus represents the lifecycle state of an installed extension.
type ClusterExtensionStatus string

const (
	ClusterExtensionStatusPending      ClusterExtensionStatus = "pending"
	ClusterExtensionStatusInstalling   ClusterExtensionStatus = "installing"
	ClusterExtensionStatusDeployed     ClusterExtensionStatus = "deployed"
	ClusterExtensionStatusFailed       ClusterExtensionStatus = "failed"
	ClusterExtensionStatusUpgrading    ClusterExtensionStatus = "upgrading"
	ClusterExtensionStatusUninstalling ClusterExtensionStatus = "uninstalling"
)

// ClusterExtension tracks an extension installed on a specific cluster.
// The composite unique index uidx_cluster_ns_ext ensures the same extension
// cannot be installed in the same namespace of the same cluster more than once.
type ClusterExtension struct {
	ID           string                 `gorm:"type:varchar(36);primaryKey"`
	CreatedAt    time.Time              `gorm:"autoCreateTime"`
	UpdatedAt    time.Time              `gorm:"autoUpdateTime"`
	ClusterID    string                 `gorm:"type:varchar(36);not null;index;uniqueIndex:uidx_cluster_ns_ext"`
	ExtensionID  string                 `gorm:"type:varchar(36);not null;index;uniqueIndex:uidx_cluster_ns_ext"`
	Namespace    string                 `gorm:"type:varchar(128);not null;uniqueIndex:uidx_cluster_ns_ext"`
	ReleaseName  string                 `gorm:"type:varchar(256);not null"`
	Version      string                 `gorm:"type:varchar(64)"`
	Values       string                 `gorm:"type:longtext"`
	Status       ClusterExtensionStatus `gorm:"type:varchar(32);default:'pending'"`
	ErrorMessage string                 `gorm:"type:text"`
	InstalledBy  string                 `gorm:"type:varchar(36)"`
	Phase        string                 `gorm:"type:varchar(32)"` // Phase records the operation in progress when status is "failed": "installing", "upgrading", or "uninstalling"
}
