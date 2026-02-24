package services

import (
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
)

func GetAdminDashboardStats() (*models.DashboardStatsResponse, error) {
	var stats models.DashboardStatsResponse

	if err := db.DB.Model(&entities.Cluster{}).Count(&stats.ClusterCount).Error; err != nil {
		return nil, err
	}
	if err := db.DB.Model(&entities.Project{}).Count(&stats.ProjectCount).Error; err != nil {
		return nil, err
	}
	if err := db.DB.Model(&entities.Env{}).Count(&stats.EnvironmentCount).Error; err != nil {
		return nil, err
	}
	if err := db.DB.Model(&entities.App{}).Count(&stats.ApplicationCount).Error; err != nil {
		return nil, err
	}
	if err := db.DB.Model(&entities.User{}).Count(&stats.UserCount).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

func GetUserDashboardStats(projectID string) (*models.DashboardStatsResponse, error) {
	var stats models.DashboardStatsResponse

	if err := db.DB.Model(&entities.Env{}).Where("project_id = ?", projectID).Count(&stats.EnvironmentCount).Error; err != nil {
		return nil, err
	}

	if err := db.DB.Model(&entities.App{}).
		Joins("JOIN envs ON apps.env_id = envs.id AND envs.deleted_at IS NULL").
		Where("envs.project_id = ?", projectID).
		Count(&stats.ApplicationCount).Error; err != nil {
		return nil, err
	}

	if err := db.DB.Model(&entities.ProjectMember{}).Where("project_id = ?", projectID).Count(&stats.MemberCount).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

func GetProjectEnvironmentsWithNamespaces(projectID string) ([]models.EnvironmentResourceUsage, error) {
	var envs []entities.Env
	if err := db.DB.Where("project_id = ?", projectID).Find(&envs).Error; err != nil {
		return nil, err
	}

	result := make([]models.EnvironmentResourceUsage, 0, len(envs))
	for _, env := range envs {
		result = append(result, models.EnvironmentResourceUsage{
			EnvironmentID:   env.ID,
			EnvironmentName: env.Name,
			Namespace:       env.ClusterNamespace,
		})
	}

	return result, nil
}
