package services

import (
	"errors"
	"fmt"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

// builtinExtensions is the seed list of platform built-in OCI chart extensions.
var builtinExtensions = []entities.Extension{
	{
		Name:        "gateway-helm",
		DisplayName: "Envoy Gateway",
		Description: "Envoy Gateway is an open source project for managing Envoy Proxy as a standalone or Kubernetes-based application gateway.",
		OCIUrl:      "oci://docker.io/envoyproxy/gateway-helm",
		Builtin:     true,
	},
}

// SeedBuiltinExtensions inserts built-in extensions if they don't already exist.
func SeedBuiltinExtensions() error {
	for _, item := range builtinExtensions {
		var existing entities.Extension
		err := db.DB.Where("name = ?", item.Name).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to query extension %q: %w", item.Name, err)
		}
		item.ID = uuid.New()
		if err := db.DB.Create(&item).Error; err != nil {
			return fmt.Errorf("failed to seed built-in extension %q: %w", item.Name, err)
		}
	}
	return nil
}

// ListExtensions returns all catalog extensions (builtin + admin-added) with install counts from DB.
func ListExtensions() ([]models.Extension, error) {
	var items []entities.Extension
	if err := db.DB.Order("builtin DESC, created_at ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list extensions: %w", err)
	}

	// Count installs per extension from the cluster_extensions table (fast, no Helm calls).
	type installCountRow struct {
		ExtensionID string
		Count       int
	}
	var counts []installCountRow
	db.DB.Model(&entities.ClusterExtension{}).
		Select("extension_id, COUNT(*) as count").
		Where("deleted_at IS NULL").
		Group("extension_id").
		Scan(&counts)
	installMap := make(map[string]int, len(counts))
	for _, row := range counts {
		installMap[row.ExtensionID] = row.Count
	}

	result := make([]models.Extension, 0, len(items))
	for _, item := range items {
		m := toExtensionModel(&item)
		m.InstallCount = installMap[m.ID]
		result = append(result, m)
	}
	return result, nil
}

// GetExtension returns a single catalog extension by ID.
func GetExtension(extensionID string) (*models.Extension, error) {
	var item entities.Extension
	if err := db.DB.Where("id = ?", extensionID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("extension not found")
		}
		return nil, err
	}
	m := toExtensionModel(&item)
	return &m, nil
}

// GetExtensionEntity returns the raw extension entity by ID (used by other services).
func GetExtensionEntity(extensionID string) (*entities.Extension, error) {
	var item entities.Extension
	if err := db.DB.Where("id = ?", extensionID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("extension not found")
		}
		return nil, err
	}
	return &item, nil
}

// CreateExtension creates a new admin-added catalog extension.
func CreateExtension(req *models.CreateExtensionRequest, createdBy string) (*models.Extension, error) {
	var existing entities.Extension
	if err := db.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("extension with name %q already exists", req.Name)
	}
	item := &entities.Extension{
		Base:        entities.Base{ID: uuid.New()},
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		OCIUrl:      req.OCIUrl,
		IconURL:     req.IconURL,
		Builtin:     false,
		CreatedBy:   createdBy,
	}
	if err := db.DB.Create(item).Error; err != nil {
		return nil, fmt.Errorf("failed to create extension: %w", err)
	}
	m := toExtensionModel(item)
	return &m, nil
}

// DeleteExtension removes a catalog extension by ID (builtin extensions cannot be deleted).
func DeleteExtension(extensionID string) error {
	var item entities.Extension
	if err := db.DB.Where("id = ?", extensionID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("extension not found")
		}
		return err
	}
	if item.Builtin {
		return fmt.Errorf("built-in extensions cannot be deleted")
	}
	return db.DB.Delete(&item).Error
}

// UpdateExtension updates a non-builtin catalog extension's metadata.
func UpdateExtension(extensionID string, req *models.UpdateExtensionRequest) (*models.Extension, error) {
	var item entities.Extension
	if err := db.DB.Where("id = ?", extensionID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("extension not found")
		}
		return nil, err
	}
	if item.Builtin {
		return nil, fmt.Errorf("built-in extensions cannot be modified")
	}
	if req.DisplayName != "" {
		item.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		item.Description = req.Description
	}
	if req.OCIUrl != "" {
		item.OCIUrl = req.OCIUrl
	}
	item.IconURL = req.IconURL
	if err := db.DB.Save(&item).Error; err != nil {
		return nil, fmt.Errorf("failed to update extension: %w", err)
	}
	m := toExtensionModel(&item)
	return &m, nil
}

// GetInstalledClustersForExtension returns all clusters that have a given extension installed,
// queried directly from the cluster_extensions table (no Helm calls).
func GetInstalledClustersForExtension(extensionID string) ([]models.InstalledCluster, error) {
	// Verify the extension exists.
	if _, err := GetExtensionEntity(extensionID); err != nil {
		return nil, err
	}

	// Join cluster_extensions with clusters.
	type row struct {
		ClusterID   string
		ClusterName string
		ReleaseName string
		Namespace   string
		Version     string
		Status      string
	}
	var rows []row
	err := db.DB.Table("cluster_extensions ce").
		Select("ce.cluster_id, c.name as cluster_name, ce.release_name, ce.namespace, ce.version, ce.status").
		Joins("JOIN clusters c ON c.id = ce.cluster_id").
		Where("ce.extension_id = ? AND ce.deleted_at IS NULL AND c.deleted_at IS NULL", extensionID).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query installed clusters: %w", err)
	}

	result := make([]models.InstalledCluster, 0, len(rows))
	for _, r := range rows {
		result = append(result, models.InstalledCluster{
			ClusterID:   r.ClusterID,
			ClusterName: r.ClusterName,
			ReleaseName: r.ReleaseName,
			Namespace:   r.Namespace,
			Version:     r.Version,
			Status:      r.Status,
		})
	}
	return result, nil
}

// toExtensionModel converts the DB entity to the API model.
func toExtensionModel(e *entities.Extension) models.Extension {
	return models.Extension{
		ID:          e.ID,
		Name:        e.Name,
		DisplayName: e.DisplayName,
		Description: e.Description,
		OCIUrl:      e.OCIUrl,
		IconURL:     e.IconURL,
		Builtin:     e.Builtin,
		CreatedAt:   e.CreatedAt,
	}
}
