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
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// AppSimpleResponse is a simplified app response used inside group listings.
type AppSimpleResponse struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// AppGroupWithApps includes the group metadata and its apps for a given env.
type AppGroupWithApps struct {
	AppGroupResponse
	Apps []AppSimpleResponse `json:"apps"`
}

// AppFavoriteResponse is the response for an app favorite record.
type AppFavoriteResponse struct {
	ID    string `json:"id"`
	AppID string `json:"app_id"`
}
