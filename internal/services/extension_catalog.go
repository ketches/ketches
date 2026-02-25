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
var builtinExtensions = []entities.ExtensionCatalogItem{
	{
		Name:        "gateway-helm",
		DisplayName: "Envoy Gateway",
		Description: "Envoy Gateway is an open source project for managing Envoy Proxy as a standalone or Kubernetes-based application gateway.",
		OCIUrl:      "oci://docker.io/envoyproxy/gateway-helm",
		Builtin:     true,
	},
}

// SeedBuiltinExtensionCatalog inserts built-in catalog items if they don't already exist.
func SeedBuiltinExtensionCatalog() error {
	for _, item := range builtinExtensions {
		var existing entities.ExtensionCatalogItem
		err := db.DB.Where("name = ?", item.Name).First(&existing).Error
		if err == nil {
			// Already seeded — skip.
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to query extension catalog item %q: %w", item.Name, err)
		}
		item.ID = uuid.New()
		if err := db.DB.Create(&item).Error; err != nil {
			return fmt.Errorf("failed to seed built-in extension %q: %w", item.Name, err)
		}
	}
	return nil
}

// ListExtensionCatalog returns all catalog items (builtin + admin-added).
func ListExtensionCatalog() ([]models.ExtensionCatalogItem, error) {
	var items []entities.ExtensionCatalogItem
	if err := db.DB.Order("builtin DESC, created_at ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list extension catalog: %w", err)
	}
	result := make([]models.ExtensionCatalogItem, 0, len(items))
	for _, item := range items {
		result = append(result, toExtensionCatalogItemModel(&item))
	}
	return result, nil
}

// GetExtensionCatalogItem returns a single catalog item by ID.
func GetExtensionCatalogItem(itemID string) (*models.ExtensionCatalogItem, error) {
	var item entities.ExtensionCatalogItem
	if err := db.DB.Where("id = ?", itemID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("extension catalog item not found")
		}
		return nil, err
	}
	m := toExtensionCatalogItemModel(&item)
	return &m, nil
}

// GetExtensionCatalogItemEntity returns the raw entity by ID (used by services).
func GetExtensionCatalogItemEntity(itemID string) (*entities.ExtensionCatalogItem, error) {
	var item entities.ExtensionCatalogItem
	if err := db.DB.Where("id = ?", itemID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("extension catalog item not found")
		}
		return nil, err
	}
	return &item, nil
}

// CreateExtensionCatalogItem creates a new admin-added catalog item.
func CreateExtensionCatalogItem(req *models.CreateExtensionCatalogItemRequest, createdBy string) (*models.ExtensionCatalogItem, error) {
	// Check for name uniqueness.
	var existing entities.ExtensionCatalogItem
	if err := db.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("extension catalog item with name %q already exists", req.Name)
	}

	item := &entities.ExtensionCatalogItem{
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
		return nil, fmt.Errorf("failed to create extension catalog item: %w", err)
	}
	m := toExtensionCatalogItemModel(item)
	return &m, nil
}

// DeleteExtensionCatalogItem removes a catalog item by ID (builtin items cannot be deleted).
func DeleteExtensionCatalogItem(itemID string) error {
	var item entities.ExtensionCatalogItem
	if err := db.DB.Where("id = ?", itemID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("extension catalog item not found")
		}
		return err
	}
	if item.Builtin {
		return fmt.Errorf("built-in extension catalog items cannot be deleted")
	}
	return db.DB.Delete(&item).Error
}

// toExtensionCatalogItemModel converts the DB entity to the API model.
func toExtensionCatalogItemModel(e *entities.ExtensionCatalogItem) models.ExtensionCatalogItem {
	return models.ExtensionCatalogItem{
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
