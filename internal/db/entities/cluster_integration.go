package entities

import "time"

type IntegrationType string

const (
	IntegrationTypePrometheus   IntegrationType = "prometheus"
	IntegrationTypeGrafana      IntegrationType = "grafana"
	IntegrationTypeLoki         IntegrationType = "loki"
	IntegrationTypeAlertManager IntegrationType = "alertmanager"
)

type ClusterIntegration struct {
	ID              string          `gorm:"type:varchar(36);primaryKey"`
	CreatedAt       time.Time       `gorm:"autoCreateTime"`
	UpdatedAt       time.Time       `gorm:"autoUpdateTime"`
	ClusterID       string          `gorm:"type:varchar(36);not null;index"`
	IntegrationType IntegrationType `gorm:"type:varchar(32);not null"`
	Name            string          `gorm:"type:varchar(128);not null"`
	Endpoint        string          `gorm:"type:varchar(512);not null"`
	Namespace       string          `gorm:"type:varchar(64)"`
	ServiceName     string          `gorm:"type:varchar(128)"`
	ServicePort     int             `gorm:"type:int"`
	Username        string          `gorm:"type:varchar(128)"`
	Password        string          `gorm:"type:varchar(256)"`
	Token           string          `gorm:"type:text"`
	CACert          string          `gorm:"type:text"`
	SkipTLSVerify   bool            `gorm:"type:bool;default:false"`
	Enabled         bool            `gorm:"type:bool;default:true"`

	Cluster *Cluster `gorm:"foreignKey:ClusterID"`
}

func (ClusterIntegration) TableName() string {
	return "cluster_integrations"
}
