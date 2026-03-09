package services

import (
	"context"

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
			Apps: []models.SimpleApp{},
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
		Select("app_group_members.group_id, apps.id, apps.slug, apps.name").
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
		result[idx].Apps = append(result[idx].Apps, models.SimpleApp{
			ID:   row.ID,
			Slug: row.Slug,
			Name: row.Name,
		})
	}

	return result, nil
}

// ListSpecificGroupedApps returns paginated, full app details for a specific group using Joins and AppListRow DTO.
func ListSpecificGroupedApps(c context.Context, groupID string, page, pageSize int, search string) (int64, []models.AppResponse, error) {
	var rows []models.AppListRow
	var total int64

	// Count query with joins
	countQ := db.DB.Model(&entities.App{}).
		Joins("JOIN app_group_members ON app_group_members.app_id = apps.id").
		Where("app_group_members.group_id = ?", groupID)
	if search != "" {
		countQ = countQ.Where("apps.name LIKE ? OR apps.slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := countQ.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	// Data query with explicit JOINs to envs and clusters
	dataQ := db.DB.Table("apps").
		Select(`apps.id, apps.slug, apps.name, apps.description, apps.env_id,
			apps.app_type, apps.code_repository_id, apps.container_image,
			apps.container_command, apps.registry_username, apps.registry_password,
			apps.replicas, apps.request_cpu, apps.request_memory,
			apps.limit_cpu, apps.limit_memory, apps.deploy_status, apps.created_at,
			envs.name AS env_name, envs.slug AS env_slug,
			envs.cluster_id, envs.cluster_namespace, envs.is_build_env,
			clusters.name AS cluster_name`).
		Joins("JOIN app_group_members ON app_group_members.app_id = apps.id").
		Joins("JOIN envs ON envs.id = apps.env_id").
		Joins("JOIN clusters ON clusters.id = envs.cluster_id").
		Where("app_group_members.group_id = ?", groupID).
		Order("app_group_members.created_at ASC")
	if search != "" {
		dataQ = dataQ.Where("apps.name LIKE ? OR apps.slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := dataQ.Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return 0, nil, err
	}

	result := make([]models.AppResponse, 0, len(rows))
	for i := range rows {
		result = append(result, ToAppListResponse(c, &rows[i]))
	}
	return total, result, nil
}
