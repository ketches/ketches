package services

import (
	"encoding/json"
	"errors"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

func CreatePlugin(req *models.CreatePluginRequest) (*entities.Plugin, error) {
	var existing entities.Plugin
	if err := db.DB.Where("project_id = ? AND slug = ?", req.ProjectID, req.Slug).First(&existing).Error; err == nil {
		return nil, errors.New("plugin with this slug already exists in the project")
	}

	envVarsJSON, err := json.Marshal(req.EnvVars)
	if err != nil {
		return nil, err
	}

	plugin := &entities.Plugin{
		Base:             entities.Base{ID: uuid.New()},
		ProjectID:        req.ProjectID,
		Slug:             req.Slug,
		Name:             req.Name,
		Description:      req.Description,
		Image:            req.Image,
		RegistryUsername: req.RegistryUsername,
		RegistryPassword: req.RegistryPassword,
		Command:          req.Command,
		EnvVars:          string(envVarsJSON),
		PluginType:       req.PluginType,
	}

	if err := db.DB.Create(plugin).Error; err != nil {
		return nil, err
	}

	return plugin, nil
}

func GetPlugin(pluginID string) (*entities.Plugin, error) {
	var plugin entities.Plugin
	if err := db.DB.
		Select("plugins.*, (SELECT COUNT(*) FROM app_plugins WHERE app_plugins.plugin_id = plugins.id) as install_count").
		First(&plugin, "id = ?", pluginID).Error; err != nil {
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

func ListProjectPluginsSimple(projectID string) ([]entities.Plugin, error) {
  var plugins []entities.Plugin
  if err := db.DB.Select("id, slug, name, description, plugin_type, env_vars").Where("project_id = ?", projectID).Order("name").Find(&plugins).Error; err != nil {
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

func UpdatePlugin(pluginID string, req *models.UpdatePluginRequest) (*entities.Plugin, error) {
	var plugin entities.Plugin
	if err := db.DB.First(&plugin, "id = ?", pluginID).Error; err != nil {
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
	if req.RegistryUsername != "" {
		updates["registry_username"] = req.RegistryUsername
	}
	if req.RegistryPassword != "" {
		updates["registry_password"] = req.RegistryPassword
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

	if err := db.DB.Model(&plugin).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &plugin, nil
}

func DeletePlugin(pluginID string) error {
	var count int64
	if err := db.DB.Model(&entities.AppPlugin{}).Where("plugin_id = ?", pluginID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("cannot delete plugin: it is installed in one or more apps")
	}

	return db.DB.Delete(&entities.Plugin{}, "id = ?", pluginID).Error
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
		ID:       uuid.New(),
		AppID:    appID,
		PluginID: req.PluginID,
		Enabled:  true,
		EnvVars:  string(envVarsJSON),
	}

	if err := db.DB.Create(appPlugin).Error; err != nil {
		return nil, err
	}

	appPlugin.Plugin = plugin

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

func ListAppPlugins(appID string) ([]entities.AppPlugin, error) {
	var appPlugins []entities.AppPlugin
	if err := db.DB.Where("app_id = ?", appID).Preload("Plugin").Find(&appPlugins).Error; err != nil {
		return nil, err
	}
	return appPlugins, nil
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

func GetPluginInstalledApps(pluginID string) ([]entities.App, error) {
	var apps []entities.App
	err := db.DB.
		Joins("JOIN app_plugins ON app_plugins.app_id = apps.id").
		Where("app_plugins.plugin_id = ?", pluginID).
		Find(&apps).Error
	if err != nil {
		return nil, err
	}
	return apps, nil
}
