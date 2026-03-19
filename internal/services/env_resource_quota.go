package services

import (
	"context"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

func GetResourceQuota(envID string) (*models.ResourceQuotaResponse, error) {
	var quota entities.EnvResourceQuota
	err := db.DB.Where("env_id = ?", envID).First(&quota).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &models.ResourceQuotaResponse{
				CPURequest:    entities.DefaultCPURequest,
				CPULimit:      entities.DefaultCPULimit,
				MemoryRequest: entities.DefaultMemoryRequest,
				MemoryLimit:   entities.DefaultMemoryLimit,
				Pods:          entities.DefaultPods,
			}, nil
		}
		return nil, err
	}

	return &models.ResourceQuotaResponse{
		CPURequest:    quota.CPURequest,
		CPULimit:      quota.CPULimit,
		MemoryRequest: quota.MemoryRequest,
		MemoryLimit:   quota.MemoryLimit,
		Pods:          quota.Pods,
	}, nil
}

func UpdateResourceQuota(envID string, req *models.UpdateResourceQuotaRequest) (*models.ResourceQuotaResponse, error) {
	env, err := GetEnv(envID)
	if err != nil {
		return nil, err
	}

	var quota entities.EnvResourceQuota
	err = db.DB.Where("env_id = ?", envID).First(&quota).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			quota = entities.EnvResourceQuota{
				Base:          entities.Base{ID: uuid.New()},
				EnvID:         envID,
				CPURequest:    req.CPURequest,
				CPULimit:      req.CPULimit,
				MemoryRequest: req.MemoryRequest,
				MemoryLimit:   req.MemoryLimit,
				Pods:          req.Pods,
			}
			if err := db.DB.Create(&quota).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		quota.CPURequest = req.CPURequest
		quota.CPULimit = req.CPULimit
		quota.MemoryRequest = req.MemoryRequest
		quota.MemoryLimit = req.MemoryLimit
		quota.Pods = req.Pods
		if err := db.DB.Save(&quota).Error; err != nil {
			return nil, err
		}
	}

	if err := core.ApplyResourceQuota(context.Background(), env.ClusterID, env.ClusterNamespace, req); err != nil {
		return nil, err
	}

	return &models.ResourceQuotaResponse{
		CPURequest:    req.CPURequest,
		CPULimit:      req.CPULimit,
		MemoryRequest: req.MemoryRequest,
		MemoryLimit:   req.MemoryLimit,
		Pods:          req.Pods,
	}, nil
}

func CreateDefaultEnvResourceQuota(envID string) error {
	quota := entities.EnvResourceQuota{
		Base:          entities.Base{ID: uuid.New()},
		EnvID:         envID,
		CPURequest:    entities.DefaultCPURequest,
		CPULimit:      entities.DefaultCPULimit,
		MemoryRequest: entities.DefaultMemoryRequest,
		MemoryLimit:   entities.DefaultMemoryLimit,
		Pods:          entities.DefaultPods,
	}
	return db.DB.Create(&quota).Error
}
