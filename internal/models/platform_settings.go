package models

type UpdatePlatformBrandingRequest struct {
	Name string `json:"name" binding:"required"`
}

type PlatformBrandingResponse struct {
	Name string `json:"name"`
}

type PublicSignUpSettingsResponse struct {
	Enabled                   bool `json:"enabled"`
	EmailVerificationRequired bool `json:"email_verification_required"`
}

type UpdatePublicSignUpSettingsRequest struct {
	Enabled                   bool `json:"enabled"`
	EmailVerificationRequired bool `json:"email_verification_required"`
}
