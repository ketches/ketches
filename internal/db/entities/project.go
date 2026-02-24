package entities

import "time"

type Project struct {
	Base
	Slug        string          `gorm:"type:varchar(64);uniqueIndex;not null"`
	Name        string          `gorm:"type:varchar(128);not null"`
	Description string          `gorm:"type:text"`
	Members     []ProjectMember `gorm:"foreignKey:ProjectID"`
	Envs        []Env           `gorm:"foreignKey:ProjectID"`
}

type ProjectMember struct {
	ID          string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
	ProjectID   string    `gorm:"type:varchar(36);index;not null"`
	UserID      string    `gorm:"type:varchar(36);index;not null"`
	ProjectRole string    `gorm:"type:varchar(16);not null"`
	User        User      `gorm:"foreignKey:UserID"`
}
