package models

import "time"

type CreateProjectRequest struct {
	Slug        string `json:"slug" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type ProjectResponse struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerName   string    `json:"owner_name"`
	CreatedAt   time.Time `json:"created_at"`
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
