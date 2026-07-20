package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

type AppPluginWithPlugin struct {
	entities.AppPlugin
	Plugin entities.Plugin `gorm:"embedded;embeddedPrefix:plugin_"`
}

var ErrPluginInstalledInApps = errors.New("cannot delete plugin: it is installed in one or more apps")

func CreatePlugin(req *models.CreatePluginRequest) (*entities.Plugin, error) {
	var existing entities.Plugin
	if err := db.DB.Where("project_id = ? AND slug = ?", req.ProjectID, req.Slug).First(&existing).Error; err == nil {
		return nil, errors.New("plugin with this slug already exists in the project")
	}

	envVarsJSON, err := json.Marshal(req.EnvVars)
	if err != nil {
		return nil, err
	}
	registryPassword, err := secrets.EncryptString(req.RegistryPassword)
	if err != nil {
		return nil, err
	}

	plugin := &entities.Plugin{
		ID:               uuid.New(),
		ProjectID:        req.ProjectID,
		Slug:             req.Slug,
		Name:             req.Name,
		Description:      req.Description,
		Image:            req.Image,
		ImagePullPolicy:  req.ImagePullPolicy,
		RegistryUsername: req.RegistryUsername,
		RegistryPassword: registryPassword,
		Command:          req.Command,
		EnvVars:          string(envVarsJSON),
		PluginType:       req.PluginType,
	}

	if err := db.DB.Create(plugin).Error; err != nil {
		return nil, err
	}

	return plugin, nil
}

func GetPlugin(projectID, pluginID string) (*entities.Plugin, error) {
	var plugin entities.Plugin
	if err := db.DB.
		Select("plugins.*, (SELECT COUNT(*) FROM app_plugins WHERE app_plugins.plugin_id = plugins.id) as install_count").
		Where("plugins.project_id = ? AND plugins.id = ?", projectID, pluginID).
		First(&plugin).Error; err != nil {
		return nil, err
	}
	return &plugin, nil
}

func ListPlugins(page, pageSize int, search string) (int64, []entities.Plugin, error) {
	var plugins []entities.Plugin
	var total int64
	query := db.DB.Model(&entities.Plugin{})
	if search != "" {
		query = query.Where("name LIKE ? OR slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	query = db.DB.
		Select("plugins.*, (SELECT COUNT(*) FROM app_plugins WHERE app_plugins.plugin_id = plugins.id) as install_count").
		Order("created_at DESC")
	if search != "" {
		query = query.Where("name LIKE ? OR slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&plugins).Error; err != nil {
		return 0, nil, err
	}
	return total, plugins, nil
}

func ListPluginsSimple() ([]entities.Plugin, error) {
	var plugins []entities.Plugin
	if err := db.DB.Select("id, slug, name, description").Order("name").Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

func ListProjectPluginsSimple(projectID string) ([]models.SimplePlugin, error) {
	var plugins []models.SimplePlugin
	if err := db.DB.Model(&entities.Plugin{}).Select("id, slug, name, description, plugin_type").Where("project_id = ?", projectID).Order("name").Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

func ListProjectPlugins(projectID string, page, pageSize int, search string) (int64, []entities.Plugin, error) {
	var plugins []entities.Plugin
	var total int64
	query := db.DB.Model(&entities.Plugin{}).Where("project_id = ?", projectID)
	if search != "" {
		query = query.Where("name LIKE ? OR slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	query = db.DB.
		Select("plugins.*, (SELECT COUNT(*) FROM app_plugins WHERE app_plugins.plugin_id = plugins.id) as install_count").
		Where("project_id = ?", projectID).
		Order("created_at DESC")
	if search != "" {
		query = query.Where("name LIKE ? OR slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&plugins).Error; err != nil {
		return 0, nil, err
	}
	return total, plugins, nil
}

func UpdatePlugin(projectID, pluginID string, req *models.UpdatePluginRequest) (*entities.Plugin, error) {
	if _, err := GetPlugin(projectID, pluginID); err != nil {
		return nil, err
	}

	updates := make(map[string]any)

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Image != "" {
		updates["image"] = req.Image
	}
	if req.ImagePullPolicy != "" {
		updates["image_pull_policy"] = req.ImagePullPolicy
	}
	if req.RegistryUsername != nil {
		updates["registry_username"] = *req.RegistryUsername
	}
	if req.ClearRegistryPassword != nil && *req.ClearRegistryPassword {
		updates["registry_password"] = ""
	}
	if req.RegistryPassword != nil && *req.RegistryPassword != "" {
		registryPassword, err := secrets.EncryptString(*req.RegistryPassword)
		if err != nil {
			return nil, err
		}
		updates["registry_password"] = registryPassword
	}
	if req.Command != "" {
		updates["command"] = req.Command
	}
	if req.EnvVars != nil {
		envVarsJSON, err := json.Marshal(req.EnvVars)
		if err != nil {
			return nil, err
		}
		updates["env_vars"] = string(envVarsJSON)
	}
	if req.PluginType != "" {
		updates["plugin_type"] = req.PluginType
	}

	if err := db.DB.Model(&entities.Plugin{}).
		Where("project_id = ? AND id = ?", projectID, pluginID).
		Updates(updates).Error; err != nil {
		return nil, err
	}

	return GetPlugin(projectID, pluginID)
}

func DeletePlugin(projectID, pluginID string) error {
	if _, err := GetPlugin(projectID, pluginID); err != nil {
		return err
	}

	var count int64
	if err := db.DB.Model(&entities.AppPlugin{}).
		Joins("JOIN plugins ON plugins.id = app_plugins.plugin_id").
		Where("plugins.project_id = ? AND plugins.id = ?", projectID, pluginID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrPluginInstalledInApps
	}

	return db.DB.Where("project_id = ? AND id = ?", projectID, pluginID).Delete(&entities.Plugin{}).Error
}

func InstallPluginToApp(appID string, req *models.InstallPluginRequest) (*entities.AppPlugin, error) {
	var app entities.App
	if err := db.DB.First(&app, "id = ?", appID).Error; err != nil {
		return nil, err
	}

	var plugin entities.Plugin
	if err := db.DB.First(&plugin, "id = ?", req.PluginID).Error; err != nil {
		return nil, err
	}

	var existing entities.AppPlugin
	if err := db.DB.Where("app_id = ? AND plugin_id = ?", appID, req.PluginID).First(&existing).Error; err == nil {
		return nil, errors.New("plugin is already installed in this app")
	}

	envVars := req.EnvVars
	if len(envVars) == 0 {
		var pluginEnvVars []models.PluginEnvVar
		if err := json.Unmarshal([]byte(plugin.EnvVars), &pluginEnvVars); err == nil {
			envVars = pluginEnvVars
		}
	}

	envVarsJSON, _ := json.Marshal(envVars)

	appPlugin := &entities.AppPlugin{
		ID:            uuid.New(),
		AppID:         appID,
		PluginID:      req.PluginID,
		Enabled:       true,
		EnvVars:       string(envVarsJSON),
		RequestCPU:    entities.DefaultAppPluginRequestCPU,
		RequestMemory: entities.DefaultAppPluginRequestMemory,
		LimitCPU:      entities.DefaultAppPluginLimitCPU,
		LimitMemory:   entities.DefaultAppPluginLimitMemory,
	}

	if err := db.DB.Create(appPlugin).Error; err != nil {
		return nil, err
	}
	_ = plugin

	return appPlugin, nil
}

func UninstallPluginFromApp(appID, pluginID string) error {
	result := db.DB.Where("app_id = ? AND plugin_id = ?", appID, pluginID).Delete(&entities.AppPlugin{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func ListAppPlugins(appID string) ([]AppPluginWithPlugin, error) {
	var rows []AppPluginWithPlugin
	if err := db.DB.Table("app_plugins").
		Select("app_plugins.*, plugins.id AS plugin_id, plugins.created_at AS plugin_created_at, plugins.updated_at AS plugin_updated_at, plugins.project_id AS plugin_project_id, plugins.slug AS plugin_slug, plugins.name AS plugin_name, plugins.description AS plugin_description, plugins.image AS plugin_image, plugins.registry_username AS plugin_registry_username, plugins.registry_password AS plugin_registry_password, plugins.command AS plugin_command, plugins.env_vars AS plugin_env_vars, plugins.plugin_type AS plugin_plugin_type").
		Joins("JOIN plugins ON plugins.id = app_plugins.plugin_id").
		Where("app_plugins.app_id = ?", appID).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func ToggleAppPlugin(appID, pluginID string, enabled bool) error {
	result := db.DB.Model(&entities.AppPlugin{}).
		Where("app_id = ? AND plugin_id = ?", appID, pluginID).
		Update("enabled", enabled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func UpdateAppPluginEnvVars(appID, pluginID string, envVars []models.PluginEnvVar) error {
	envVarsJSON, err := json.Marshal(envVars)
	if err != nil {
		return err
	}

	result := db.DB.Model(&entities.AppPlugin{}).
		Where("app_id = ? AND plugin_id = ?", appID, pluginID).
		Update("env_vars", string(envVarsJSON))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func UpdateAppPluginResources(ctx context.Context, appID, pluginID string, req *models.UpdateAppPluginResourcesRequest) error {
	result := db.DB.Model(&entities.AppPlugin{}).
		Where("app_id = ? AND plugin_id = ?", appID, pluginID).
		Updates(map[string]any{
			"request_cpu":    req.RequestCPU,
			"request_memory": req.RequestMemory,
			"limit_cpu":      req.LimitCPU,
			"limit_memory":   req.LimitMemory,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return err
	}

	return applyAppFn(ctx, appCtx)
}

func GetPluginInstalledApps(projectID, pluginID string) ([]entities.App, error) {
	if _, err := GetPlugin(projectID, pluginID); err != nil {
		return nil, err
	}

	var apps []entities.App
	err := db.DB.
		Joins("JOIN app_plugins ON app_plugins.app_id = apps.id").
		Joins("JOIN plugins ON plugins.id = app_plugins.plugin_id").
		Where("plugins.project_id = ? AND plugins.id = ?", projectID, pluginID).
		Find(&apps).Error
	if err != nil {
		return nil, err
	}
	return apps, nil
}
