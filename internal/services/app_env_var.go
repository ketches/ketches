package services

import (
	"errors"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

func ListAppEnvVars(appID string) ([]models.AppEnvVarResponse, error) {
	var envVars []entities.AppEnvVar
	err := db.DB.Where("app_id = ?", appID).Find(&envVars).Error
	if err != nil {
		return nil, err
	}
	result := make([]models.AppEnvVarResponse, 0, len(envVars))
	for _, ev := range envVars {
		result = append(result, toAppEnvVarResponse(&ev))
	}
	return result, nil
}

func CreateAppEnvVar(appID string, key, value string) (*models.AppEnvVarResponse, error) {
	var existing entities.AppEnvVar
	err := db.DB.Where("app_id = ? AND `key` = ?", appID, key).First(&existing).Error
	if err == nil {
		return nil, errors.New("environment variable with this key already exists for this app")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	ev := &entities.AppEnvVar{
		ID:    uuid.New(),
		AppID: appID,
		Key:   key,
		Value: value,
	}
	err = db.DB.Create(ev).Error
	if err != nil {
		return nil, err
	}
	res := toAppEnvVarResponse(ev)
	return &res, nil
}

func UpdateAppEnvVar(id string, value string) (*models.AppEnvVarResponse, error) {
	var envVar entities.AppEnvVar
	err := db.DB.First(&envVar, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("environment variable not found")
		}
		return nil, err
	}

	// Only update value, key cannot be changed
	envVar.Value = value

	err = db.DB.Save(&envVar).Error
	if err != nil {
		return nil, err
	}
	res := toAppEnvVarResponse(&envVar)
	return &res, nil
}

func DeleteAppEnvVar(id string) error {
	return db.DB.Delete(&entities.AppEnvVar{}, "id = ?", id).Error
}

// toAppEnvVarResponse converts an AppEnvVar entity to a response model with snake_case JSON fields.
func toAppEnvVarResponse(ev *entities.AppEnvVar) models.AppEnvVarResponse {
	return models.AppEnvVarResponse{
		ID:        ev.ID,
		AppID:     ev.AppID,
		Key:       ev.Key,
		Value:     ev.Value,
		CreatedAt: ev.CreatedAt,
		UpdatedAt: ev.UpdatedAt,
	}
}
