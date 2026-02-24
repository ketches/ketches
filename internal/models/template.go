package models

import "time"

// CreateTemplateRequest is the payload for creating a new template.
type CreateTemplateRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Status      string `json:"status"`
	Enabled     *bool  `json:"enabled"`
}

// UpdateTemplateRequest is the payload for updating an existing template.
type UpdateTemplateRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Status      string `json:"status"`
	Enabled     *bool  `json:"enabled"`
}

// TemplateResponse is the API response for a template.
type TemplateResponse struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Content     string    `json:"content"`
	Status      string    `json:"status"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
