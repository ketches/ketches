package entities

import "time"

type Cluster struct {
	Base
	Slug        string `gorm:"type:varchar(64);uniqueIndex;not null"`
	Name        string `gorm:"type:varchar(128);not null"`
	Description string `gorm:"type:text"`
	KubeConfig  string `gorm:"type:text;not null"`
	GatewayIP   string `gorm:"type:varchar(64)"`
	Enabled     bool   `gorm:"type:bool;default:true"`

	ConnectionStatus       string     `gorm:"type:varchar(32);default:'unknown'"`
	ConnectionStatusReason string     `gorm:"type:text"`
	LastCheckedAt          *time.Time `gorm:"type:timestamp"`
}
