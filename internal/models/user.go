package models

import (
	"time"
)

type SignUpRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Fullname string `json:"fullname"`
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Fullname string `json:"fullname"`
	Phone    string `json:"phone"`
	Role     string `json:"role" binding:"required,oneof=user admin"`
}

type SignInRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type SignInResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Fullname  string    `json:"fullname"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// BatchImportRequest represents a batch of users to import
type BatchImportRequest struct {
	Users []CreateUserRequest `json:"users" binding:"required,dive"`
}

// BatchImportResponse represents the result of a batch import operation
type BatchImportResponse struct {
	Succeeded int            `json:"succeeded"`
	Failed    int            `json:"failed"`
	Errors    []ImportError  `json:"errors"`
	Users     []UserResponse `json:"users"`
}

type ImportError struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

// ChangeUserRoleRequest represents the request body for changing a user's role
type ChangeUserRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=user admin"`
}

// ListUsersResponse represents the paginated list of users
type ListUsersResponse struct {
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Users    []UserResponse `json:"users"`
}
