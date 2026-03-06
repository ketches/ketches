package services

import (
	"context"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

func ListDeploymentHistory(appID string, page, pageSize int) (int64, []entities.DeploymentHistory, error) {
	var total int64
	var histories []entities.DeploymentHistory
	query := db.DB.Model(&entities.DeploymentHistory{}).Where("app_id = ?", appID)
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&histories).Error; err != nil {
		return 0, nil, err
	}
	return total, histories, nil
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

	appCtx, err := GetApp(context.Background(), appID)
	if err != nil {
		return nil, err
	}

	appBefore := &entities.App{
		ContainerImage: appCtx.App.ContainerImage,
		Replicas:       appCtx.App.Replicas,
		RequestCPU:     appCtx.App.RequestCPU,
		RequestMemory:  appCtx.App.RequestMemory,
		LimitCPU:       appCtx.App.LimitCPU,
		LimitMemory:    appCtx.App.LimitMemory,
	}

	appCtx.App.ContainerImage = history.ImageBefore
	appCtx.App.Replicas = history.ReplicasBefore
	appCtx.App.RequestCPU = history.RequestCPUBefore
	appCtx.App.RequestMemory = history.RequestMemoryBefore
	appCtx.App.LimitCPU = history.LimitCPUBefore
	appCtx.App.LimitMemory = history.LimitMemoryBefore

	if err := db.DB.Save(&appCtx.App).Error; err != nil {
		return nil, err
	}

	if err := core.ApplyApp(context.Background(), appCtx); err != nil {
		return nil, err
	}

	if err := RecordDeployment(appBefore, &appCtx.App, "rollback", "system", "Rollback to previous deployment", nil); err != nil {
		return nil, err
	}

	return &appCtx.App, nil
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
