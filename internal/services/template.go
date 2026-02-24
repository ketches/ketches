package services

import (
	"fmt"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

// validTemplateStatuses defines the allowed status values for templates.
var validTemplateStatuses = map[string]bool{
	"draft":      true,
	"reviewing":  true,
	"published":  true,
	"deprecated": true,
}

func ListTemplates(projectID string) ([]entities.Template, error) {
	var templates []entities.Template
	if err := db.DB.Where("project_id = ?", projectID).
		Order("created_at desc").
		Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

func GetTemplate(id string) (*entities.Template, error) {
	var tmpl entities.Template
	if err := db.DB.Preload("Project").
		First(&tmpl, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &tmpl, nil
}

func CreateTemplate(projectID string, req *models.CreateTemplateRequest) (*entities.Template, error) {
	status := req.Status
	if status == "" {
		status = "draft"
	}
	if !validTemplateStatuses[status] {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	templateType := req.Type
	if templateType == "" {
		templateType = "application"
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	tmpl := &entities.Template{
		Base:        entities.Base{ID: uuid.New()},
		ProjectID:   projectID,
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Type:        templateType,
		Content:     req.Content,
		Status:      status,
		Enabled:     enabled,
	}
	if err := db.DB.Create(tmpl).Error; err != nil {
		return nil, err
	}
	return GetTemplate(tmpl.ID)
}

func UpdateTemplate(id string, req *models.UpdateTemplateRequest) (*entities.Template, error) {
	tmpl, err := GetTemplate(id)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		tmpl.Name = req.Name
	}
	if req.Slug != "" {
		tmpl.Slug = req.Slug
	}
	if req.Description != "" {
		tmpl.Description = req.Description
	}
	if req.Type != "" {
		tmpl.Type = req.Type
	}
	if req.Content != "" {
		tmpl.Content = req.Content
	}
	if req.Status != "" {
		if !validTemplateStatuses[req.Status] {
			return nil, fmt.Errorf("invalid status: %s", req.Status)
		}
		tmpl.Status = req.Status
	}
	if req.Enabled != nil {
		tmpl.Enabled = *req.Enabled
	}
	if err := db.DB.Save(tmpl).Error; err != nil {
		return nil, err
	}
	return GetTemplate(id)
}

func DeleteTemplate(id string) error {
	return db.DB.Delete(&entities.Template{}, "id = ?", id).Error
}

func ToTemplateResponse(t *entities.Template) models.TemplateResponse {
	return models.TemplateResponse{
		ID:          t.ID,
		ProjectID:   t.ProjectID,
		Name:        t.Name,
		Slug:        t.Slug,
		Description: t.Description,
		Type:        t.Type,
		Content:     t.Content,
		Status:      t.Status,
		Enabled:     t.Enabled,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}
