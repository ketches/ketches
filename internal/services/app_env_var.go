package services

import (
	"errors"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

func ListAppEnvVars(appID string) ([]entities.AppEnvVar, error) {
	var envVars []entities.AppEnvVar
	err := db.DB.Where("app_id = ?", appID).Find(&envVars).Error
	return envVars, err
}

func CreateAppEnvVar(appID string, key, value string) (*entities.AppEnvVar, error) {
	var existing entities.AppEnvVar
	err := db.DB.Where("app_id = ? AND `key` = ?", appID, key).First(&existing).Error
	if err == nil {
		return nil, errors.New("environment variable with this key already exists for this app")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	ev := &entities.AppEnvVar{
		ID:    uuid.New(),
		AppID: appID,
		Key:   key,
		Value: value,
	}
	err = db.DB.Create(ev).Error
	return ev, err
}

func UpdateAppEnvVar(id string, value string) (*entities.AppEnvVar, error) {
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
	return &envVar, err
}

func DeleteAppEnvVar(id string) error {
	return db.DB.Delete(&entities.AppEnvVar{}, "id = ?", id).Error
}
