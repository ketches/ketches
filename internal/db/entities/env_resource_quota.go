package entities

type EnvResourceQuota struct {
	Base
	EnvID         string `gorm:"type:varchar(36);not null;uniqueIndex"`
	CPURequest    string `gorm:"type:varchar(32);not null;default:'2'"`
	CPULimit      string `gorm:"type:varchar(32);not null;default:'4'"`
	MemoryRequest string `gorm:"type:varchar(32);not null;default:'4Gi'"`
	MemoryLimit   string `gorm:"type:varchar(32);not null;default:'8Gi'"`
	Pods          string `gorm:"type:varchar(32);not null;default:'50'"`
}

func (EnvResourceQuota) TableName() string {
	return "env_resource_quota"
}

const DefaultCPURequest = "2"
const DefaultCPULimit = "4"
const DefaultMemoryRequest = "4Gi"
const DefaultMemoryLimit = "8Gi"
const DefaultPods = "50"
