package entities

type Env struct {
	Base
	Slug             string  `gorm:"type:varchar(64);not null;uniqueIndex:idx_project_env_slug"`
	Name             string  `gorm:"type:varchar(128);not null"`
	Description      string  `gorm:"type:text"`
	ProjectID        string  `gorm:"type:varchar(36);not null;uniqueIndex:idx_project_env_slug;index"`
	ClusterID        string  `gorm:"type:varchar(36);index;not null"`
	ClusterNamespace string  `gorm:"type:varchar(128)"`
	IsBuildEnv       bool    `gorm:"type:bool;default:false"`
	Project          Project `gorm:"foreignKey:ProjectID"`
	Cluster          Cluster `gorm:"foreignKey:ClusterID"`
}
