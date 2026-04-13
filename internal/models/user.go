package models

import (
	"time"
)

type SignUpRequest struct {
	Username         string `json:"username" binding:"required"`
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required,min=8"`
	VerificationCode string `json:"verification_code" binding:"omitempty,len=6"`
	Fullname         string `json:"fullname"`
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Fullname string `json:"fullname"`
	Phone    string `json:"phone"`
	Role     string `json:"role" binding:"required,oneof=user admin"`
}

type SignInRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type SignInResponse struct {
	User                  UserResponse `json:"user"`
	MustChangePassword    bool         `json:"must_change_password"`
	DefaultPasswordNotice string       `json:"default_password_notice"`
}

type UserResponse struct {
	ID        string     `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Fullname  string     `json:"fullname"`
	Bio       string     `json:"bio"`
	Role      string     `json:"role"`
	IsLocked  bool       `json:"is_locked"`
	LockedAt  *time.Time `json:"locked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type UpdateCurrentUserProfileRequest struct {
	Fullname string `json:"fullname" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Bio      string `json:"bio"`
}

type ChangeCurrentUserPasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

type ChangeUserPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8"`
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

type UpdateUserLockRequest struct {
	Locked bool   `json:"locked"`
	Reason string `json:"reason"`
}

type SignUpVerificationCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type SignUpVerificationCodeResponse struct {
	ExpiresInSeconds   int `json:"expires_in_seconds"`
	ResendAfterSeconds int `json:"resend_after_seconds"`
}

// ListUsersResponse represents the paginated list of users
type ListUsersResponse struct {
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Users    []UserResponse `json:"users"`
}
