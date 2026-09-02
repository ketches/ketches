package services

import (
	"context"
	"errors"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

func ListAppConfigFiles(appID string) ([]models.AppConfigFileResponse, error) {
	return ListAppConfigFilesForProjectRole(appID, "")
}

func ListAppConfigFilesForProjectRole(appID string, projectRole app.ProjectRole) ([]models.AppConfigFileResponse, error) {
	return listAppConfigFiles(appID, canRevealAppConfigurationValues(projectRole))
}

func listAppConfigFiles(appID string, revealSecrets bool) ([]models.AppConfigFileResponse, error) {
	var configFiles []entities.AppConfigFile
	err := db.DB.Where("app_id = ?", appID).Find(&configFiles).Error
	if err != nil {
		return nil, err
	}
	result := make([]models.AppConfigFileResponse, 0, len(configFiles))
	for _, cf := range configFiles {
		response, err := toAppConfigFileResponse(&cf, revealSecrets)
		if err != nil {
			return nil, err
		}
		result = append(result, response)
	}
	return result, nil
}

func canRevealAppConfigurationValues(projectRole app.ProjectRole) bool {
	switch projectRole {
	case app.ProjectRoleOwner, app.ProjectRoleDeveloper:
		return true
	default:
		return false
	}
}

func CreateAppConfigFile(ctx context.Context, appID string, req *models.CreateConfigFileRequest) (*models.AppConfigFileResponse, error) {
	fileMode := req.FileMode
	if fileMode == "" {
		fileMode = "0644"
	}

	content := req.Content
	if req.IsSecret {
		var err error
		content, err = secrets.EncryptString(req.Content)
		if err != nil {
			return nil, err
		}
	}
	entity := &entities.AppConfigFile{
		ID:        uuid.New(),
		AppID:     appID,
		Slug:      req.Slug,
		MountPath: req.MountPath,
		Content:   content,
		FileMode:  fileMode,
		IsSecret:  req.IsSecret,
	}

	// Check slug uniqueness
	var existing entities.AppConfigFile
	err := db.DB.Where("app_id = ? AND slug = ?", appID, req.Slug).First(&existing).Error
	if err == nil {
		return nil, errors.New("config file with this slug already exists for this app")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Check mount path conflicts
	if err := checkMountPathConflicts(appID, req.MountPath, ""); err != nil {
		return nil, err
	}

	// Create in database
	if err := db.DB.Create(entity).Error; err != nil {
		return nil, err
	}

	// Fetch full app context from DB
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	// Sync ConfigMap to Kubernetes
	if err := core.SyncConfigMapToK8s(ctx, appCtx); err != nil {
		return nil, err
	}

	res, err := toAppConfigFileResponse(entity, true)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func UpdateAppConfigFile(ctx context.Context, id string, req *models.UpdateConfigFileRequest) (*models.AppConfigFileResponse, error) {
	var configFile entities.AppConfigFile
	err := db.DB.First(&configFile, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("config file not found")
		}
		return nil, err
	}

	// Check slug uniqueness (excluding current file)
	if req.Slug != configFile.Slug {
		var existing entities.AppConfigFile
		err := db.DB.Where("app_id = ? AND slug = ? AND id != ?", configFile.AppID, req.Slug, id).First(&existing).Error
		if err == nil {
			return nil, errors.New("config file with this slug already exists for this app")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// Check mount path conflicts (excluding current file)
	if req.MountPath != configFile.MountPath {
		if err := checkMountPathConflicts(configFile.AppID, req.MountPath, id); err != nil {
			return nil, err
		}
	}

	// Update fields
	configFile.Slug = req.Slug
	configFile.MountPath = req.MountPath
	isSecret := configFile.IsSecret
	if req.IsSecret != nil {
		isSecret = *req.IsSecret
	}
	content := req.Content
	if isSecret {
		content, err = secrets.EncryptString(req.Content)
		if err != nil {
			return nil, err
		}
	}
	configFile.Content = content
	configFile.IsSecret = isSecret
	if req.FileMode != "" {
		configFile.FileMode = req.FileMode
	}

	// Save to database
	if err := db.DB.Save(&configFile).Error; err != nil {
		return nil, err
	}

	// Fetch full app context from DB
	appCtx, err := GetAppContext(ctx, configFile.AppID)
	if err != nil {
		return nil, err
	}

	// Sync ConfigMap to Kubernetes
	if err := core.SyncConfigMapToK8s(ctx, appCtx); err != nil {
		return nil, err
	}

	res, err := toAppConfigFileResponse(&configFile, true)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func DeleteAppConfigFile(ctx context.Context, id string) error {
	var configFile entities.AppConfigFile
	err := db.DB.First(&configFile, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("config file not found")
		}
		return err
	}

	// Delete from database
	if err := db.DB.Delete(&configFile).Error; err != nil {
		return err
	}
	appCtx, err := GetAppContext(ctx, configFile.AppID)
	if err != nil {
		return err
	}

	// Sync ConfigMap to Kubernetes (will remove deleted file)
	if err := core.SyncConfigMapToK8s(ctx, appCtx); err != nil {
		return err
	}

	return nil
}

// toAppConfigFileResponse converts an AppConfigFile entity to a response model with snake_case JSON fields.
func toAppConfigFileResponse(cf *entities.AppConfigFile, revealSecret bool) (models.AppConfigFileResponse, error) {
	content := cf.Content
	if !revealSecret {
		content = ""
	} else if cf.IsSecret {
		plaintext, err := secrets.DecryptStringCompatible(cf.Content)
		if err != nil {
			return models.AppConfigFileResponse{}, err
		}
		content = plaintext
	}
	return models.AppConfigFileResponse{
		ID:        cf.ID,
		AppID:     cf.AppID,
		Slug:      cf.Slug,
		MountPath: cf.MountPath,
		Content:   content,
		FileMode:  cf.FileMode,
		IsSecret:  cf.IsSecret,
		HasValue:  cf.Content != "",
		CreatedAt: cf.CreatedAt,
		UpdatedAt: cf.UpdatedAt,
	}, nil
}

// checkMountPathConflicts checks if the mount path conflicts with existing config files or volumes
func checkMountPathConflicts(appID, mountPath, excludeID string) error {
	// Check against other config files
	var configFiles []entities.AppConfigFile
	query := db.DB.Where("app_id = ?", appID)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Find(&configFiles).Error; err != nil {
		return err
	}

	for _, cf := range configFiles {
		if pathsConflict(mountPath, cf.MountPath) {
			return errors.New("mount path conflicts with existing config file: " + cf.MountPath)
		}
	}

	// Check against volumes
	var volumes []entities.AppVolume
	if err := db.DB.Where("app_id = ?", appID).Find(&volumes).Error; err != nil {
		return err
	}

	for _, vol := range volumes {
		if pathsConflict(mountPath, vol.MountPath) {
			return errors.New("mount path conflicts with existing volume: " + vol.MountPath)
		}
	}

	return nil
}

// pathsConflict checks if two paths conflict (one is a parent/child of the other)
func pathsConflict(path1, path2 string) bool {
	// Normalize paths
	p1 := strings.TrimSuffix(path1, "/")
	p2 := strings.TrimSuffix(path2, "/")

	if p1 == p2 {
		return true
	}

	// Check if one path is a prefix of another
	if strings.HasPrefix(p1, p2+"/") || strings.HasPrefix(p2, p1+"/") {
		return true
	}

	return false
}
