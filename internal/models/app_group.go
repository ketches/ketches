package models

// CreateAppGroupRequest is the request body for creating an app group.
type CreateAppGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateAppGroupRequest is the request body for updating an app group.
type UpdateAppGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// AppGroupResponse is the response body for an app group.
type AppGroupResponse struct {
	ID          string `json:"id"`
	EnvID       string `json:"env_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// AppGroupWithApps includes the group metadata and its apps for a given env.
type AppGroupWithApps struct {
	AppGroupResponse
	Apps []SimpleApp `json:"apps"`
}
