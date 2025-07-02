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

type AppProbeService interface {
	ListAppProbes(ctx context.Context, req *models.ListAppProbesRequest) ([]*models.AppProbeModel, app.Error)
	CreateAppProbe(ctx context.Context, req *models.CreateAppProbeRequest) (*models.AppProbeModel, app.Error)
	UpdateAppProbe(ctx context.Context, req *models.UpdateAppProbeRequest) (*models.AppProbeModel, app.Error)
	ToggleAppProbe(ctx context.Context, req *models.ToggleAppProbeRequest) (*models.AppProbeModel, app.Error)
	DeleteAppProbe(ctx context.Context, req *models.DeleteAppProbeRequest) app.Error
}

type appProbeService struct {
	Service
}

var appProbeServiceInstance = &appProbeService{
	Service: LoadService(),
}

func NewAppProbeService() AppProbeService {
	return appProbeServiceInstance
}

func (s *appProbeService) ListAppProbes(ctx context.Context, req *models.ListAppProbesRequest) ([]*models.AppProbeModel, app.Error) {
	var result []*models.AppProbeModel
	if err := db.Instance().Model(&entities.AppProbe{}).
		Where("app_id = ?", req.AppID).
		Find(&result).Error; err != nil {
		log.Printf("failed to list probes for app %s: %v", req.AppID, err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func (s *appProbeService) CreateAppProbe(ctx context.Context, req *models.CreateAppProbeRequest) (*models.AppProbeModel, app.Error) {
	entity := &entities.AppProbe{
		UUIDBase:            entities.UUIDBase{ID: uuid.New()},
		AppID:               req.AppID,
		Enabled:             req.Enabled,
		Type:                req.Type,
		InitialDelaySeconds: req.Probe.InitialDelaySeconds,
		TimeoutSeconds:      req.Probe.TimeoutSeconds,
		PeriodSeconds:       req.Probe.PeriodSeconds,
		SuccessThreshold:    req.Probe.SuccessThreshold,
		FailureThreshold:    req.Probe.FailureThreshold,
		ProbeMode:           req.Probe.ProbeMode,
		AuditBase: entities.AuditBase{
			CreatedBy: api.UserID(ctx),
			UpdatedBy: api.UserID(ctx),
		},
	}
	switch req.Probe.ProbeMode {
	case app.AppProbeModeHTTPGet:
		entity.HTTPGetPort = req.Probe.HTTPGetPort
		entity.HTTPGetPath = req.Probe.HTTPGetPath
	case app.AppProbeModeTCPSocket:
		entity.TCPSocketPort = req.Probe.TCPSocketPort
	case app.AppProbeModeExec:
		entity.ExecCommand = req.Probe.ExecCommand
	default:
		log.Printf("invalid probe mode: %s", req.Probe.ProbeMode)
		return nil, app.NewError(http.StatusBadRequest, "invalid probe mode: "+req.Probe.ProbeMode)
	}
	if err := db.Instance().Create(entity).Error; err != nil {
		log.Printf("failed to create app probe for app %s: %v", req.AppID, err)
		return nil, app.ErrDatabaseOperationFailed
	}

	if _, err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after creating probe for app %s: %v", req.AppID, err)
	}

	return &models.AppProbeModel{
		ProbeID: entity.ID,
		AppID:   entity.AppID,
		Type:    entity.Type,
		Enabled: entity.Enabled,
		Probe:   req.Probe,
	}, nil
}

func (s *appProbeService) UpdateAppProbe(ctx context.Context, req *models.UpdateAppProbeRequest) (*models.AppProbeModel, app.Error) {
	var entity entities.AppProbe
	if err := db.Instance().First(&entity, "id = ?", req.ProbeID).Error; err != nil {
		log.Printf("failed to find app probe %s: %v", req.ProbeID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "probe not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	entity.Enabled = req.Enabled
	entity.Type = req.Type
	entity.ProbeMode = req.Probe.ProbeMode
	switch entity.ProbeMode {
	case app.AppProbeModeHTTPGet:
		entity.HTTPGetPath = req.Probe.HTTPGetPath
		entity.HTTPGetPort = req.Probe.HTTPGetPort

		// reset other fields if switching to HTTP Get
		entity.TCPSocketPort = 0
		entity.ExecCommand = ""
	case app.AppProbeModeTCPSocket:
		entity.TCPSocketPort = req.Probe.TCPSocketPort

		// reset other fields if switching to TCP socket
		entity.ExecCommand = ""
		entity.HTTPGetPath = ""
		entity.HTTPGetPort = 0
	case app.AppProbeModeExec:
		entity.ExecCommand = req.Probe.ExecCommand

		// reset other fields if switching to Exec
		entity.HTTPGetPath = ""
		entity.HTTPGetPort = 0
		entity.TCPSocketPort = 0
	default:
		return nil, app.NewError(http.StatusBadRequest, "invalid probe mode: "+entity.ProbeMode)
	}

	entity.InitialDelaySeconds = req.Probe.InitialDelaySeconds
	entity.TimeoutSeconds = req.Probe.TimeoutSeconds
	entity.PeriodSeconds = req.Probe.PeriodSeconds
	entity.SuccessThreshold = req.Probe.SuccessThreshold
	entity.FailureThreshold = req.Probe.FailureThreshold

	entity.AuditBase.UpdatedBy = api.UserID(ctx)

	if err := db.Instance().Model(&entity).Select("Enabled", "Type", "ProbeMode", "HTTPGetPath", "HTTPGetPort", "TCPSocketPort", "ExecCommand", "InitialDelaySeconds", "TimeoutSeconds", "PeriodSeconds",
		"SuccessThreshold", "FailureThreshold", "UpdatedBy").
		Updates(&entity).Error; err != nil {
		log.Printf("failed to update probe var %s: %v", req.ProbeID, err)
		return nil, app.ErrDatabaseOperationFailed
	}

	if _, err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after updating probe var for app %s: %v", req.AppID, err)
	}

	return &models.AppProbeModel{
		ProbeID: entity.ID,
		AppID:   entity.AppID,
		Type:    entity.Type,
		Enabled: entity.Enabled,
		Probe:   req.Probe,
	}, nil
}

func (s *appProbeService) ToggleAppProbe(ctx context.Context, req *models.ToggleAppProbeRequest) (*models.AppProbeModel, app.Error) {
	var entity entities.AppProbe
	if err := db.Instance().First(&entity, "id = ?", req.ProbeID).Error; err != nil {
		log.Printf("failed to find app probe %s: %v", req.ProbeID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "probe not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	entity.Enabled = req.Enabled
	entity.AuditBase.UpdatedBy = api.UserID(ctx)

	if err := db.Instance().Model(&entity).Select("Enabled", "UpdatedBy").
		Updates(&entity).Error; err != nil {
		log.Printf("failed to update probe var %s: %v", req.ProbeID, err)
		return nil, app.ErrDatabaseOperationFailed
	}

	if _, err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after updating probe var for app %s: %v", req.AppID, err)
	}

	return &models.AppProbeModel{
		ProbeID: entity.ID,
		AppID:   entity.AppID,
		Type:    entity.Type,
		Enabled: entity.Enabled,
	}, nil
}

func (s *appProbeService) DeleteAppProbe(ctx context.Context, req *models.DeleteAppProbeRequest) app.Error {
	if err := db.Instance().Delete(&entities.AppProbe{}, "id = ?", req.ProbeID).Error; err != nil {
		log.Printf("failed to delete app probe for app %s: %v", req.AppID, err)
		return app.ErrDatabaseOperationFailed
	}

	if _, err := orm.UpdateAppEdition(ctx, req.AppID); err != nil {
		log.Printf("failed to update app edition after creating env var for app %s: %v", req.AppID, err)
	}

	return nil
}
