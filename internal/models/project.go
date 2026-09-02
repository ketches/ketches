package models

import "time"

type CreateProjectRequest struct {
	Slug                 string `json:"slug" binding:"required"`
	Name                 string `json:"name" binding:"required"`
	Description          string `json:"description"`
	CollaborationEnabled bool   `json:"collaboration_enabled"`
}

type ProjectResponse struct {
	ID                   string    `json:"id"`
	Slug                 string    `json:"slug"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	CollaborationEnabled bool      `json:"collaboration_enabled"`
	OwnerName            string    `json:"owner_name"`
	CreatedAt            time.Time `json:"created_at"`
}

type ListProjectResponse struct {
	Items      []ProjectResponse  `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

type ProjectMemberResponse struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	Fullname    string    `json:"fullname"`
	Email       string    `json:"email"`
	ProjectRole string    `json:"project_role"`
	JoinedAt    time.Time `json:"joined_at"`
}

type ListProjectMemberResponse struct {
	Items      []ProjectMemberResponse `json:"items"`
	Pagination PaginationResponse      `json:"pagination"`
}

// ProjectCapabilitiesResponse describes the authenticated user's effective
// access to a project. It is intentionally scoped to one user and project so
// callers do not need to fetch or scan a paginated member list.
type ProjectCapabilitiesResponse struct {
	ProjectRole  string              `json:"project_role"`
	Capabilities ProjectCapabilities `json:"capabilities"`
}

type ProjectCapabilities struct {
	Read   bool `json:"read"`
	Write  bool `json:"write"`
	Manage bool `json:"manage"`
}

// ProjectListRow is a flattened DTO for listing projects via JOIN queries.
// It avoids GORM Preload by scanning the owner name directly from a joined subquery.
type ProjectListRow struct {
	ID                   string    `gorm:"column:id"`
	Slug                 string    `gorm:"column:slug"`
	Name                 string    `gorm:"column:name"`
	Description          string    `gorm:"column:description"`
	CollaborationEnabled bool      `gorm:"column:collaboration_enabled"`
	CreatedAt            time.Time `gorm:"column:created_at"`
	OwnerName            string    `gorm:"column:owner_name"`
}
