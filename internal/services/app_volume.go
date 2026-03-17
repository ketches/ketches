package services

import (
	"context"
	"errors"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ListAppVolumes(appID string) ([]models.AppVolumeResponse, error) {
	var volumes []entities.AppVolume
	err := db.DB.Where("app_id = ?", appID).Find(&volumes).Error
	if err != nil {
		return nil, err
	}

	pvcStatuses, pvcStatusFetchSucceeded := make(map[string]string), false
	if hasPVCVolumes(volumes) {
		appCtx, err := GetAppContext(context.Background(), appID)
		if err == nil {
			pvcStatuses, pvcStatusFetchSucceeded = listPVCStatuses(context.Background(), appCtx)
		}
	}

	result := make([]models.AppVolumeResponse, 0, len(volumes))
	for _, vol := range volumes {
		res := toAppVolumeResponse(&vol)
		if vol.VolumeType == "pvc" {
			switch {
			case pvcStatusFetchSucceeded:
				res.Status = pvcStatuses[vol.Slug]
				if res.Status == "" {
					res.Status = "NotFound"
				}
			default:
				res.Status = "Unknown"
			}
		}
		result = append(result, res)
	}
	return result, nil
}

func CreateAppVolume(ctx context.Context, appID string, req *models.CreateVolumeRequest) (*models.AppVolumeResponse, error) {
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

	// Fetch full app context from DB
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	// Sync PVC to Kubernetes if needed
	if req.VolumeType == "pvc" {
		if err := core.SyncVolumeToK8s(ctx, appCtx, entity); err != nil {
			return nil, err
		}
	}

	res := toAppVolumeResponse(entity)
	return &res, nil
}

func UpdateAppVolume(ctx context.Context, id string, req *models.UpdateVolumeRequest) (*models.AppVolumeResponse, error) {
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

	// Fetch full app context from DB
	appCtx, err := GetAppContext(ctx, volume.AppID)
	if err != nil {
		return nil, err
	}

	// Sync PVC to Kubernetes if needed
	if volume.VolumeType == "pvc" {
		if err := core.SyncVolumeToK8s(ctx, appCtx, &volume); err != nil {
			return nil, err
		}
	}

	res := toAppVolumeResponse(&volume)
	return &res, nil
}

func DeleteAppVolume(ctx context.Context, id string) error {
	var volume entities.AppVolume
	err := db.DB.First(&volume, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("volume not found")
		}
		return err
	}

	// Fetch full app context from DB
	appCtx, err := GetAppContext(ctx, volume.AppID)
	if err != nil {
		return err
	}

	// Delete PVC from Kubernetes if applicable
	if volume.VolumeType == "pvc" {
		if err := core.DeleteVolumeFromK8s(ctx, appCtx, &volume); err != nil {
			return err
		}
	}

	// Delete from database
	if err := db.DB.Delete(&volume).Error; err != nil {
		return err
	}

	return nil
}

// toAppVolumeResponse converts an AppVolume entity to a response model with snake_case JSON fields.
func toAppVolumeResponse(vol *entities.AppVolume) models.AppVolumeResponse {
	return models.AppVolumeResponse{
		ID:           vol.ID,
		AppID:        vol.AppID,
		Slug:         vol.Slug,
		MountPath:    vol.MountPath,
		SubPath:      vol.SubPath,
		VolumeType:   vol.VolumeType,
		Status:       "",
		Capacity:     vol.Capacity,
		StorageClass: vol.StorageClass,
		VolumeMode:   vol.VolumeMode,
		AccessModes:  vol.AccessModes,
		CreatedAt:    vol.CreatedAt,
		UpdatedAt:    vol.UpdatedAt,
	}
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

func hasPVCVolumes(volumes []entities.AppVolume) bool {
	for _, volume := range volumes {
		if volume.VolumeType == "pvc" {
			return true
		}
	}

	return false
}

func listPVCStatuses(ctx context.Context, appCtx *models.AppContext) (map[string]string, bool) {
	clusterID := appCtx.EnvContext.Env.ClusterID
	namespace := appCtx.EnvContext.Env.ClusterNamespace
	if clusterID == "" || namespace == "" {
		return nil, false
	}

	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return nil, false
	}

	pvcList, err := client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, false
	}

	statuses := make(map[string]string, len(pvcList.Items))
	for _, pvc := range pvcList.Items {
		statuses[pvc.Name] = string(pvc.Status.Phase)
		if pvc.Status.Phase == "" {
			statuses[pvc.Name] = string(corev1.ClaimPending)
		}
	}

	return statuses, true
}
