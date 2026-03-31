package models

type AIProviderResponse struct {
	ID                     string `json:"id"`
	ProviderKey            string `json:"provider_key"`
	DisplayName            string `json:"display_name"`
	BaseURL                string `json:"base_url"`
	DefaultModelProfileKey string `json:"default_model_profile_key"`
	Enabled                bool   `json:"enabled"`
	IsDefault              bool   `json:"is_default"`
}

type CreateAIProviderRequest struct {
	ProviderKey            string `json:"provider_key" binding:"required"`
	DisplayName            string `json:"display_name" binding:"required"`
	BaseURL                string `json:"base_url" binding:"required"`
	APIKey                 string `json:"api_key" binding:"required"`
	DefaultModelProfileKey string `json:"default_model_profile_key" binding:"required"`
	Enabled                bool   `json:"enabled"`
	IsDefault              bool   `json:"is_default"`
}

type BuilderAvailableModelOptionResponse struct {
	Key             string `json:"key"`
	ModelLabel      string `json:"model_label"`
	ProviderLabel   string `json:"provider_label"`
	Scope           string `json:"scope"`
	ProviderKey     string `json:"provider_key"`
	ModelProfileKey string `json:"model_profile_key"`
}

type BuilderEffectiveDefaultSelectionResponse struct {
	EffectiveDefaultSource string                               `json:"effective_default_source"`
	EffectiveDefaultOption *BuilderAvailableModelOptionResponse `json:"effective_default_option,omitempty"`
}

type BuilderModelSelectionResponse struct {
	Options                []BuilderAvailableModelOptionResponse `json:"options"`
	EffectiveDefaultSource string                                `json:"effective_default_source"`
	EffectiveDefaultOption *BuilderAvailableModelOptionResponse  `json:"effective_default_option,omitempty"`
}
