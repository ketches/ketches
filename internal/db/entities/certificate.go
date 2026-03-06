package entities

import "time"

// Certificate represents a TLS certificate used for HTTPS gateways.
// Certificates can be scoped to a cluster or an environment.
type Certificate struct {
	ID          string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
	Name        string    `gorm:"type:varchar(128);not null"`
	Description string    `gorm:"type:text"`
	Cert        string    `gorm:"type:text;not null"`
	Key         string    `gorm:"type:text;not null"`
	Scope       string    `gorm:"type:varchar(16);not null"`
	ClusterID   string    `gorm:"type:varchar(36);index;not null"`
	EnvID       string    `gorm:"type:varchar(36);index"`

	Cluster     *Cluster  `gorm:"foreignKey:ClusterID"`
	Env         *Env      `gorm:"foreignKey:EnvID"`
}

func (Certificate) TableName() string {
	return "certificates"
}
