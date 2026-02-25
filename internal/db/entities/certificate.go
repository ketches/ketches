package entities

// Certificate represents a TLS certificate used for HTTPS gateways.
// Certificates can be scoped to a cluster or an environment.
type Certificate struct {
	Base
	Name        string   `gorm:"type:varchar(128);not null"`
	Description string   `gorm:"type:text"`
	Cert        string   `gorm:"type:text;not null"`
	Key         string   `gorm:"type:text;not null"`
	Scope       string   `gorm:"type:varchar(16);not null"`
	ClusterID   string   `gorm:"type:varchar(36);index"`
	EnvID       string   `gorm:"type:varchar(36);index"`
	Cluster     *Cluster `gorm:"foreignKey:ClusterID"`
	Env         *Env     `gorm:"foreignKey:EnvID"`
}

func (Certificate) TableName() string {
	return "certificates"
}
