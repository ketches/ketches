package services

import (
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

// ListAppGroups returns all groups for an environment.
func ListAppGroups(envID string) ([]entities.AppGroup, error) {
	var groups []entities.AppGroup
	if err := db.DB.Where("env_id = ?", envID).Order("created_at ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

// CreateAppGroup creates a new app group for an environment.
func CreateAppGroup(envID string, req *models.CreateAppGroupRequest) (*entities.AppGroup, error) {
	group := &entities.AppGroup{
		ID:          uuid.New(),
		EnvID:       envID,
		Name:        req.Name,
		Description: req.Description,
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

// GetAppGroup returns an app group by ID.
func GetAppGroup(groupID string) (*entities.AppGroup, error) {
	var group entities.AppGroup
	if err := db.DB.First(&group, "id = ?", groupID).Error; err != nil {
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
		ID:      uuid.New(),
		GroupID: groupID,
		AppID:   appID,
	}
	return db.DB.Create(member).Error
}

// RemoveAppFromGroup removes an app from a group.
func RemoveAppFromGroup(groupID, appID string) error {
	return db.DB.Where("group_id = ? AND app_id = ?", groupID, appID).Delete(&entities.AppGroupMember{}).Error
}

// ListGroupedApps returns each group with its apps for the given environment.
func ListGroupedApps(envID string) ([]models.AppGroupWithApps, error) {
	groups, err := ListAppGroups(envID)
	if err != nil {
		return nil, err
	}

	result := make([]models.AppGroupWithApps, 0, len(groups))
	groupIndexByID := make(map[string]int, len(groups))
	groupIDs := make([]string, 0, len(groups))
	for _, g := range groups {
		result = append(result, models.AppGroupWithApps{
			AppGroupResponse: models.AppGroupResponse{
				ID:          g.ID,
				EnvID:       g.EnvID,
				Name:        g.Name,
				Description: g.Description,
				CreatedAt:   g.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
			Apps: []models.AppSimpleResponse{},
		})
		groupIndexByID[g.ID] = len(result) - 1
		groupIDs = append(groupIDs, g.ID)
	}

	if len(groupIDs) == 0 {
		return result, nil
	}

	type groupAppRow struct {
		GroupID      string
		ID           string
		Slug         string
		Name         string
		DeployStatus string
	}

	var rows []groupAppRow
	if err := db.DB.Model(&entities.AppGroupMember{}).
		Select("app_group_members.group_id, apps.id, apps.slug, apps.name, apps.deploy_status").
		Joins("JOIN apps ON apps.id = app_group_members.app_id").
		Where("app_group_members.group_id IN ? AND apps.env_id = ?", groupIDs, envID).
		Order("app_group_members.created_at ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		idx, ok := groupIndexByID[row.GroupID]
		if !ok {
			continue
		}
		result[idx].Apps = append(result[idx].Apps, models.AppSimpleResponse{
			ID:     row.ID,
			Slug:   row.Slug,
			Name:   row.Name,
			Status: row.DeployStatus,
		})
	}

	return result, nil
}

// ListSpecificGroupedApps returns one group with its apps.
func ListSpecificGroupedApps(groupID string) (models.AppGroupWithApps, error) {
	group, err := GetAppGroup(groupID)
	if err != nil {
		return models.AppGroupWithApps{}, err
	}

	type appRow struct {
		ID           string
		Slug         string
		Name         string
		DeployStatus string
	}

	var rows []appRow
	if err := db.DB.Model(&entities.AppGroupMember{}).
		Select("apps.id, apps.slug, apps.name, apps.deploy_status").
		Joins("JOIN apps ON apps.id = app_group_members.app_id").
		Where("app_group_members.group_id = ?", group.ID).
		Order("app_group_members.created_at ASC").
		Scan(&rows).Error; err != nil {
		return models.AppGroupWithApps{}, err
	}

	apps := make([]models.AppSimpleResponse, 0, len(rows))
	for _, row := range rows {
		apps = append(apps, models.AppSimpleResponse{
			ID:     row.ID,
			Slug:   row.Slug,
			Name:   row.Name,
			Status: row.DeployStatus,
		})
	}

	result := models.AppGroupWithApps{
		AppGroupResponse: models.AppGroupResponse{
			ID:          group.ID,
			EnvID:       group.EnvID,
			Name:        group.Name,
			Description: group.Description,
			CreatedAt:   group.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		Apps: apps,
	}

	return result, nil
}
