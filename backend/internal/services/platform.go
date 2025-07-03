package services

import (
	"context"
	"log"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
)

type PlatformService interface {
	GetStatistics(ctx context.Context) (*models.PlatformStatisticsModel, app.Error)
}

type platformService struct {
	Service
}

func NewPlatformService() PlatformService {
	return &platformService{
		Service: LoadService(),
	}
}

func (s *platformService) GetStatistics(ctx context.Context) (*models.PlatformStatisticsModel, app.Error) {
	stats := &models.PlatformStatisticsModel{}

	// Get total clusters
	if err := db.Instance().Model(&entities.Cluster{}).Count(&stats.TotalClusters).Error; err != nil {
		log.Printf("failed to count clusters: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	// Get total projects
	if err := db.Instance().Model(&entities.Project{}).Count(&stats.TotalProjects).Error; err != nil {
		log.Printf("failed to count projects: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	// Get total users
	if err := db.Instance().Model(&entities.User{}).Count(&stats.TotalUsers).Error; err != nil {
		log.Printf("failed to count users: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	// Get total environments
	if err := db.Instance().Model(&entities.Env{}).Count(&stats.TotalEnvs).Error; err != nil {
		log.Printf("failed to count environments: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	// Get total apps
	if err := db.Instance().Model(&entities.App{}).Count(&stats.TotalApps).Error; err != nil {
		log.Printf("failed to count apps: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	// Get total app gateways
	if err := db.Instance().Model(&entities.AppGateway{}).Count(&stats.TotalAppGateways).Error; err != nil {
		log.Printf("failed to count app gateways: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return stats, nil
}
