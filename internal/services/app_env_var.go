package services

import (
	"errors"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

func ListAppEnvVars(appID string) ([]models.AppEnvVarResponse, error) {
	return ListAppEnvVarsForProjectRole(appID, "")
}

func ListAppEnvVarsForProjectRole(appID string, projectRole app.ProjectRole) ([]models.AppEnvVarResponse, error) {
	return listAppEnvVars(appID, canRevealAppConfigurationValues(projectRole))
}

func listAppEnvVars(appID string, revealSecrets bool) ([]models.AppEnvVarResponse, error) {
	var envVars []entities.AppEnvVar
	err := db.DB.Where("app_id = ?", appID).Find(&envVars).Error
	if err != nil {
		return nil, err
	}
	result := make([]models.AppEnvVarResponse, 0, len(envVars))
	for _, ev := range envVars {
		response, err := toAppEnvVarResponse(&ev, revealSecrets)
		if err != nil {
			return nil, err
		}
		result = append(result, response)
	}
	return result, nil
}

func CreateAppEnvVar(appID string, key, value string, secretFlag ...bool) (*models.AppEnvVarResponse, error) {
	var existing entities.AppEnvVar
	err := db.DB.Where(&entities.AppEnvVar{AppID: appID, Key: key}).First(&existing).Error
	if err == nil {
		return nil, errors.New("environment variable with this key already exists for this app")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	isSecret := len(secretFlag) > 0 && secretFlag[0]
	storedValue := value
	if isSecret {
		storedValue, err = secrets.EncryptString(value)
		if err != nil {
			return nil, err
		}
	}
	ev := &entities.AppEnvVar{
		ID:       uuid.New(),
		AppID:    appID,
		Key:      key,
		Value:    storedValue,
		IsSecret: isSecret,
	}
	err = db.DB.Create(ev).Error
	if err != nil {
		return nil, err
	}
	res, err := toAppEnvVarResponse(ev, true)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func UpdateAppEnvVar(id string, value string, secretFlag ...bool) (*models.AppEnvVarResponse, error) {
	var envVar entities.AppEnvVar
	err := db.DB.First(&envVar, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("environment variable not found")
		}
		return nil, err
	}

	isSecret := envVar.IsSecret
	if len(secretFlag) > 0 {
		isSecret = secretFlag[0]
	}
	storedValue := value
	if isSecret {
		storedValue, err = secrets.EncryptString(value)
		if err != nil {
			return nil, err
		}
	}
	envVar.Value = storedValue
	envVar.IsSecret = isSecret

	err = db.DB.Save(&envVar).Error
	if err != nil {
		return nil, err
	}
	res, err := toAppEnvVarResponse(&envVar, true)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func DeleteAppEnvVar(id string) error {
	return db.DB.Delete(&entities.AppEnvVar{}, "id = ?", id).Error
}

// toAppEnvVarResponse converts an AppEnvVar entity to a response model with snake_case JSON fields.
func toAppEnvVarResponse(ev *entities.AppEnvVar, revealSecret bool) (models.AppEnvVarResponse, error) {
	value := ev.Value
	if !revealSecret {
		value = ""
	} else if ev.IsSecret {
		plaintext, err := secrets.DecryptStringCompatible(ev.Value)
		if err != nil {
			return models.AppEnvVarResponse{}, err
		}
		value = plaintext
	}
	return models.AppEnvVarResponse{
		ID:        ev.ID,
		AppID:     ev.AppID,
		Key:       ev.Key,
		Value:     value,
		IsSecret:  ev.IsSecret,
		HasValue:  ev.Value != "",
		CreatedAt: ev.CreatedAt,
		UpdatedAt: ev.UpdatedAt,
	}, nil
}
