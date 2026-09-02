package entities

type Env struct {
	Base
	Slug             string `gorm:"type:varchar(64);not null;uniqueIndex:idx_project_env_slug"`
	Name             string `gorm:"type:varchar(128);not null"`
	Description      string `gorm:"type:text"`
	ProjectID        string `gorm:"type:varchar(36);not null;uniqueIndex:idx_project_env_slug;index"`
	ClusterID        string `gorm:"type:varchar(36);index;not null;uniqueIndex:idx_cluster_namespace,where:cluster_namespace <> ''"`
	ClusterNamespace string `gorm:"type:varchar(128);serializer:nullable_string;uniqueIndex:idx_cluster_namespace,where:cluster_namespace <> ''"`
	IsBuildEnv       bool   `gorm:"type:bool;default:false"`
}
