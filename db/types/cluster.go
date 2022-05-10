package types

type Cluster struct {
	Base
	Name       string `json:"name" gorm:"column:name;type:varchar(255);not null;unique"`
	Desc       string `json:"desc" gorm:"column:desc;type:longtext"`
	KubeConfig string `json:"kubeConfig" gorm:"column:kube_config;type:longtext"`
	TenantID   uint32 `json:"tenantID" gorm:"column:tenant_id;not null"`
}
