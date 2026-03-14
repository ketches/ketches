package entities

import "time"

type Notification struct {
	Base

	RecipientID string `gorm:"type:varchar(36);not null;index:idx_notif_recipient_read"`
	SenderID    string `gorm:"type:varchar(36)"` // nullable for system notifications

	// Classification
	Category  string `gorm:"type:varchar(32);not null;index"`                         // "invitation", "assignment", "info"
	EventType string `gorm:"type:varchar(64);not null"`                               // "project_invitation", "task_assigned", etc.

	// Content
	Title   string `gorm:"type:varchar(200);not null"`
	Message string `gorm:"type:text"`

	// State
	Status string     `gorm:"type:varchar(16);not null;default:'pending';index:idx_notif_recipient_read"` // "pending", "accepted", "refused", "read", "dismissed"
	ReadAt *time.Time

	// Context
	ResourceType string `gorm:"type:varchar(32)"`        // "project", "task", "defect", "app"
	ResourceID   string `gorm:"type:varchar(36);index"`
	ProjectID    string `gorm:"type:varchar(36);index"`

	// Action-specific payload (JSON)
	ActionData string `gorm:"type:text"` // e.g. {"role": "developer"}
}

func (Notification) TableName() string {
	return "notifications"
}
