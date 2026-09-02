package entities

import "time"

type Project struct {
	Base
	Slug                 string `gorm:"type:varchar(64);uniqueIndex;not null"`
	Name                 string `gorm:"type:varchar(128);not null"`
	Description          string `gorm:"type:text"`
	CollaborationEnabled bool   `gorm:"type:boolean;not null;default:false"`
}

type ProjectMember struct {
	ID          string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
	ProjectID   string    `gorm:"type:varchar(36);index:idx_project_members_project_id;uniqueIndex:idx_project_members_project_user;not null"`
	UserID      string    `gorm:"type:varchar(36);index:idx_project_members_user_id;uniqueIndex:idx_project_members_project_user;not null"`
	ProjectRole string    `gorm:"type:varchar(16);not null"`
}
