package entities

import "time"

type BuilderMessageRole string

const (
	BuilderMessageRoleUser      BuilderMessageRole = "user"
	BuilderMessageRoleAssistant BuilderMessageRole = "assistant"
	BuilderMessageRoleSystem    BuilderMessageRole = "system"
)

type BuilderMessage struct {
	ID           string             `gorm:"type:varchar(36);primaryKey"`
	CreatedAt    time.Time          `gorm:"autoCreateTime"`
	UpdatedAt    time.Time          `gorm:"autoUpdateTime"`
	SessionID    string             `gorm:"type:varchar(36);index;not null"`
	RunID        *string            `gorm:"type:varchar(36);index"`
	Role         BuilderMessageRole `gorm:"type:varchar(16);not null"`
	Content      string             `gorm:"type:text;not null"`
	MetadataJSON string             `gorm:"type:text"`
	CreatedBy    string             `gorm:"type:varchar(36);index;not null"`
}
