package entities

type Domain struct {
	Base
	Name        string  `gorm:"type:varchar(128);not null"`
	Domain      string  `gorm:"type:varchar(255);not null;index"`
	Description string  `gorm:"type:text"`
	Scope       string  `gorm:"type:varchar(16);not null;index"`
	ClusterID   string  `gorm:"type:varchar(36);not null;index"`
	EnvID       *string `gorm:"type:varchar(36);index"`
}
