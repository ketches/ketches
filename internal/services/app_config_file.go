package services

import (
	"context"
	"errors"
	"strings"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

func ListAppConfigFiles(appID string) ([]entities.AppConfigFile, error) {
	var configFiles []entities.AppConfigFile
	err := db.DB.Where("app_id = ?", appID).Find(&configFiles).Error
	return configFiles, err
}

func CreateAppConfigFile(appID string, req *models.CreateConfigFileRequest) (*entities.AppConfigFile, error) {
	fileMode := req.FileMode
	if fileMode == "" {
		fileMode = "0644"
	}

	entity := &entities.AppConfigFile{
		ID:        uuid.New(),
		AppID:     appID,
		Slug:      req.Slug,
		MountPath: req.MountPath,
		Content:   req.Content,
		FileMode:  fileMode,
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

	// Load app with environment for K8s sync
	var app entities.App
	if err := db.DB.Preload("Env").Preload("ConfigFiles").First(&app, "id = ?", appID).Error; err != nil {
		return nil, err
	}

	// Sync ConfigMap to Kubernetes
	if err := core.SyncConfigMapToK8s(context.Background(), &app); err != nil {
		return nil, err
	}

	return entity, nil
}

func UpdateAppConfigFile(id string, req *models.UpdateConfigFileRequest) (*entities.AppConfigFile, error) {
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
	configFile.Content = req.Content
	if req.FileMode != "" {
		configFile.FileMode = req.FileMode
	}

	// Save to database
	if err := db.DB.Save(&configFile).Error; err != nil {
		return nil, err
	}

	// Load app with environment for K8s sync
	var app entities.App
	if err := db.DB.Preload("Env").Preload("ConfigFiles").First(&app, "id = ?", configFile.AppID).Error; err != nil {
		return nil, err
	}

	// Sync ConfigMap to Kubernetes
	if err := core.SyncConfigMapToK8s(context.Background(), &app); err != nil {
		return nil, err
	}

	return &configFile, nil
}

func DeleteAppConfigFile(id string) error {
	var configFile entities.AppConfigFile
	err := db.DB.First(&configFile, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("config file not found")
		}
		return err
	}

	// Load app with environment for K8s sync
	var app entities.App
	if err := db.DB.Preload("Env").Preload("ConfigFiles").First(&app, "id = ?", configFile.AppID).Error; err != nil {
		return err
	}

	// Delete from database
	if err := db.DB.Delete(&configFile).Error; err != nil {
		return err
	}

	// Sync ConfigMap to Kubernetes (will remove deleted file)
	if err := core.SyncConfigMapToK8s(context.Background(), &app); err != nil {
		return err
	}

	return nil
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
