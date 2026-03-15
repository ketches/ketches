package models

type UpdatePlatformBrandingRequest struct {
	Name string `json:"name" binding:"required"`
}

type PlatformBrandingResponse struct {
	Name string `json:"name"`
}
