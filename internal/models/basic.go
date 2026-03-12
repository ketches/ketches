package models

type UpdateBasicInfoRequest struct {
	Name                 string `json:"name" binding:"required"`
	Description          string `json:"description"`
	CollaborationEnabled bool   `json:"collaboration_enabled"`
}
