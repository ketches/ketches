package entities

import "time"

type ProjectAIProvider struct {
	ID                     string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt              time.Time `gorm:"autoCreateTime"`
	UpdatedAt              time.Time `gorm:"autoUpdateTime"`
	ProjectID              string    `gorm:"type:varchar(36);index;not null"`
	ProviderKey            string    `gorm:"type:varchar(128);index;not null"`
	DisplayName            string    `gorm:"type:varchar(128);not null"`
	BaseURL                string    `gorm:"type:varchar(1024);not null"`
	APIKey                 string    `gorm:"type:text;not null"`
	DefaultModelProfileKey string    `gorm:"type:varchar(128);not null"`
	Enabled                bool      `gorm:"not null;index"`
	IsDefault              bool      `gorm:"not null;index"`
}

func (ProjectAIProvider) TableName() string {
	return "project_ai_providers"
}
