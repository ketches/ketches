package services

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/db/orm"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

type AppVolumeService interface {
	ListAppVolumes(ctx context.Context, req *models.ListAppVolumesRequest) ([]*models.AppVolumeModel, app.Error)
	CreateAppVolume(ctx context.Context, req *models.CreateAppVolumeRequest) (*models.AppVolumeModel, app.Error)
	UpdateAppVolume(ctx context.Context, req *models.UpdateAppVolumeRequest) (*models.AppVolumeModel, app.Error)
	DeleteAppVolumes(ctx context.Context, req *models.DeleteAppVolumesRequest) app.Error
}

type appVolumeService struct{}

var appVolumeServiceInstance = &appVolumeService{}

func NewAppVolumeService() AppVolumeService {
	return appVolumeServiceInstance
}

func (s *appVolumeService) ListAppVolumes(ctx context.Context, req *models.ListAppVolumesRequest) ([]*models.AppVolumeModel, app.Error) {
	var result []*entities.AppVolume
	if err := db.Instance().Model(&entities.AppVolume{}).
		Where("app_id = ?", req.AppID).
		Find(&result).Error; err != nil {
		log.Println("failed to list app volumes:", err)
		return nil, app.ErrDatabaseOperationFailed
	}
	var records []*models.AppVolumeModel
	for _, v := range result {
		records = append(records, &models.AppVolumeModel{
			VolumeID:   v.ID,
			AppID:      v.AppID,
			Slug:       v.Slug,
			MountPath:  v.MountPath,
			SubPath:    v.SubPath,
			Capacity:   v.Capacity,
			VolumeType: v.VolumeType,
			AccessModes: func() []string {
				if v.AccessModes == "" {
					return nil
				}
				return strings.Split(v.AccessModes, ";")
			}(),
			VolumeMode:   v.VolumeMode,
			StorageClass: v.StorageClass,
		})
	}
	return records, nil
}

func (s *appVolumeService) CreateAppVolume(ctx context.Context, req *models.CreateAppVolumeRequest) (*models.AppVolumeModel, app.Error) {
	entity := &entities.AppVolume{
		UUIDBase:     entities.UUIDBase{ID: uuid.New()},
		AppID:        req.AppID,
		Slug:         req.Slug,
		MountPath:    req.MountPath,
		SubPath:      req.SubPath,
		Capacity:     req.Capacity,
		VolumeType:   req.VolumeType,
		AccessModes:  strings.Join(req.AccessModes, ";"),
		StorageClass: req.StorageClass,
		VolumeMode:   req.VolumeMode,
		AuditBase: entities.AuditBase{
			CreatedBy: api.UserID(ctx),
			UpdatedBy: api.UserID(ctx),
		},
	}
	if err := db.Instance().Create(entity).Error; err != nil {
		log.Println("failed to create app volume:", err)
		if db.IsErrDuplicatedKey(err) {
			return nil, app.NewError(http.StatusBadRequest, "volume slug or mount path already exists")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	if err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after creating volume for app %s: %v", req.AppID, err)
	}

	return &models.AppVolumeModel{
		VolumeID:     entity.ID,
		AppID:        entity.AppID,
		Slug:         entity.Slug,
		MountPath:    entity.MountPath,
		Capacity:     entity.Capacity,
		VolumeType:   entity.VolumeType,
		AccessModes:  splitAccessModes(entity.AccessModes),
		StorageClass: entity.StorageClass,
	}, nil
}

func (s *appVolumeService) UpdateAppVolume(ctx context.Context, req *models.UpdateAppVolumeRequest) (*models.AppVolumeModel, app.Error) {
	var entity entities.AppVolume
	if err := db.Instance().First(&entity, "id = ?", req.VolumeID).Error; err != nil {
		log.Printf("failed to find app volume %s: %v", req.VolumeID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "volume not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}
	entity.MountPath = req.MountPath
	entity.SubPath = req.SubPath

	if err := db.Instance().Model(&entity).Select("MountPath", "SubPath", "UpdatedBy").Updates(&entities.AppVolume{
		MountPath: req.MountPath,
		SubPath:   req.SubPath,
		AuditBase: entities.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
	}).Error; err != nil {
		log.Printf("failed to update app volume %s: %v", req.VolumeID, err)
		if db.IsErrDuplicatedKey(err) {
			return nil, app.NewError(http.StatusBadRequest, "volume mount path already exists")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	if err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after updating volume for app %s: %v", req.AppID, err)
	}

	return &models.AppVolumeModel{
		VolumeID:     entity.ID,
		AppID:        entity.AppID,
		Slug:         entity.Slug,
		MountPath:    entity.MountPath,
		SubPath:      entity.SubPath,
		Capacity:     entity.Capacity,
		VolumeType:   entity.VolumeType,
		AccessModes:  splitAccessModes(entity.AccessModes),
		StorageClass: entity.StorageClass,
	}, nil
}

func (s *appVolumeService) DeleteAppVolumes(ctx context.Context, req *models.DeleteAppVolumesRequest) app.Error {
	if len(req.VolumeIDs) == 0 {
		return nil
	}

	if err := db.Instance().Delete(&entities.AppVolume{}, req.VolumeIDs).Error; err != nil {
		log.Println("failed to delete app volumes:", err)
		return app.ErrDatabaseOperationFailed
	}

	if err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after deleting volume for app %s: %v", req.AppID, err)
	}

	return nil
}

func splitAccessModes(modes string) []string {
	if modes == "" {
		return nil
	}
	return strings.Split(modes, ";")
}
