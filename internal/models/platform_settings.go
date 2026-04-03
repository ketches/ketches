package models

type UpdatePlatformBrandingRequest struct {
	Name string `json:"name" binding:"required"`
}

type PlatformBrandingResponse struct {
	Name string `json:"name"`
}

type PublicSignUpSettingsResponse struct {
	Enabled bool `json:"enabled"`
}

type UpdatePublicSignUpSettingsRequest struct {
	Enabled bool `json:"enabled"`
}
