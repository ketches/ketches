package core

import (
	"context"
	"errors"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"gorm.io/gorm"
)

// LoadAppContext loads the complete desired state used to reconcile an
// application. Relationship data is fetched explicitly because entities do
// not declare GORM associations or physical foreign keys.
func LoadAppContext(ctx context.Context, appID string) (*models.AppContext, error) {
	if db.DB == nil {
		return nil, errors.New("database is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	database := db.DB.WithContext(ctx)
	var application entities.App
	if err := database.First(&application, "id = ?", appID).Error; err != nil {
		return nil, err
	}

	var env entities.Env
	if err := database.First(&env, "id = ?", application.EnvID).Error; err != nil {
		return nil, app.WrapErrorf(err, "load environment for app %s: %w", appID, err)
	}
	var project entities.Project
	if err := database.First(&project, "id = ?", env.ProjectID).Error; err != nil {
		return nil, app.WrapErrorf(err, "load project for app %s: %w", appID, err)
	}
	var cluster entities.Cluster
	if err := database.First(&cluster, "id = ?", env.ClusterID).Error; err != nil {
		return nil, app.WrapErrorf(err, "load cluster for app %s: %w", appID, err)
	}

	appCtx := &models.AppContext{
		App: application,
		EnvContext: models.EnvContext{
			Env:     env,
			Project: project,
			Cluster: cluster,
		},
		Plugins: make(map[string]entities.Plugin),
	}

	if err := database.Where("app_id = ?", appID).Order("id ASC").Find(&appCtx.EnvVars).Error; err != nil {
		return nil, app.WrapErrorf(err, "load environment variables for app %s: %w", appID, err)
	}
	if err := database.Where("app_id = ?", appID).Order("id ASC").Find(&appCtx.Volumes).Error; err != nil {
		return nil, app.WrapErrorf(err, "load volumes for app %s: %w", appID, err)
	}
	if err := database.Where("app_id = ?", appID).Order("id ASC").Find(&appCtx.ConfigFiles).Error; err != nil {
		return nil, app.WrapErrorf(err, "load config files for app %s: %w", appID, err)
	}
	if err := database.Where("app_id = ?", appID).Order("id ASC").Find(&appCtx.Probes).Error; err != nil {
		return nil, app.WrapErrorf(err, "load probes for app %s: %w", appID, err)
	}
	if err := database.Where("app_id = ?", appID).Order("port ASC, id ASC").Find(&appCtx.Gateways).Error; err != nil {
		return nil, app.WrapErrorf(err, "load gateways for app %s: %w", appID, err)
	}
	if err := loadAppGatewayContext(database, appID, appCtx); err != nil {
		return nil, err
	}

	var autoScaling entities.AppAutoScaling
	if err := database.Where("app_id = ?", appID).First(&autoScaling).Error; err == nil {
		appCtx.AutoScaling = &autoScaling
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, app.WrapErrorf(err, "load autoscaling for app %s: %w", appID, err)
	}

	var schedulingRule entities.AppSchedulingRule
	if err := database.Where("app_id = ?", appID).First(&schedulingRule).Error; err == nil {
		appCtx.SchedulingRule = &schedulingRule
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, app.WrapErrorf(err, "load scheduling rule for app %s: %w", appID, err)
	}

	if err := database.Where("app_id = ?", appID).Order("id ASC").Find(&appCtx.AppPlugins).Error; err != nil {
		return nil, app.WrapErrorf(err, "load plugins for app %s: %w", appID, err)
	}
	if err := loadAppPlugins(database, appID, appCtx); err != nil {
		return nil, err
	}

	return appCtx, nil
}

func loadAppGatewayContext(database *gorm.DB, appID string, appCtx *models.AppContext) error {
	gatewayIDs := make([]string, 0, len(appCtx.Gateways))
	for _, gateway := range appCtx.Gateways {
		gatewayIDs = append(gatewayIDs, gateway.ID)
	}
	if len(gatewayIDs) == 0 {
		return nil
	}

	if err := database.Where("app_gateway_id IN ?", gatewayIDs).
		Order("sort_order ASC, host ASC, path ASC, id ASC").
		Find(&appCtx.GatewayRoutes).Error; err != nil {
		return app.WrapErrorf(err, "load gateway routes for app %s: %w", appID, err)
	}

	routeIDs := make([]string, 0, len(appCtx.GatewayRoutes))
	for _, route := range appCtx.GatewayRoutes {
		routeIDs = append(routeIDs, route.ID)
	}
	if len(routeIDs) == 0 {
		return nil
	}
	if err := database.Where("route_id IN ?", routeIDs).Order("id ASC").Find(&appCtx.GatewayBackends).Error; err != nil {
		return app.WrapErrorf(err, "load gateway backends for app %s: %w", appID, err)
	}
	return nil
}

func loadAppPlugins(database *gorm.DB, appID string, appCtx *models.AppContext) error {
	pluginIDs := make([]string, 0, len(appCtx.AppPlugins))
	seenPluginIDs := make(map[string]struct{}, len(appCtx.AppPlugins))
	for _, appPlugin := range appCtx.AppPlugins {
		if appPlugin.PluginID == "" {
			continue
		}
		if _, exists := seenPluginIDs[appPlugin.PluginID]; exists {
			continue
		}
		seenPluginIDs[appPlugin.PluginID] = struct{}{}
		pluginIDs = append(pluginIDs, appPlugin.PluginID)
	}
	if len(pluginIDs) == 0 {
		return nil
	}

	var plugins []entities.Plugin
	if err := database.Where("id IN ?", pluginIDs).Order("id ASC").Find(&plugins).Error; err != nil {
		return app.WrapErrorf(err, "load plugin definitions for app %s: %w", appID, err)
	}
	for _, plugin := range plugins {
		appCtx.Plugins[plugin.ID] = plugin
	}
	return nil
}
