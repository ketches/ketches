package models

import (
	"github.com/ketches/ketches/internal/api"
)

type ProjectModel struct {
	ProjectID   string `json:"projectID"`
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
}

type ListProjectsRequest struct {
	api.QueryAndPagedFilter `form:",inline"`
}

type ListProjectResponse struct {
	Total   int64           `json:"total"`
	Records []*ProjectModel `json:"records"`
}

type GetProjectRefRequest struct {
	ProjectID string `uri:"projectID" binding:"required"`
}

type ProjectRef struct {
	ProjectID   string `json:"projectID" gorm:"column:id"`
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
}

type GetProjectRequest struct {
	ProjectID string `uri:"projectID" binding:"required"`
}

type CreateProjectRequest struct {
	Slug        string `json:"slug" binding:"required,slug"`
	DisplayName string `json:"displayName" binding:"required"`
	Description string `json:"description,omitempty"`
	Operator    string `json:"-"`
}

type UpdateProjectRequest struct {
	ProjectID   string `json:"-" uri:"projectID"`
	DisplayName string `json:"displayName" binding:"required"`
	Description string `json:"description,omitempty"`
}

type DeleteProjectRequest struct {
	ProjectID string `uri:"projectID" binding:"required"`
}

type ListProjectMembersRequest struct {
	ProjectID               string `uri:"projectID"`
	api.QueryAndPagedFilter `form:",inline"`
}

type ListProjectMembersResponse struct {
	Total   int64                 `json:"total"`
	Records []*ProjectMemberModel `json:"records"`
}

type ProjectMemberModel struct {
	ProjectID   string `json:"projectID"`
	UserID      string `json:"userID"`
	Username    string `json:"username"`
	Fullname    string `json:"fullname,omitempty"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	ProjectRole string `json:"projectRole"`
	CreatedAt   string `json:"createdAt"`
}

type ProjectMemberRole struct {
	UserID      string `json:"userID"`
	ProjectRole string `json:"projectRole" binding:"required,oneof=owner developer viewer"`
}

type AddProjectMembersRequest struct {
	ProjectID          string               `json:"-" uri:"projectID"`
	ProjectMemberRoles []*ProjectMemberRole `json:"projectMembers" binding:"required,unique"`
}

type UpdateProjectMemberRequest struct {
	ProjectID   string `json:"-" uri:"projectID"`
	UserID      string `json:"-" uri:"userID"`
	ProjectRole string `json:"projectRole" binding:"required"`
}

type RemoveProjectMembersRequest struct {
	ProjectID string   `json:"-" uri:"projectID"`
	UserIDs   []string `json:"userIDs" uri:"userIDs" binding:"required,unique"`
}

type SetUserDefaultProjectRequest struct {
	UserID    string `json:"-" uri:"userID" binding:"required"`
	ProjectID string `json:"-" uri:"projectID" binding:"required"`
}
