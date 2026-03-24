package services

import (
	"errors"
	"fmt"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

func CreateUserAIProvider(userID string, req *models.CreateAIProviderRequest) (*models.AIProviderResponse, error) {
	provider := &entities.UserAIProvider{
		ID:                     uuid.New(),
		UserID:                 userID,
		ProviderKey:            req.ProviderKey,
		DisplayName:            req.DisplayName,
		BaseURL:                req.BaseURL,
		APIKey:                 req.APIKey,
		DefaultModelProfileKey: req.DefaultModelProfileKey,
		Enabled:                req.Enabled,
	}
	if err := db.DB.Create(provider).Error; err != nil {
		return nil, err
	}
	return &models.AIProviderResponse{
		ID:                     provider.ID,
		ProviderKey:            provider.ProviderKey,
		DisplayName:            provider.DisplayName,
		BaseURL:                provider.BaseURL,
		DefaultModelProfileKey: provider.DefaultModelProfileKey,
		Enabled:                provider.Enabled,
	}, nil
}

func CreateProjectAIProvider(projectID string, req *models.CreateAIProviderRequest) (*models.AIProviderResponse, error) {
	provider := &entities.ProjectAIProvider{
		ID:                     uuid.New(),
		ProjectID:              projectID,
		ProviderKey:            req.ProviderKey,
		DisplayName:            req.DisplayName,
		BaseURL:                req.BaseURL,
		APIKey:                 req.APIKey,
		DefaultModelProfileKey: req.DefaultModelProfileKey,
		Enabled:                req.Enabled,
	}
	if err := db.DB.Create(provider).Error; err != nil {
		return nil, err
	}
	return &models.AIProviderResponse{
		ID:                     provider.ID,
		ProviderKey:            provider.ProviderKey,
		DisplayName:            provider.DisplayName,
		BaseURL:                provider.BaseURL,
		DefaultModelProfileKey: provider.DefaultModelProfileKey,
		Enabled:                provider.Enabled,
	}, nil
}

func UpdateUserAIProvider(userID, providerID string, req *models.CreateAIProviderRequest) (*models.AIProviderResponse, error) {
	var provider entities.UserAIProvider
	if err := db.DB.Where("id = ? AND user_id = ?", providerID, userID).First(&provider).Error; err != nil {
		return nil, err
	}
	provider.ProviderKey = req.ProviderKey
	provider.DisplayName = req.DisplayName
	provider.BaseURL = req.BaseURL
	provider.APIKey = req.APIKey
	provider.DefaultModelProfileKey = req.DefaultModelProfileKey
	provider.Enabled = req.Enabled
	if err := db.DB.Save(&provider).Error; err != nil {
		return nil, err
	}
	return &models.AIProviderResponse{
		ID:                     provider.ID,
		ProviderKey:            provider.ProviderKey,
		DisplayName:            provider.DisplayName,
		BaseURL:                provider.BaseURL,
		DefaultModelProfileKey: provider.DefaultModelProfileKey,
		Enabled:                provider.Enabled,
	}, nil
}

func DeleteUserAIProvider(userID, providerID string) error {
	result := db.DB.Where("id = ? AND user_id = ?", providerID, userID).Delete(&entities.UserAIProvider{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func UpdateProjectAIProvider(projectID, providerID string, req *models.CreateAIProviderRequest) (*models.AIProviderResponse, error) {
	var provider entities.ProjectAIProvider
	if err := db.DB.Where("id = ? AND project_id = ?", providerID, projectID).First(&provider).Error; err != nil {
		return nil, err
	}
	provider.ProviderKey = req.ProviderKey
	provider.DisplayName = req.DisplayName
	provider.BaseURL = req.BaseURL
	provider.APIKey = req.APIKey
	provider.DefaultModelProfileKey = req.DefaultModelProfileKey
	provider.Enabled = req.Enabled
	if err := db.DB.Save(&provider).Error; err != nil {
		return nil, err
	}
	return &models.AIProviderResponse{
		ID:                     provider.ID,
		ProviderKey:            provider.ProviderKey,
		DisplayName:            provider.DisplayName,
		BaseURL:                provider.BaseURL,
		DefaultModelProfileKey: provider.DefaultModelProfileKey,
		Enabled:                provider.Enabled,
	}, nil
}

func DeleteProjectAIProvider(projectID, providerID string) error {
	result := db.DB.Where("id = ? AND project_id = ?", providerID, projectID).Delete(&entities.ProjectAIProvider{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func ListUserAIProviders(userID string) ([]models.AIProviderResponse, error) {
	var providers []entities.UserAIProvider
	if err := db.DB.Where("user_id = ? AND enabled = ?", userID, true).Order("created_at ASC, id ASC").Find(&providers).Error; err != nil {
		return nil, err
	}

	responses := make([]models.AIProviderResponse, 0, len(providers))
	for i := range providers {
		responses = append(responses, models.AIProviderResponse{
			ID:                     providers[i].ID,
			ProviderKey:            providers[i].ProviderKey,
			DisplayName:            providers[i].DisplayName,
			BaseURL:                providers[i].BaseURL,
			DefaultModelProfileKey: providers[i].DefaultModelProfileKey,
			Enabled:                providers[i].Enabled,
		})
	}

	return responses, nil
}

func ListProjectAIProviders(projectID string) ([]models.AIProviderResponse, error) {
	var providers []entities.ProjectAIProvider
	if err := db.DB.Where("project_id = ? AND enabled = ?", projectID, true).Order("created_at ASC, id ASC").Find(&providers).Error; err != nil {
		return nil, err
	}

	responses := make([]models.AIProviderResponse, 0, len(providers))
	for i := range providers {
		responses = append(responses, models.AIProviderResponse{
			ID:                     providers[i].ID,
			ProviderKey:            providers[i].ProviderKey,
			DisplayName:            providers[i].DisplayName,
			BaseURL:                providers[i].BaseURL,
			DefaultModelProfileKey: providers[i].DefaultModelProfileKey,
			Enabled:                providers[i].Enabled,
		})
	}

	return responses, nil
}

func ListBuilderAvailableModelOptions(projectID, userID string) ([]models.BuilderAvailableModelOptionResponse, error) {
	if projectID == "" || userID == "" {
		return nil, errors.New("project id and user id are required")
	}
	registry, err := loadBuilderProviderRegistry(app.Config)
	if err != nil {
		return nil, err
	}

	projectProviders, err := ListProjectAIProviders(projectID)
	if err != nil {
		return nil, err
	}
	userProviders, err := ListUserAIProviders(userID)
	if err != nil {
		return nil, err
	}

	options := make([]models.BuilderAvailableModelOptionResponse, 0, len(projectProviders)+len(userProviders))
	for _, provider := range projectProviders {
		if !provider.Enabled {
			continue
		}
		resolved, err := registry.resolveBuilderProviderProfile(provider.ProviderKey, provider.DefaultModelProfileKey)
		if err != nil {
			continue
		}
		options = append(options, models.BuilderAvailableModelOptionResponse{
			Key:             fmt.Sprintf("project:%s:%s", provider.ProviderKey, provider.DefaultModelProfileKey),
			ModelLabel:      resolved.ModelProfile.Model,
			ProviderLabel:   provider.DisplayName,
			Scope:           "project",
			ProviderKey:     provider.ProviderKey,
			ModelProfileKey: provider.DefaultModelProfileKey,
		})
	}
	for _, provider := range userProviders {
		if !provider.Enabled {
			continue
		}
		resolved, err := registry.resolveBuilderProviderProfile(provider.ProviderKey, provider.DefaultModelProfileKey)
		if err != nil {
			continue
		}
		options = append(options, models.BuilderAvailableModelOptionResponse{
			Key:             fmt.Sprintf("user:%s:%s", provider.ProviderKey, provider.DefaultModelProfileKey),
			ModelLabel:      resolved.ModelProfile.Model,
			ProviderLabel:   provider.DisplayName,
			Scope:           "user",
			ProviderKey:     provider.ProviderKey,
			ModelProfileKey: provider.DefaultModelProfileKey,
		})
	}

	return options, nil
}

func ResolveBuilderEffectiveDefault(projectID, userID string) (*models.BuilderEffectiveDefaultSelectionResponse, error) {
	options, err := ListBuilderAvailableModelOptions(projectID, userID)
	if err != nil {
		return nil, err
	}

	var projectDefault *entities.ProjectAIProvider
	if err := db.DB.Where("project_id = ? AND enabled = ? AND is_default = ?", projectID, true, true).Order("created_at ASC, id ASC").First(&projectDefault).Error; err == nil {
		for _, option := range options {
			if option.Scope == "project" && option.ProviderKey == projectDefault.ProviderKey && option.ModelProfileKey == projectDefault.DefaultModelProfileKey {
				resolved := option
				return &models.BuilderEffectiveDefaultSelectionResponse{
					EffectiveDefaultSource: "project",
					EffectiveDefaultOption: &resolved,
				}, nil
			}
		}
	}

	var userDefault *entities.UserAIProvider
	if err := db.DB.Where("user_id = ? AND enabled = ? AND is_default = ?", userID, true, true).Order("created_at ASC, id ASC").First(&userDefault).Error; err == nil {
		for _, option := range options {
			if option.Scope == "user" && option.ProviderKey == userDefault.ProviderKey && option.ModelProfileKey == userDefault.DefaultModelProfileKey {
				resolved := option
				return &models.BuilderEffectiveDefaultSelectionResponse{
					EffectiveDefaultSource: "user",
					EffectiveDefaultOption: &resolved,
				}, nil
			}
		}
	}

	return &models.BuilderEffectiveDefaultSelectionResponse{EffectiveDefaultSource: "none"}, nil
}

func GetBuilderModelSelection(projectID, userID string) (*models.BuilderModelSelectionResponse, error) {
	options, err := ListBuilderAvailableModelOptions(projectID, userID)
	if err != nil {
		return nil, err
	}
	selection, err := ResolveBuilderEffectiveDefault(projectID, userID)
	if err != nil {
		return nil, err
	}
	return &models.BuilderModelSelectionResponse{
		Options:                options,
		EffectiveDefaultSource: selection.EffectiveDefaultSource,
		EffectiveDefaultOption: selection.EffectiveDefaultOption,
	}, nil
}
