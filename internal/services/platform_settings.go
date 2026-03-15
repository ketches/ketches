package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

const (
	platformBrandingSettingKey  = "platform_branding"
	platformBrandingDefaultName = "Ketches Admin"
)

type storedPlatformBranding struct {
	Name string `json:"name"`
}

func GetPlatformBranding() (*models.PlatformBrandingResponse, error) {
	branding, err := loadStoredPlatformBranding()
	if err != nil {
		return nil, err
	}
	return platformBrandingResponseFromStored(branding), nil
}

func UpdatePlatformBranding(req *models.UpdatePlatformBrandingRequest, _ *app.Claims) (*models.PlatformBrandingResponse, error) {
	if req == nil {
		return nil, errors.New("branding request is required")
	}

	branding, err := loadStoredPlatformBranding()
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("platform name is required")
	}
	branding.Name = name

	if err := saveStoredPlatformBranding(branding); err != nil {
		return nil, err
	}

	return platformBrandingResponseFromStored(branding), nil
}

func loadStoredPlatformBranding() (*storedPlatformBranding, error) {
	setting, err := getSystemSetting(platformBrandingSettingKey)
	if err != nil {
		return nil, err
	}
	if setting == nil || strings.TrimSpace(setting.Value) == "" {
		return &storedPlatformBranding{Name: platformBrandingDefaultName}, nil
	}

	var branding storedPlatformBranding
	if err := json.Unmarshal([]byte(setting.Value), &branding); err != nil {
		return nil, fmt.Errorf("failed to decode platform branding: %w", err)
	}
	if strings.TrimSpace(branding.Name) == "" {
		branding.Name = platformBrandingDefaultName
	}
	return &branding, nil
}

func saveStoredPlatformBranding(branding *storedPlatformBranding) error {
	payload, err := json.Marshal(branding)
	if err != nil {
		return fmt.Errorf("failed to encode platform branding: %w", err)
	}

	setting, err := getSystemSetting(platformBrandingSettingKey)
	if err != nil {
		return err
	}
	if setting == nil {
		return db.DB.Create(&entities.SystemSetting{
			Base:  entities.Base{ID: uuid.New()},
			Key:   platformBrandingSettingKey,
			Value: string(payload),
		}).Error
	}

	return db.DB.Model(&entities.SystemSetting{}).
		Where("id = ?", setting.ID).
		Update("value", string(payload)).Error
}

func platformBrandingResponseFromStored(branding *storedPlatformBranding) *models.PlatformBrandingResponse {
	if branding == nil {
		return &models.PlatformBrandingResponse{Name: platformBrandingDefaultName}
	}

	response := &models.PlatformBrandingResponse{
		Name: branding.Name,
	}
	return response
}
