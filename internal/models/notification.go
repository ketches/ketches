package models

import "time"

// NotificationResponse is the API response shape for a single notification.
type NotificationResponse struct {
	ID           string     `json:"id"`
	SenderID     string     `json:"sender_id"`
	SenderName   string     `json:"sender_name"`
	Category     string     `json:"category"`
	EventType    string     `json:"event_type"`
	Title        string     `json:"title"`
	Message      string     `json:"message"`
	Status       string     `json:"status"`
	ReadAt       *time.Time `json:"read_at"`
	ResourceType string     `json:"resource_type"`
	ResourceID   string     `json:"resource_id"`
	ProjectID    string     `json:"project_id"`
	ProjectName  string     `json:"project_name"`
	ActionData   string     `json:"action_data"`
	CreatedAt    time.Time  `json:"created_at"`
}

type ListNotificationResponse struct {
	Items      []NotificationResponse `json:"items"`
	Pagination PaginationResponse     `json:"pagination"`
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

type NotificationActionRequest struct {
	Action string `json:"action" binding:"required,oneof=accept refuse read dismiss"`
}

// NotificationJoinRow is a flat DTO for listing notifications with joined user/project names.
type NotificationJoinRow struct {
	ID           string     `gorm:"column:id"`
	SenderID     string     `gorm:"column:sender_id"`
	SenderName   string     `gorm:"column:sender_name"`
	Category     string     `gorm:"column:category"`
	EventType    string     `gorm:"column:event_type"`
	Title        string     `gorm:"column:title"`
	Message      string     `gorm:"column:message"`
	Status       string     `gorm:"column:status"`
	ReadAt       *time.Time `gorm:"column:read_at"`
	ResourceType string     `gorm:"column:resource_type"`
	ResourceID   string     `gorm:"column:resource_id"`
	ProjectID    string     `gorm:"column:project_id"`
	ProjectName  string     `gorm:"column:project_name"`
	ActionData   string     `gorm:"column:action_data"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
}
