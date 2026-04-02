package services

import (
	"errors"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
)

var (
	ErrProjectScopeRequired = errors.New("project_id is required")
	ErrProjectAccessDenied  = errors.New("insufficient permissions")
	ErrClusterProjectDenied = errors.New("cluster is not associated with the project")
)

func EnsureProjectAccess(userID, role, projectID string) error {
	if role == app.UserRoleAdmin {
		return nil
	}

	if strings.TrimSpace(projectID) == "" {
		return ErrProjectScopeRequired
	}

	isMember, err := IsProjectMember(projectID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrProjectAccessDenied
	}

	return nil
}

func EnsureClusterProjectAccess(userID, role, projectID, clusterID string) error {
	if role == app.UserRoleAdmin {
		return nil
	}

	if err := EnsureProjectAccess(userID, role, projectID); err != nil {
		return err
	}

	var count int64
	if err := db.DB.Model(&entities.Env{}).
		Where("project_id = ? AND cluster_id = ?", projectID, clusterID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrClusterProjectDenied
	}

	return nil
}

func ListProjectClustersSimple(projectID string) ([]models.SimpleCluster, error) {
	var clusters []models.SimpleCluster

	if err := db.DB.Model(&entities.Cluster{}).
		Select("clusters.id, clusters.slug, clusters.name, clusters.description, clusters.connection_status, clusters.enabled").
		Joins("JOIN envs ON envs.cluster_id = clusters.id").
		Where("envs.project_id = ?", projectID).
		Group("clusters.id, clusters.slug, clusters.name, clusters.description, clusters.connection_status, clusters.enabled").
		Order("clusters.created_at").
		Find(&clusters).Error; err != nil {
		return nil, err
	}

	return clusters, nil
}
