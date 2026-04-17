package entities

import "time"

type ClusterGatewayProvider struct {
	ID                 string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt          time.Time `gorm:"autoCreateTime"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime"`
	ClusterID          string    `gorm:"type:varchar(36);not null;index"`
	SourceType         string    `gorm:"type:varchar(32);not null"`
	DisplayName        string    `gorm:"type:varchar(255);not null"`
	GatewayClassName   string    `gorm:"type:varchar(255);not null;uniqueIndex:uidx_cluster_gateway_class"`
	ControllerName     string    `gorm:"type:varchar(255);not null"`
	ExtensionID        *string   `gorm:"type:varchar(36)"`
	ClusterExtensionID *string   `gorm:"type:varchar(36)"`
	IsDefault          bool      `gorm:"type:bool;default:false;index"`
}
