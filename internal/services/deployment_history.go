package services

import (
	"context"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

func ListDeploymentHistory(appID string) ([]entities.DeploymentHistory, error) {
	var histories []entities.DeploymentHistory
	if err := db.DB.Where("app_id = ?", appID).Order("created_at DESC").Find(&histories).Error; err != nil {
		return nil, err
	}
	return histories, nil
}

func CreateDeploymentHistory(history *entities.DeploymentHistory) error {
	history.ID = uuid.New()
	return db.DB.Create(history).Error
}

func RecordDeployment(appBefore, appAfter *entities.App, deployType, deployedBy, reason string, buildID *string) error {
	history := &entities.DeploymentHistory{
		AppID: appAfter.ID,

		ImageBefore:    appBefore.ContainerImage,
		ImageAfter:     appAfter.ContainerImage,
		ReplicasBefore: appBefore.Replicas,
		ReplicasAfter:  appAfter.Replicas,

		RequestCPUBefore:    appBefore.RequestCPU,
		RequestCPUAfter:     appAfter.RequestCPU,
		RequestMemoryBefore: appBefore.RequestMemory,
		RequestMemoryAfter:  appAfter.RequestMemory,
		LimitCPUBefore:      appBefore.LimitCPU,
		LimitCPUAfter:       appAfter.LimitCPU,
		LimitMemoryBefore:   appBefore.LimitMemory,
		LimitMemoryAfter:    appAfter.LimitMemory,

		DeployType: deployType,
		DeployedBy: deployedBy,
		Reason:     reason,
		Status:     "success",
		BuildID:    buildID,
	}

	return CreateDeploymentHistory(history)
}

func RollbackDeployment(appID, historyID string) (*entities.App, error) {
	var history entities.DeploymentHistory
	if err := db.DB.First(&history, "id = ? AND app_id = ?", historyID, appID).Error; err != nil {
		return nil, err
	}

	app, err := GetApp(appID)
	if err != nil {
		return nil, err
	}

	appBefore := &entities.App{
		ContainerImage: app.ContainerImage,
		Replicas:       app.Replicas,
		RequestCPU:     app.RequestCPU,
		RequestMemory:  app.RequestMemory,
		LimitCPU:       app.LimitCPU,
		LimitMemory:    app.LimitMemory,
	}

	app.ContainerImage = history.ImageBefore
	app.Replicas = history.ReplicasBefore
	app.RequestCPU = history.RequestCPUBefore
	app.RequestMemory = history.RequestMemoryBefore
	app.LimitCPU = history.LimitCPUBefore
	app.LimitMemory = history.LimitMemoryBefore

	if err := db.DB.Save(app).Error; err != nil {
		return nil, err
	}

	if err := core.ApplyApp(context.Background(), app); err != nil {
		return nil, err
	}

	if err := RecordDeployment(appBefore, app, "rollback", "system", "Rollback to previous deployment", nil); err != nil {
		return nil, err
	}

	return app, nil
}

func ConvertDeploymentHistoryToModel(entity *entities.DeploymentHistory) *models.DeploymentHistory {
	return &models.DeploymentHistory{
		ID:        entity.ID,
		AppID:     entity.AppID,
		CreatedAt: entity.CreatedAt,

		ImageBefore:    entity.ImageBefore,
		ImageAfter:     entity.ImageAfter,
		ReplicasBefore: entity.ReplicasBefore,
		ReplicasAfter:  entity.ReplicasAfter,

		RequestCPUBefore:    entity.RequestCPUBefore,
		RequestCPUAfter:     entity.RequestCPUAfter,
		RequestMemoryBefore: entity.RequestMemoryBefore,
		RequestMemoryAfter:  entity.RequestMemoryAfter,
		LimitCPUBefore:      entity.LimitCPUBefore,
		LimitCPUAfter:       entity.LimitCPUAfter,
		LimitMemoryBefore:   entity.LimitMemoryBefore,
		LimitMemoryAfter:    entity.LimitMemoryAfter,

		DeployType: entity.DeployType,
		DeployedBy: entity.DeployedBy,
		Reason:     entity.Reason,
		Status:     entity.Status,
		BuildID:    entity.BuildID,
	}
}
