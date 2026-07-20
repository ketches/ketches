package services

import (
	"errors"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

func ListClusterDomains(clusterID string, page, pageSize int, search string) (int64, []entities.Domain, error) {
	var items []entities.Domain
	var total int64
	query := db.DB.Model(&entities.Domain{}).Where("cluster_id = ? AND scope = ?", clusterID, "cluster")
	if search != "" {
		query = query.Where("name LIKE ? OR domain LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}

func ListEnvDomains(envID string, page, pageSize int, search string) (int64, []entities.Domain, error) {
	var items []entities.Domain
	var total int64
	query := db.DB.Model(&entities.Domain{}).Where("env_id = ? AND scope = ?", envID, "env")
	if search != "" {
		query = query.Where("name LIKE ? OR domain LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}

func GetDomain(envID, domainID string) (*entities.Domain, error) {
	var item entities.Domain
	if err := db.DB.Where("env_id = ? AND id = ? AND scope = ?", envID, domainID, "env").First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func CreateClusterDomain(clusterID string, req *models.CreateDomainRequest) (*entities.Domain, error) {
	cluster, err := GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	domain, err := normalizeDomainValue(req.Domain)
	if err != nil {
		return nil, err
	}

	var existing entities.Domain
	if err := db.DB.Where("cluster_id = ? AND scope = ? AND domain = ?", cluster.ID, "cluster", domain).First(&existing).Error; err == nil {
		return nil, errors.New("domain already exists in this cluster")
	}

	item := &entities.Domain{
		Base:        entities.Base{ID: uuid.New()},
		Name:        req.Name,
		Domain:      domain,
		Description: req.Description,
		Scope:       "cluster",
		ClusterID:   cluster.ID,
	}

	if err := db.DB.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func CreateEnvDomain(envID string, req *models.CreateDomainRequest) (*entities.Domain, error) {
	env, err := GetEnv(envID)
	if err != nil {
		return nil, err
	}
	if env.ClusterID == "" {
		return nil, errors.New("environment is not bound to a cluster")
	}

	domain, err := normalizeDomainValue(req.Domain)
	if err != nil {
		return nil, err
	}

	var existing entities.Domain
	if err := db.DB.Where("env_id = ? AND scope = ? AND domain = ?", envID, "env", domain).First(&existing).Error; err == nil {
		return nil, errors.New("domain already exists in this environment")
	}

	envIDCopy := envID
	item := &entities.Domain{
		Base:        entities.Base{ID: uuid.New()},
		Name:        req.Name,
		Domain:      domain,
		Description: req.Description,
		Scope:       "env",
		ClusterID:   env.ClusterID,
		EnvID:       &envIDCopy,
	}

	if err := db.DB.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func UpdateDomain(envID, domainID string, req *models.UpdateDomainRequest) (*entities.Domain, error) {
	item, err := GetDomain(envID, domainID)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = *req.Name
		item.Name = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
		item.Description = *req.Description
	}
	if req.Domain != nil {
		domain, err := normalizeDomainValue(*req.Domain)
		if err != nil {
			return nil, err
		}
		var existing entities.Domain
		query := db.DB.Where("scope = ? AND cluster_id = ? AND domain = ? AND id != ?", item.Scope, item.ClusterID, domain, item.ID)
		if item.Scope == "env" && item.EnvID != nil {
			query = query.Where("env_id = ?", *item.EnvID)
		}
		if err := query.First(&existing).Error; err == nil {
			return nil, errors.New("domain already exists in this scope")
		}
		updates["domain"] = domain
		item.Domain = domain
	}

	if len(updates) > 0 {
		if err := db.DB.Model(&entities.Domain{}).
			Where("env_id = ? AND id = ? AND scope = ?", envID, domainID, "env").
			Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	if err := db.DB.Where("env_id = ? AND id = ? AND scope = ?", envID, domainID, "env").First(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func DeleteDomain(envID, domainID string) error {
	if _, err := GetDomain(envID, domainID); err != nil {
		return err
	}
	return db.DB.Where("env_id = ? AND id = ? AND scope = ?", envID, domainID, "env").Delete(&entities.Domain{}).Error
}
