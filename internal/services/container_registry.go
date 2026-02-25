package services

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

func ListClusterRegistries(clusterID string, page, pageSize int, search string) (int64, []entities.ContainerRegistry, error) {
	var registries []entities.ContainerRegistry
	var total int64
	query := db.DB.Model(&entities.ContainerRegistry{}).Where("cluster_id = ? AND scope = ?", clusterID, entities.RegistryScopeCluster)
	if search != "" {
		query = query.Where("name LIKE ? OR endpoint LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.Order("created_at asc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&registries).Error; err != nil {
		return 0, nil, err
	}
	return total, registries, nil
}

func ListProjectContainerRegistries(projectID string, page, pageSize int, search string) (int64, []entities.ContainerRegistry, error) {
	var registries []entities.ContainerRegistry
	var total int64
	query := db.DB.Model(&entities.ContainerRegistry{}).Where("project_id = ? AND scope = ?", projectID, entities.RegistryScopeProject)
	if search != "" {
		query = query.Where("name LIKE ? OR endpoint LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.Order("created_at asc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&registries).Error; err != nil {
		return 0, nil, err
	}
	return total, registries, nil
}

func ListProjectContainerRegistriesSimple(projectID string) ([]entities.ContainerRegistry, error) {
	var registries []entities.ContainerRegistry
	if err := db.DB.Select("id, name, endpoint, provider").Where("project_id = ? AND scope = ?", projectID, entities.RegistryScopeProject).Order("name").Find(&registries).Error; err != nil {
		return nil, err
	}
	return registries, nil
}

func ListAvailableRegistries(clusterID, projectID string) ([]entities.ContainerRegistry, error) {
	var registries []entities.ContainerRegistry
	if err := db.DB.Where(
		"(cluster_id = ? AND scope = ?) OR (project_id = ? AND scope = ?)",
		clusterID, entities.RegistryScopeCluster,
		projectID, entities.RegistryScopeProject,
	).Where("enabled = ?", true).Order("scope asc, created_at asc").Find(&registries).Error; err != nil {
		return nil, err
	}
	return registries, nil
}

func GetContainerRegistry(id string) (*entities.ContainerRegistry, error) {
	var registry entities.ContainerRegistry
	if err := db.DB.First(&registry, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &registry, nil
}

func CreateClusterRegistry(clusterID string, req *models.CreateContainerRegistryRequest) (*entities.ContainerRegistry, error) {
	cid := clusterID
	skipTLS := req.SkipTLSVerify != nil && *req.SkipTLSVerify
	registry := &entities.ContainerRegistry{
		Base:          entities.Base{ID: uuid.New()},
		Name:          req.Name,
		Provider:      entities.RegistryProvider(req.Provider),
		Endpoint:      ensureEndpointScheme(req.Endpoint),
		SkipTLSVerify: skipTLS,
		Namespace:     req.Namespace,
		Username:      req.Username,
		Password:      req.Password,
		Scope:         entities.RegistryScopeCluster,
		ClusterID:     &cid,
		ProjectID:     nil,
		IsDefault:     req.IsDefault,
		Enabled:       req.Enabled,
		Description:   req.Description,
	}

	if req.IsDefault {
		if err := clearDefaultRegistry(entities.RegistryScopeCluster, clusterID, ""); err != nil {
			return nil, err
		}
	}

	if err := db.DB.Create(registry).Error; err != nil {
		return nil, err
	}
	return registry, nil
}

func CreateProjectContainerRegistry(projectID string, req *models.CreateContainerRegistryRequest) (*entities.ContainerRegistry, error) {
	pid := projectID
	skipTLS := req.SkipTLSVerify != nil && *req.SkipTLSVerify
	registry := &entities.ContainerRegistry{
		Base:          entities.Base{ID: uuid.New()},
		Name:          req.Name,
		Provider:      entities.RegistryProvider(req.Provider),
		Endpoint:      ensureEndpointScheme(req.Endpoint),
		SkipTLSVerify: skipTLS,
		Namespace:     req.Namespace,
		Username:      req.Username,
		Password:      req.Password,
		Scope:         entities.RegistryScopeProject,
		ClusterID:     nil,
		ProjectID:     &pid,
		IsDefault:     req.IsDefault,
		Enabled:       req.Enabled,
		Description:   req.Description,
	}

	if req.IsDefault {
		if err := clearDefaultRegistry(entities.RegistryScopeProject, "", projectID); err != nil {
			return nil, err
		}
	}

	if err := db.DB.Create(registry).Error; err != nil {
		return nil, err
	}
	return registry, nil
}

func UpdateContainerRegistry(id string, req *models.UpdateContainerRegistryRequest) (*entities.ContainerRegistry, error) {
	registry, err := GetContainerRegistry(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		registry.Name = req.Name
	}
	if req.Provider != "" {
		registry.Provider = entities.RegistryProvider(req.Provider)
	}
	if req.Endpoint != "" {
		registry.Endpoint = ensureEndpointScheme(req.Endpoint)
	}
	if req.SkipTLSVerify != nil {
		registry.SkipTLSVerify = *req.SkipTLSVerify
	}
	registry.Namespace = req.Namespace
	if req.Username != "" {
		registry.Username = req.Username
	}
	if req.Password != "" {
		registry.Password = req.Password
	}
	if req.IsDefault != nil {
		if *req.IsDefault {
			cid, pid := "", ""
			if registry.ClusterID != nil {
				cid = *registry.ClusterID
			}
			if registry.ProjectID != nil {
				pid = *registry.ProjectID
			}
			if err := clearDefaultRegistry(registry.Scope, cid, pid); err != nil {
				return nil, err
			}
		}
		registry.IsDefault = *req.IsDefault
	}
	if req.Enabled != nil {
		registry.Enabled = *req.Enabled
	}
	registry.Description = req.Description

	if err := db.DB.Save(registry).Error; err != nil {
		return nil, err
	}
	return registry, nil
}

func DeleteContainerRegistry(id string) error {
	// Check if any build config references this registry
	var count int64
	if err := db.DB.Model(&entities.AppBuildConfig{}).Where("registry_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("cannot delete registry: it is referenced by build configurations")
	}
	return db.DB.Delete(&entities.ContainerRegistry{}, "id = ?", id).Error
}

// ensureEndpointScheme returns endpoint with https:// if it has no scheme (default TLS).
func ensureEndpointScheme(endpoint string) string {
	s := strings.TrimSpace(endpoint)
	if s == "" {
		return s
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s
	}
	return "https://" + s
}

func TestContainerRegistryConnection(req *models.TestContainerRegistryRequest) *models.TestContainerRegistryResponse {
	endpoint := ensureEndpointScheme(req.Endpoint)
	if req.Provider == string(entities.RegistryProviderDockerHub) {
		endpoint = "https://registry-1.docker.io/v2/"
	} else {
		endpoint = fmt.Sprintf("%s/v2/", strings.TrimSuffix(endpoint, "/"))
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: req.SkipTLSVerify},
		},
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		return &models.TestContainerRegistryResponse{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		}
	}
	defer resp.Body.Close()

	// 200 = open registry, 401 = auth required (registry exists)
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		return &models.TestContainerRegistryResponse{
			Success: true,
			Message: "Registry is reachable",
		}
	}

	return &models.TestContainerRegistryResponse{
		Success: false,
		Message: fmt.Sprintf("Registry returned status %d", resp.StatusCode),
	}
}

func clearDefaultRegistry(scope entities.RegistryScope, clusterID, projectID string) error {
	query := db.DB.Model(&entities.ContainerRegistry{}).Where("scope = ? AND is_default = ?", scope, true)
	if scope == entities.RegistryScopeCluster {
		query = query.Where("cluster_id = ?", clusterID)
	} else {
		query = query.Where("project_id = ?", projectID)
	}
	return query.Update("is_default", false).Error
}

func ToContainerRegistryResponse(r *entities.ContainerRegistry) models.ContainerRegistryResponse {
	cid, pid := "", ""
	if r.ClusterID != nil {
		cid = *r.ClusterID
	}
	if r.ProjectID != nil {
		pid = *r.ProjectID
	}
	return models.ContainerRegistryResponse{
		ID:            r.ID,
		Name:          r.Name,
		Provider:      string(r.Provider),
		Endpoint:      r.Endpoint,
		SkipTLSVerify: r.SkipTLSVerify,
		Namespace:     r.Namespace,
		Username:      r.Username,
		Password:      r.Password,
		Scope:         string(r.Scope),
		ClusterID:     cid,
		ProjectID:     pid,
		IsDefault:     r.IsDefault,
		Enabled:       r.Enabled,
		Description:   r.Description,
		CreatedAt:     r.CreatedAt,
	}
}
