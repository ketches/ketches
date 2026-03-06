package services

import (
	"context"
	"errors"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

func ListAppVolumes(appID string) ([]entities.AppVolume, error) {
	var volumes []entities.AppVolume
	err := db.DB.Where("app_id = ?", appID).Find(&volumes).Error
	return volumes, err
}

func CreateAppVolume(appID string, req *models.CreateVolumeRequest) (*entities.AppVolume, error) {
	volumeMode := req.VolumeMode
	if volumeMode == "" {
		volumeMode = "Filesystem"
	}
	accessModes := req.AccessModes
	if accessModes == "" {
		accessModes = "ReadWriteOnce"
	}

	entity := &entities.AppVolume{
		ID:           uuid.New(),
		AppID:        appID,
		Slug:         req.Slug,
		VolumeType:   req.VolumeType,
		MountPath:    req.MountPath,
		SubPath:      req.SubPath,
		Capacity:     req.Capacity,
		StorageClass: req.StorageClass,
		VolumeMode:   volumeMode,
		AccessModes:  accessModes,
	}

	// Check slug uniqueness
	var existing entities.AppVolume
	err := db.DB.Where("app_id = ? AND slug = ?", appID, req.Slug).First(&existing).Error
	if err == nil {
		return nil, errors.New("volume with this slug already exists for this app")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Check mount path conflicts
	if err := checkVolumeMountPathConflicts(appID, req.MountPath, ""); err != nil {
		return nil, err
	}

	// Create in database
	if err := db.DB.Create(entity).Error; err != nil {
		return nil, err
	}

	// Load app with environment for K8s sync
	var app entities.App
	if err := db.DB.First(&app, "id = ?", appID).Error; err != nil {
		return nil, err
	}
	var env entities.Env
	if err := db.DB.First(&env, "id = ?", app.EnvID).Error; err != nil {
		return nil, err
	}
	app.Env = env
	if err := db.DB.Where("app_id = ?", appID).Find(&app.Volumes).Error; err != nil {
		return nil, err
	}

	// Sync PVC to Kubernetes if needed
	if req.VolumeType == "pvc" {
		if err := core.SyncVolumeToK8s(context.Background(), &app, entity); err != nil {
			return nil, err
		}
	}

	return entity, nil
}

func UpdateAppVolume(id string, req *models.UpdateVolumeRequest) (*entities.AppVolume, error) {
	var volume entities.AppVolume
	err := db.DB.First(&volume, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("volume not found")
		}
		return nil, err
	}

	// Check slug uniqueness (excluding current volume)
	if req.Slug != volume.Slug {
		var existing entities.AppVolume
		err := db.DB.Where("app_id = ? AND slug = ? AND id != ?", volume.AppID, req.Slug, id).First(&existing).Error
		if err == nil {
			return nil, errors.New("volume with this slug already exists for this app")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// Check mount path conflicts (excluding current volume)
	if req.MountPath != volume.MountPath {
		if err := checkVolumeMountPathConflicts(volume.AppID, req.MountPath, id); err != nil {
			return nil, err
		}
	}

	// Update fields
	volume.Slug = req.Slug
	volume.MountPath = req.MountPath
	volume.SubPath = req.SubPath
	volume.VolumeType = req.VolumeType
	volume.Capacity = req.Capacity
	volume.StorageClass = req.StorageClass
	volume.VolumeMode = req.VolumeMode
	volume.AccessModes = req.AccessModes

	// Save to database
	if err := db.DB.Save(&volume).Error; err != nil {
		return nil, err
	}

	// Load app with environment for K8s sync
	var app entities.App
	if err := db.DB.First(&app, "id = ?", volume.AppID).Error; err != nil {
		return nil, err
	}
	var env entities.Env
	if err := db.DB.First(&env, "id = ?", app.EnvID).Error; err != nil {
		return nil, err
	}
	app.Env = env
	if err := db.DB.Where("app_id = ?", volume.AppID).Find(&app.Volumes).Error; err != nil {
		return nil, err
	}

	// Sync PVC to Kubernetes if needed
	if volume.VolumeType == "pvc" {
		if err := core.SyncVolumeToK8s(context.Background(), &app, &volume); err != nil {
			return nil, err
		}
	}

	return &volume, nil
}

func DeleteAppVolume(id string) error {
	var volume entities.AppVolume
	err := db.DB.First(&volume, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("volume not found")
		}
		return err
	}

	// Load app with environment for K8s sync
	var app entities.App
	if err := db.DB.First(&app, "id = ?", volume.AppID).Error; err != nil {
		return err
	}
	var env entities.Env
	if err := db.DB.First(&env, "id = ?", app.EnvID).Error; err != nil {
		return err
	}
	app.Env = env

	// Delete PVC from Kubernetes if applicable
	if volume.VolumeType == "pvc" {
		if err := core.DeleteVolumeFromK8s(context.Background(), &app, &volume); err != nil {
			return err
		}
	}

	// Delete from database
	if err := db.DB.Delete(&volume).Error; err != nil {
		return err
	}

	return nil
}

// checkVolumeMountPathConflicts checks if the mount path conflicts with existing volumes or config files
func checkVolumeMountPathConflicts(appID, mountPath, excludeID string) error {
	// Check against other volumes
	var volumes []entities.AppVolume
	query := db.DB.Where("app_id = ?", appID)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Find(&volumes).Error; err != nil {
		return err
	}

	for _, vol := range volumes {
		if pathsConflict(mountPath, vol.MountPath) {
			return errors.New("mount path conflicts with existing volume: " + vol.MountPath)
		}
	}

	// Check against config files
	var configFiles []entities.AppConfigFile
	if err := db.DB.Where("app_id = ?", appID).Find(&configFiles).Error; err != nil {
		return err
	}

	for _, cf := range configFiles {
		if pathsConflict(mountPath, cf.MountPath) {
			return errors.New("mount path conflicts with existing config file: " + cf.MountPath)
		}
	}

	return nil
}
