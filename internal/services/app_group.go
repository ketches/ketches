package services

import (
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

// ListAppGroups returns all groups for a project.
func ListAppGroups(projectID string) ([]entities.AppGroup, error) {
	var groups []entities.AppGroup
	if err := db.DB.Where("project_id = ?", projectID).Order("created_at ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

// CreateAppGroup creates a new app group for a project.
func CreateAppGroup(projectID, userID string, req *models.CreateAppGroupRequest) (*entities.AppGroup, error) {
	group := &entities.AppGroup{
		Base:            entities.Base{ID: uuid.New()},
		ProjectID:       projectID,
		Name:            req.Name,
		Description:     req.Description,
		CreatedByUserID: userID,
	}
	if err := db.DB.Create(group).Error; err != nil {
		return nil, err
	}
	return group, nil
}

// UpdateAppGroup updates an app group's name and description.
func UpdateAppGroup(groupID string, req *models.UpdateAppGroupRequest) (*entities.AppGroup, error) {
	var group entities.AppGroup
	if err := db.DB.First(&group, "id = ?", groupID).Error; err != nil {
		return nil, err
	}
	group.Name = req.Name
	group.Description = req.Description
	if err := db.DB.Save(&group).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// DeleteAppGroup removes all members then deletes the group.
func DeleteAppGroup(groupID string) error {
	if err := db.DB.Where("group_id = ?", groupID).Delete(&entities.AppGroupMember{}).Error; err != nil {
		return err
	}
	return db.DB.Delete(&entities.AppGroup{}, "id = ?", groupID).Error
}

// AddAppToGroup adds an app to a group (idempotent — ignores duplicate).
func AddAppToGroup(groupID, appID string) error {
	var count int64
	db.DB.Model(&entities.AppGroupMember{}).Where("group_id = ? AND app_id = ?", groupID, appID).Count(&count)
	if count > 0 {
		return nil
	}
	member := &entities.AppGroupMember{
		Base:    entities.Base{ID: uuid.New()},
		GroupID: groupID,
		AppID:   appID,
	}
	return db.DB.Create(member).Error
}

// RemoveAppFromGroup removes an app from a group.
func RemoveAppFromGroup(groupID, appID string) error {
	return db.DB.Where("group_id = ? AND app_id = ?", groupID, appID).Delete(&entities.AppGroupMember{}).Error
}

// ListGroupedApps returns each group with its apps scoped to the given env.
func ListGroupedApps(projectID, envID string) ([]models.AppGroupWithApps, error) {
	groups, err := ListAppGroups(projectID)
	if err != nil {
		return nil, err
	}

	var result []models.AppGroupWithApps
	for _, g := range groups {
		var members []entities.AppGroupMember
		if err := db.DB.Where("group_id = ?", g.ID).Find(&members).Error; err != nil {
			return nil, err
		}

		var apps []models.AppSimpleResponse
		for _, m := range members {
			var app entities.App
			if err := db.DB.Select("id, slug, name, deploy_status").
				Where("id = ? AND env_id = ?", m.AppID, envID).
				First(&app).Error; err != nil {
				continue // skip apps not in this env
			}
			apps = append(apps, models.AppSimpleResponse{
				ID:     app.ID,
				Slug:   app.Slug,
				Name:   app.Name,
				Status: app.DeployStatus,
			})
		}
		if apps == nil {
			apps = []models.AppSimpleResponse{}
		}

		result = append(result, models.AppGroupWithApps{
			AppGroupResponse: models.AppGroupResponse{
				ID:          g.ID,
				ProjectID:   g.ProjectID,
				Name:        g.Name,
				Description: g.Description,
				CreatedAt:   g.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
			Apps: apps,
		})
	}
	if result == nil {
		result = []models.AppGroupWithApps{}
	}
	return result, nil
}
