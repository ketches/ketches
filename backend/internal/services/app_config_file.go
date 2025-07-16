package services

import (
	"context"
	"log"
	"net/http"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/db/orm"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

type AppConfigFileService interface {
	ListAppConfigFiles(ctx context.Context, req *models.ListAppConfigFilesRequest) ([]*models.AppConfigFileModel, app.Error)
	CreateAppConfigFile(ctx context.Context, req *models.CreateAppConfigFileRequest) (*models.AppConfigFileModel, app.Error)
	UpdateAppConfigFile(ctx context.Context, req *models.UpdateAppConfigFileRequest) (*models.AppConfigFileModel, app.Error)
	DeleteAppConfigFiles(ctx context.Context, req *models.DeleteAppConfigFilesRequest) app.Error
}

type appConfigFileService struct {
	Service
}

var appConfigFileServiceInstance = &appConfigFileService{
	Service: LoadService(),
}

func NewAppConfigFileService() AppConfigFileService {
	return appConfigFileServiceInstance
}

// ListAppConfigFiles returns all config files for an app
func (s *appConfigFileService) ListAppConfigFiles(ctx context.Context, req *models.ListAppConfigFilesRequest) ([]*models.AppConfigFileModel, app.Error) {
	var result []*models.AppConfigFileModel
	var total int64
	if err := db.Instance().Model(&entities.AppConfigFile{}).
		Where("app_id = ?", req.AppID).
		Count(&total).
		Find(&result).Error; err != nil {
		log.Printf("failed to list config files for app %s: %v", req.AppID, err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

// CreateAppConfigFile creates a new config file for an app
func (s *appConfigFileService) CreateAppConfigFile(ctx context.Context, req *models.CreateAppConfigFileRequest) (*models.AppConfigFileModel, app.Error) {
	// Validate file mode
	if !isValidFileMode(req.FileMode) {
		return nil, app.NewError(http.StatusBadRequest, "invalid file mode")
	}

	// Check content size (950KB limit)
	if len(req.Content) > 972800 {
		return nil, app.NewError(http.StatusBadRequest, "file content exceeds 950KB limit")
	}

	entity := &entities.AppConfigFile{
		UUIDBase:  entities.UUIDBase{ID: uuid.New()},
		AppID:     req.AppID,
		Slug:      req.Slug,
		Content:   req.Content,
		MountPath: req.MountPath,
		FileMode:  req.FileMode,
		AuditBase: entities.AuditBase{
			CreatedBy: api.UserID(ctx),
			UpdatedBy: api.UserID(ctx),
		},
	}

	if err := db.Instance().Create(entity).Error; err != nil {
		log.Printf("failed to create app config file for app %s: %v", req.AppID, err)
		if db.IsErrDuplicatedKey(err) {
			return nil, app.NewError(http.StatusBadRequest, "config file slug or mount path already exists")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	if _, err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after creating config file for app %s: %v", req.AppID, err)
	}

	return &models.AppConfigFileModel{
		ConfigFileID: entity.ID,
		AppID:        entity.AppID,
		Slug:         entity.Slug,
		Content:      entity.Content,
		MountPath:    entity.MountPath,
		FileMode:     entity.FileMode,
	}, nil
}

func (s *appConfigFileService) UpdateAppConfigFile(ctx context.Context, req *models.UpdateAppConfigFileRequest) (*models.AppConfigFileModel, app.Error) {
	// Validate file mode
	if !isValidFileMode(req.FileMode) {
		return nil, app.NewError(http.StatusBadRequest, "invalid file mode")
	}

	// Check content size (950KB limit)
	if len(req.Content) > 972800 {
		return nil, app.NewError(http.StatusBadRequest, "file content exceeds 950KB limit")
	}

	var entity entities.AppConfigFile
	if err := db.Instance().First(&entity, "id = ?", req.ConfigFileID).Error; err != nil {
		log.Printf("failed to find app config file %s: %v", req.ConfigFileID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "config file not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	entity.Content = req.Content
	entity.MountPath = req.MountPath
	entity.FileMode = req.FileMode

	if err := db.Instance().Model(&entity).Select("FileName", "Content", "MountPath", "SubPath", "FileMode", "UpdatedBy").Updates(&entities.AppConfigFile{
		Content:   req.Content,
		MountPath: req.MountPath,
		FileMode:  req.FileMode,
		AuditBase: entities.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
	}).Error; err != nil {
		log.Printf("failed to update app config file %s: %v", req.ConfigFileID, err)
		if db.IsErrDuplicatedKey(err) {
			return nil, app.NewError(http.StatusBadRequest, "mount path already exists")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	if _, err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after updating config file for app %s: %v", req.AppID, err)
	}

	return &models.AppConfigFileModel{
		ConfigFileID: entity.ID,
		AppID:        entity.AppID,
		Slug:         entity.Slug,
		Content:      entity.Content,
		MountPath:    entity.MountPath,
		FileMode:     entity.FileMode,
	}, nil
}

func (s *appConfigFileService) DeleteAppConfigFiles(ctx context.Context, req *models.DeleteAppConfigFilesRequest) app.Error {
	if len(req.ConfigFileIDs) == 0 {
		return nil
	}

	if err := db.Instance().Delete(&entities.AppConfigFile{}, req.ConfigFileIDs).Error; err != nil {
		log.Printf("failed to delete app config files: %v", err)
		return app.ErrDatabaseOperationFailed
	}

	if _, err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after deleting config file for app %s: %v", req.AppID, err)
	}

	return nil
}

// isValidFileMode validates if the file mode is a valid octal permission string
func isValidFileMode(mode string) bool {
	if len(mode) != 4 || mode[0] != '0' {
		return false
	}
	for i := 1; i < 4; i++ {
		if mode[i] < '0' || mode[i] > '7' {
			return false
		}
	}
	return true
}
