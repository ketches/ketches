package models

type UpdateBasicInfoRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type SimpleResponse struct {
	ID          string            `json:"id"`
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Status      string            `json:"status,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
