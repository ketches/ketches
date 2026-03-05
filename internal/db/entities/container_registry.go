package entities

import "time"

type RegistryProvider string

const (
	RegistryProviderDockerHub RegistryProvider = "dockerhub"
	RegistryProviderHarbor    RegistryProvider = "harbor"
	RegistryProviderGHCR      RegistryProvider = "ghcr"
	RegistryProviderACR       RegistryProvider = "acr"
	RegistryProviderECR       RegistryProvider = "ecr"
	RegistryProviderAliyun    RegistryProvider = "aliyun"
	RegistryProviderCustom    RegistryProvider = "custom"
)

type RegistryScope string

const (
	RegistryScopeCluster RegistryScope = "cluster"
	RegistryScopeProject RegistryScope = "project"
)

type ContainerRegistry struct {
	ID            string           `gorm:"type:varchar(36);primaryKey"`
	CreatedAt     time.Time        `gorm:"autoCreateTime"`
	UpdatedAt     time.Time        `gorm:"autoUpdateTime"`
	Name          string           `gorm:"type:varchar(128);not null"`
	Provider      RegistryProvider `gorm:"type:varchar(32);not null"`
	Endpoint      string           `gorm:"type:varchar(512);not null"`
	SkipTLSVerify bool             `gorm:"type:bool;default:false"`
	Namespace     string           `gorm:"type:varchar(256)"`
	Username      string           `gorm:"type:varchar(128)"`
	Password      string           `gorm:"type:varchar(512)"`
	Scope         RegistryScope    `gorm:"type:varchar(16);not null"`
	ClusterID     *string          `gorm:"type:varchar(36);index"` // NULL for project-scoped registries
	ProjectID     *string          `gorm:"type:varchar(36);index"` // NULL for cluster-scoped registries
	IsDefault     bool             `gorm:"type:bool;default:false"`
	Enabled       bool             `gorm:"type:bool;default:true"`
	Description   string           `gorm:"type:text"`

	Cluster *Cluster `gorm:"foreignKey:ClusterID"`
	Project *Project `gorm:"foreignKey:ProjectID"`
}

func (ContainerRegistry) TableName() string {
	return "container_registries"
}
