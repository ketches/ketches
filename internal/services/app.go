// Copyright 2025 The Ketches Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/containerregistry"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

// ListApps returns a paginated list of apps for the given environment using
// explicit JOINs on envs and clusters instead of GORM Preload. Results are
// scanned into the flat AppListRow DTO.
func ListApps(envID string, page, pageSize int, search string) (int64, []models.AppListRow, error) {
	var rows []models.AppListRow
	var total int64

	// Count query (simple, no joins needed)
	countQ := db.DB.Model(&entities.App{}).Where("apps.env_id = ?", envID)
	if search != "" {
		countQ = countQ.Where("apps.name LIKE ? OR apps.slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := countQ.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	// Data query with explicit JOINs
	dataQ := db.DB.Table("apps").
		Select(`apps.id, apps.slug, apps.name, apps.description, apps.env_id,
			apps.app_type, apps.code_repository_id, apps.container_image,
			apps.container_command, apps.registry_username, apps.registry_password,
			apps.replicas, apps.request_cpu, apps.request_memory,
			apps.limit_cpu, apps.limit_memory, apps.deploy_status, apps.created_at,
			envs.name AS env_name, envs.slug AS env_slug,
			envs.cluster_id, envs.cluster_namespace, envs.is_build_env,
			clusters.name AS cluster_name`).
		Joins("JOIN envs ON envs.id = apps.env_id").
		Joins("JOIN clusters ON clusters.id = envs.cluster_id").
		Where("apps.env_id = ?", envID).
		Where("apps.deleted_at IS NULL").
		Order("apps.created_at DESC")
	if search != "" {
		dataQ = dataQ.Where("apps.name LIKE ? OR apps.slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := dataQ.Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return 0, nil, err
	}

	return total, rows, nil
}

func ListAppsSimple(c context.Context, envID string) ([]models.SimpleApp, error) {
	var apps []models.SimpleApp
	if err := db.DB.Model(&entities.App{}).Select("id, slug, name, description, code_repository_id").Where("env_id = ?", envID).Order("created_at DESC").Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

func CreateApp(ctx context.Context, envID string, req *models.CreateAppRequest) (*models.AppContext, error) {
	var existing entities.App
	if err := db.DB.Where("env_id = ? AND slug = ?", envID, req.Slug).First(&existing).Error; err == nil {
		return nil, gorm.ErrDuplicatedKey
	}

	replicas := req.Replicas
	if replicas == 0 {
		replicas = 1
	}

	application := &entities.App{
		Base:             entities.Base{ID: uuid.New()},
		Slug:             req.Slug,
		Name:             req.Name,
		Description:      req.Description,
		EnvID:            envID,
		ContainerImage:   req.ContainerImage,
		ContainerCommand: req.ContainerCommand,
		RegistryUsername: req.RegistryUsername,
		RegistryPassword: req.RegistryPassword,
		Replicas:         replicas,
		RequestCPU:       req.RequestCPU,
		RequestMemory:    req.RequestMemory,
		LimitCPU:         req.LimitCPU,
		LimitMemory:      req.LimitMemory,
		AppType:          req.AppType,
		DeployStatus:     "undeployed",
	}

	if err := db.DB.Create(application).Error; err != nil {
		return nil, err
	}

	// Create AutoScaling record if requested
	var autoScaling *entities.AppAutoScaling
	if req.AutoScaling != nil {
		autoScaling = &entities.AppAutoScaling{
			ID:                      uuid.New(),
			AppID:                   application.ID,
			MinReplicas:             req.AutoScaling.MinReplicas,
			MaxReplicas:             req.AutoScaling.MaxReplicas,
			TargetCPUUtilization:    req.AutoScaling.TargetCPUUtilization,
			TargetMemoryUtilization: req.AutoScaling.TargetMemoryUtilization,
		}
		if err := db.DB.Create(autoScaling).Error; err != nil {
			return nil, err
		}
	}

	// Create SchedulingRule record if requested
	var schedulingRule *entities.AppSchedulingRule
	if req.SchedulingRule != nil {
		schedulingRule = &entities.AppSchedulingRule{
			ID:           uuid.New(),
			AppID:        application.ID,
			RuleType:     req.SchedulingRule.RuleType,
			NodeName:     req.SchedulingRule.NodeName,
			NodeSelector: req.SchedulingRule.NodeSelector,
			NodeAffinity: req.SchedulingRule.NodeAffinity,
			Tolerations:  req.SchedulingRule.Tolerations,
		}
		if err := db.DB.Create(schedulingRule).Error; err != nil {
			return nil, err
		}
	}

	// Attempt to seed app configuration from image metadata; failure is non-fatal.
	if req.SeedImageMetadata {
		if err := seedAppFromImageMetadata(ctx, application); err != nil {
			log.Printf("warn: image metadata seed skipped for app %s: %v", application.Slug, err)
		}
	}

	envCtx, err := GetEnvContext(envID)
	if err != nil {
		return nil, err
	}

	// Build AppContext for core layer
	appCtx := &models.AppContext{
		App:            *application,
		EnvContext:     *envCtx,
		AutoScaling:    autoScaling,
		SchedulingRule: schedulingRule,
	}

	if req.Deploy {
		if err := core.ApplyApp(ctx, appCtx); err != nil {
			return nil, err
		}
		appCtx.App.DeployStatus = "deployed"
		db.DB.Model(application).Update("deploy_status", "deployed")
	}

	return appCtx, nil
}

func CreateAppFromCodeRepositoryBuild(ctx context.Context, envID, slug, name, containerImage, registryUsername, registryPassword string, codeRepositoryID string) (*models.AppContext, error) {
	var existing entities.App
	if err := db.DB.Where("env_id = ? AND slug = ?", envID, slug).First(&existing).Error; err == nil {
		return nil, gorm.ErrDuplicatedKey
	}

	repoID := codeRepositoryID
	application := &entities.App{
		Base:             entities.Base{ID: uuid.New()},
		Slug:             slug,
		Name:             name,
		EnvID:            envID,
		ContainerImage:   containerImage,
		RegistryUsername: registryUsername,
		RegistryPassword: registryPassword,
		Replicas:         1,
		RequestCPU:       100,
		RequestMemory:    128,
		LimitCPU:         1000,
		LimitMemory:      512,
		AppType:          app.AppTypeDeployment,
		DeployStatus:     app.AppStatusUndeployed,
		CodeRepositoryID: &repoID,
	}
	if err := db.DB.Create(application).Error; err != nil {
		return nil, err
	}
	envCtx, err := GetEnvContext(envID)
	if err != nil {
		return nil, err
	}

	return &models.AppContext{
		App:        *application,
		EnvContext: *envCtx,
	}, nil
}

func GetAppSimple(ctx context.Context, appID string) (*models.SimpleApp, error) {
	var app models.SimpleApp
	if err := db.DB.Model(&entities.App{}).Select("id, slug, name, description").First(&app, "id = ?", appID).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

func GetAppContext(ctx context.Context, appID string) (*models.AppContext, error) {
	var application entities.App
	if err := db.DB.First(&application, "id = ?", appID).Error; err != nil {
		return nil, err
	}

	envCtx, err := GetEnvContext(application.EnvID)
	if err != nil {
		return nil, err
	}

	// Batch-fetch 1:N relations
	var envVars []entities.AppEnvVar
	db.DB.Where("app_id = ?", appID).Find(&envVars)
	var volumes []entities.AppVolume
	db.DB.Where("app_id = ?", appID).Find(&volumes)
	var configFiles []entities.AppConfigFile
	db.DB.Where("app_id = ?", appID).Find(&configFiles)
	var probes []entities.AppProbe
	db.DB.Where("app_id = ?", appID).Find(&probes)
	var gateways []entities.AppGateway
	db.DB.Where("app_id = ?", appID).Find(&gateways)

	// Fetch 1:1 optional relations
	var autoScaling *entities.AppAutoScaling
	var as entities.AppAutoScaling
	if err := db.DB.Where("app_id = ?", appID).First(&as).Error; err == nil {
		autoScaling = &as
	}
	var schedulingRule *entities.AppSchedulingRule
	var sr entities.AppSchedulingRule
	if err := db.DB.Where("app_id = ?", appID).First(&sr).Error; err == nil {
		schedulingRule = &sr
	}

	// Fetch AppPlugins with their Plugin (N:1)
	var appPlugins []entities.AppPlugin
	db.DB.Where("app_id = ?", appID).Find(&appPlugins)
	plugins := make(map[string]entities.Plugin)
	pluginIDs := make(map[string]struct{})
	for i := range appPlugins {
		if appPlugins[i].PluginID != "" {
			pluginIDs[appPlugins[i].PluginID] = struct{}{}
		}
	}
	if len(pluginIDs) > 0 {
		ids := make([]string, 0, len(pluginIDs))
		for id := range pluginIDs {
			ids = append(ids, id)
		}
		var pluginRows []entities.Plugin
		db.DB.Where("id IN ?", ids).Find(&pluginRows)
		for _, p := range pluginRows {
			plugins[p.ID] = p
		}
	}

	return &models.AppContext{
		App:            application,
		EnvContext:     *envCtx,
		EnvVars:        envVars,
		Volumes:        volumes,
		Gateways:       gateways,
		Probes:         probes,
		ConfigFiles:    configFiles,
		SchedulingRule: schedulingRule,
		AutoScaling:    autoScaling,
		AppPlugins:     appPlugins,
		Plugins:        plugins,
	}, nil
}

func ApplyApp(ctx context.Context, appCtx *models.AppContext) error {
	return core.ApplyApp(ctx, appCtx)
}

func UpdateAppBasic(ctx context.Context, appID string, req *models.UpdateBasicInfoRequest) (*models.AppContext, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	appCtx.App.Name = req.Name
	appCtx.App.Description = req.Description

	if err := db.DB.Save(&appCtx.App).Error; err != nil {
		return nil, err
	}

	return appCtx, nil
}

func UpdateAppImage(ctx context.Context, appID string, req *models.UpdateAppImageRequest) (*models.AppContext, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	appCtx.App.ContainerImage = req.ContainerImage
	appCtx.App.RegistryUsername = req.RegistryUsername
	if req.RegistryPassword != "" {
		appCtx.App.RegistryPassword = req.RegistryPassword
	}

	if err := db.DB.Save(&appCtx.App).Error; err != nil {
		return nil, err
	}

	if err := ApplyApp(ctx, appCtx); err != nil {
		return nil, err
	}

	return appCtx, nil
}

func UpdateAppReplicas(ctx context.Context, appID string, req *models.UpdateAppReplicasRequest) (*models.AppContext, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	appCtx.App.Replicas = req.Replicas

	if err := db.DB.Save(&appCtx.App).Error; err != nil {
		return nil, err
	}

	if err := ApplyApp(ctx, appCtx); err != nil {
		return nil, err
	}

	return appCtx, nil
}

func UpdateAppResources(ctx context.Context, appID string, req *models.UpdateAppResourcesRequest) (*models.AppContext, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	appCtx.App.RequestCPU = req.RequestCPU
	appCtx.App.RequestMemory = req.RequestMemory
	appCtx.App.LimitCPU = req.LimitCPU
	appCtx.App.LimitMemory = req.LimitMemory

	if err := db.DB.Save(&appCtx.App).Error; err != nil {
		return nil, err
	}

	if err := ApplyApp(ctx, appCtx); err != nil {
		return nil, err
	}

	return appCtx, nil
}

func UpdateAppAutoScaling(ctx context.Context, appID string, req *models.UpdateAppAutoScalingRequest) (*models.AppContext, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	if req.AutoScaling != nil {
		if appCtx.AutoScaling == nil {
			appCtx.AutoScaling = &entities.AppAutoScaling{ID: uuid.New(), AppID: appCtx.App.ID}
		}
		appCtx.AutoScaling.MinReplicas = req.AutoScaling.MinReplicas
		appCtx.AutoScaling.MaxReplicas = req.AutoScaling.MaxReplicas
		appCtx.AutoScaling.TargetCPUUtilization = req.AutoScaling.TargetCPUUtilization
		appCtx.AutoScaling.TargetMemoryUtilization = req.AutoScaling.TargetMemoryUtilization
		if err := db.DB.Save(appCtx.AutoScaling).Error; err != nil {
			return nil, err
		}
	} else {
		if appCtx.AutoScaling != nil {
			db.DB.Delete(appCtx.AutoScaling)
			appCtx.AutoScaling = nil
		}
	}

	if err := ApplyApp(ctx, appCtx); err != nil {
		return nil, err
	}

	return appCtx, nil
}

func UpdateAppHealth(ctx context.Context, appID string, req *models.UpdateAppHealthRequest) (*models.AppContext, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		tx.Delete(&entities.AppProbe{}, "app_id = ?", appCtx.App.ID)
		for _, p := range req.Probes {
			probe := &entities.AppProbe{
				ID:                  uuid.New(),
				AppID:               appCtx.App.ID,
				Type:                p.Type,
				ProbeMode:           p.ProbeMode,
				Enabled:             p.Enabled,
				HttpGetPath:         p.HttpGetPath,
				HttpGetPort:         p.HttpGetPort,
				TcpSocketPort:       p.TcpSocketPort,
				ExecCommand:         p.ExecCommand,
				InitialDelaySeconds: p.InitialDelaySeconds,
				PeriodSeconds:       p.PeriodSeconds,
				TimeoutSeconds:      p.TimeoutSeconds,
				SuccessThreshold:    p.SuccessThreshold,
				FailureThreshold:    p.FailureThreshold,
			}
			if err := tx.Create(probe).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Re-fetch to get updated probes
	return GetAppContext(ctx, appCtx.App.ID)
}

func UpdateAppScheduling(ctx context.Context, appID string, req *models.UpdateAppSchedulingRequest) (*models.AppContext, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	if req.SchedulingRule != nil {
		if appCtx.SchedulingRule == nil {
			appCtx.SchedulingRule = &entities.AppSchedulingRule{ID: uuid.New(), AppID: appCtx.App.ID}
		}
		appCtx.SchedulingRule.RuleType = req.SchedulingRule.RuleType
		appCtx.SchedulingRule.NodeName = req.SchedulingRule.NodeName
		appCtx.SchedulingRule.NodeSelector = req.SchedulingRule.NodeSelector
		appCtx.SchedulingRule.NodeAffinity = req.SchedulingRule.NodeAffinity
		appCtx.SchedulingRule.Tolerations = req.SchedulingRule.Tolerations
		if err := db.DB.Save(appCtx.SchedulingRule).Error; err != nil {
			return nil, err
		}
	} else {
		if appCtx.SchedulingRule != nil {
			db.DB.Delete(appCtx.SchedulingRule)
			appCtx.SchedulingRule = nil
		}
	}

	if err := ApplyApp(ctx, appCtx); err != nil {
		return nil, err
	}

	return appCtx, nil
}

func UpdateAppCommand(ctx context.Context, appID string, req *models.UpdateAppCommandRequest) (*models.AppContext, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	appCtx.App.ContainerCommand = req.ContainerCommand

	if err := db.DB.Save(&appCtx.App).Error; err != nil {
		return nil, err
	}

	if err := ApplyApp(ctx, appCtx); err != nil {
		return nil, err
	}

	return appCtx, nil
}

func DeleteApp(ctx context.Context, appID string) error {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return err
	}

	if _, err := executeStopAction(ctx, appCtx); err != nil {
		return err
	}

	return db.DB.Delete(&entities.App{}, "id = ?", appID).Error
}

// BatchDeleteApps deletes multiple applications by their IDs
func BatchDeleteApps(ctx context.Context, ids []string) error {
	var errs []error
	for _, id := range ids {
		if err := DeleteApp(ctx, id); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete app %s: %w", id, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to delete %d app(s): %v", len(errs), errs[0])
	}
	return nil
}

func PermanentlyDeleteApp(ctx context.Context, appID string) error {
	var application entities.App
	if err := db.DB.Unscoped().First(&application, "id = ?", appID).Error; err != nil {
		return err
	}

	// Build a minimal AppContext for K8s resource cleanup
	var env entities.Env
	if err := db.DB.Unscoped().First(&env, "id = ?", application.EnvID).Error; err != nil {
		return err
	}
	envCtx, err := GetEnvContext(env.ID)
	if err != nil {
		return err
	}

	var autoScaling *entities.AppAutoScaling
	var as entities.AppAutoScaling
	if err := db.DB.Where("app_id = ?", appID).First(&as).Error; err == nil {
		autoScaling = &as
	}

	appCtx := &models.AppContext{
		App:         application,
		EnvContext:  *envCtx,
		AutoScaling: autoScaling,
	}

	// Delete Kubernetes resources created by this app
	if err := deleteAppK8sResources(ctx, appCtx, false); err != nil {
		return err
	}

	// Hard-delete all child records directly (no need to fetch first)
	if err := db.DB.Unscoped().Where("app_id = ?", appID).Delete(&entities.AppEnvVar{}).Error; err != nil {
		return err
	}
	if err := db.DB.Unscoped().Where("app_id = ?", appID).Delete(&entities.AppVolume{}).Error; err != nil {
		return err
	}
	if err := db.DB.Unscoped().Where("app_id = ?", appID).Delete(&entities.AppConfigFile{}).Error; err != nil {
		return err
	}
	if err := db.DB.Unscoped().Where("app_id = ?", appID).Delete(&entities.AppGateway{}).Error; err != nil {
		return err
	}
	if err := db.DB.Unscoped().Where("app_id = ?", appID).Delete(&entities.AppProbe{}).Error; err != nil {
		return err
	}
	if err := db.DB.Unscoped().Where("app_id = ?", appID).Delete(&entities.AppPlugin{}).Error; err != nil {
		return err
	}
	if err := db.DB.Unscoped().Where("app_id = ?", appID).Delete(&entities.AppAutoScaling{}).Error; err != nil {
		return err
	}
	if err := db.DB.Unscoped().Where("app_id = ?", appID).Delete(&entities.AppSchedulingRule{}).Error; err != nil {
		return err
	}

	return db.DB.Unscoped().Delete(&entities.App{}, "id = ?", appID).Error
}

// deleteAppK8sResources deletes all Kubernetes resources created by an app
func deleteAppK8sResources(ctx context.Context, appCtx *models.AppContext, keepStorageData bool) error {
	if appCtx.EnvContext.Env.ClusterID == "" {
		return nil
	}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	ns := appCtx.EnvContext.Env.ClusterNamespace
	appLabel := "ketches.cn/app-slug=" + appCtx.App.Slug

	// Delete Deployment or StatefulSet
	switch appCtx.App.AppType {
	case app.AppTypeDeployment:
		if err := client.AppsV1().Deployments(ns).Delete(ctx, appCtx.App.Slug, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	case app.AppTypeStatefulSet:
		if err := client.AppsV1().StatefulSets(ns).Delete(ctx, appCtx.App.Slug, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	// Delete Service
	if err := client.CoreV1().Services(ns).Delete(ctx, appCtx.App.Slug, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	// Delete ConfigMap if exists
	configMapName := appCtx.App.Slug + "-config"
	if err := client.CoreV1().ConfigMaps(ns).Delete(ctx, configMapName, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	// Delete registry Secret if exists
	if appCtx.App.RegistryUsername != "" {
		secretName := appCtx.App.Slug + "-registry"
		if err := client.CoreV1().Secrets(ns).Delete(ctx, secretName, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	// Delete PVCs
	if !keepStorageData {
		if err := client.CoreV1().PersistentVolumeClaims(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: appLabel,
		}); err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	// Delete HPA if exists
	if appCtx.AutoScaling != nil {
		hpaName := appCtx.App.Slug
		if err := client.AutoscalingV2().HorizontalPodAutoscalers(ns).Delete(ctx, hpaName, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	// Delete Gateway API resources
	if gwClient, err := kube.GlobalClusterStore.GetGatewayClient(appCtx.EnvContext.Env.ClusterID); err == nil {
		if err := gwClient.GatewayV1().HTTPRoutes(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: appLabel,
		}); err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func RestoreApp(ctx context.Context, appID string) error {
	return db.DB.Unscoped().Model(&entities.App{}).Where("id = ?", appID).Update("deleted_at", nil).Error
}

func ListAppInstances(ctx context.Context, appID string) ([]models.AppInstanceResponse, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	pods, err := client.CoreV1().Pods(appCtx.EnvContext.Env.ClusterNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "ketches.cn/app-slug=" + appCtx.App.Slug,
	})
	if err != nil {
		return nil, err
	}

	var res []models.AppInstanceResponse
	for _, p := range pods.Items {
		res = append(res, core.ToAppInstanceResponse(&p))
	}
	return res, nil
}

func ListAppInstanceEvents(ctx context.Context, appCtx *models.AppContext, instanceName string) ([]models.AppInstanceEventResponse, error) {
	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	eventList, err := client.CoreV1().Events(appCtx.EnvContext.Env.ClusterNamespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + instanceName + ",involvedObject.kind=Pod",
	})
	if err != nil {
		return nil, err
	}

	var events []models.AppInstanceEventResponse
	for _, e := range eventList.Items {
		events = append(events, models.AppInstanceEventResponse{
			Type:      e.Type,
			Reason:    e.Reason,
			Message:   e.Message,
			From:      e.Source.Component,
			Count:     e.Count,
			CreatedAt: e.LastTimestamp.Time,
		})
	}

	return events, nil
}

func DeleteAppInstance(ctx context.Context, appCtx *models.AppContext, instanceName string) error {
	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	return client.CoreV1().Pods(appCtx.EnvContext.Env.ClusterNamespace).Delete(ctx, instanceName, metav1.DeleteOptions{})
}

func StreamAppLogs(ctx context.Context, appCtx *models.AppContext, instanceName, containerName string, tailLines int64, timestamps bool) (io.ReadCloser, error) {
	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	podLogOptions := &corev1.PodLogOptions{
		Container:  containerName,
		Follow:     true,
		TailLines:  &tailLines,
		Timestamps: timestamps,
	}

	req := client.CoreV1().Pods(appCtx.EnvContext.Env.ClusterNamespace).GetLogs(instanceName, podLogOptions)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open log stream: %w", err)
	}
	return stream, nil
}

func ExecAppContainer(appCtx *models.AppContext, instanceName, containerName string, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(appCtx.EnvContext.Cluster.KubeConfig))
	if err != nil {
		return fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(instanceName).
		Namespace(appCtx.EnvContext.Env.ClusterNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   []string{"sh", "-c", "clear; exec $(command -v bash || command -v ash || command -v sh)"},
			Stdin:     stdin != nil,
			Stdout:    stdout != nil,
			Stderr:    stderr != nil,
			TTY:       tty,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to create SPDY executor: %w", err)
	}

	streamOptions := remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	}
	if tty {
		streamOptions.TerminalSizeQueue = terminalSizeQueue
	}

	if err := executor.StreamWithContext(context.Background(), streamOptions); err != nil {
		return fmt.Errorf("failed to stream exec session: %w", err)
	}

	return nil
}

func AppAction(ctx context.Context, appID string, action app.AppAction) (*models.AppContext, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	currentStatus, err := core.CalculateAppStatus(ctx, appCtx)
	if err != nil {
		return nil, err
	}

	_, err = core.ValidateStateTransition(currentStatus, action)
	if err != nil {
		return nil, err
	}

	switch action {
	case app.AppActionDeploy:
		return executeDeployAction(ctx, appCtx)
	case app.AppActionStart:
		return executeStartAction(ctx, appCtx)
	case app.AppActionStop:
		return executeStopAction(ctx, appCtx)
	case app.AppActionUpdate:
		return executeUpdateAction(ctx, appCtx)
	case app.AppActionRedeploy:
		return executeRedeployAction(ctx, appCtx)
	case app.AppActionRollback:
		return executeRollbackAction(ctx, appCtx)
	case app.AppActionDebug:
		return executeDebugAction(ctx, appCtx)
	case app.AppActionDebugOff:
		return executeDebugOffAction(ctx, appCtx)
	case app.AppActionDelete:
		return executeDeleteAction(ctx, appCtx)
	default:
		return nil, errors.New("unsupported action")
	}
}

func executeDeployAction(ctx context.Context, appCtx *models.AppContext) (*models.AppContext, error) {
	if err := core.ApplyApp(ctx, appCtx); err != nil {
		return nil, err
	}

	appCtx.App.DeployStatus = app.AppStatusRunning
	if err := db.DB.Model(&appCtx.App).Update("deploy_status", app.AppStatusRunning).Error; err != nil {
		return nil, err
	}

	return appCtx, nil
}

func executeStartAction(ctx context.Context, appCtx *models.AppContext) (*models.AppContext, error) {
	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	switch appCtx.App.AppType {
	case app.AppTypeDeployment:
		deployment, err := client.AppsV1().Deployments(appCtx.EnvContext.Env.ClusterNamespace).Get(ctx, appCtx.App.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		replicas := int32(appCtx.App.Replicas)
		deployment.Spec.Replicas = &replicas

		if _, err := client.AppsV1().Deployments(appCtx.EnvContext.Env.ClusterNamespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}
	case app.AppTypeStatefulSet:
		statefulSet, err := client.AppsV1().StatefulSets(appCtx.EnvContext.Env.ClusterNamespace).Get(ctx, appCtx.App.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		replicas := int32(appCtx.App.Replicas)
		statefulSet.Spec.Replicas = &replicas

		if _, err := client.AppsV1().StatefulSets(appCtx.EnvContext.Env.ClusterNamespace).Update(ctx, statefulSet, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}
	}

	return appCtx, nil
}

func executeStopAction(ctx context.Context, appCtx *models.AppContext) (*models.AppContext, error) {
	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	replicas := int32(0)

	switch appCtx.App.AppType {
	case app.AppTypeDeployment:
		deployment, err := client.AppsV1().Deployments(appCtx.EnvContext.Env.ClusterNamespace).Get(ctx, appCtx.App.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		deployment.Spec.Replicas = &replicas

		if _, err := client.AppsV1().Deployments(appCtx.EnvContext.Env.ClusterNamespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}
	case app.AppTypeStatefulSet:
		statefulSet, err := client.AppsV1().StatefulSets(appCtx.EnvContext.Env.ClusterNamespace).Get(ctx, appCtx.App.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		statefulSet.Spec.Replicas = &replicas

		if _, err := client.AppsV1().StatefulSets(appCtx.EnvContext.Env.ClusterNamespace).Update(ctx, statefulSet, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}
	}

	return appCtx, nil
}

func executeUpdateAction(ctx context.Context, appCtx *models.AppContext) (*models.AppContext, error) {
	if err := core.ApplyApp(ctx, appCtx); err != nil {
		return nil, err
	}

	return appCtx, nil
}

func executeRedeployAction(ctx context.Context, appCtx *models.AppContext) (*models.AppContext, error) {
	if err := deleteAppK8sResources(ctx, appCtx, true); err != nil {
		return nil, err
	}
	return executeDeployAction(ctx, appCtx)
}

func executeRollbackAction(ctx context.Context, appCtx *models.AppContext) (*models.AppContext, error) {
	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return nil, err
	}
	// TODO: Implement rollback logic, such as using Deployment's revision history or StatefulSet's update strategy to rollback to previous version
	switch appCtx.App.AppType {
	case app.AppTypeDeployment:
		if _, err := client.AppsV1().Deployments(appCtx.EnvContext.Env.ClusterNamespace).Get(ctx, appCtx.App.Slug, metav1.GetOptions{}); err != nil {
			return nil, err
		}
	case app.AppTypeStatefulSet:
		if _, err := client.AppsV1().StatefulSets(appCtx.EnvContext.Env.ClusterNamespace).Get(ctx, appCtx.App.Slug, metav1.GetOptions{}); err != nil {
			return nil, err
		}
	}

	return appCtx, nil
}

func executeDebugAction(ctx context.Context, appCtx *models.AppContext) (*models.AppContext, error) {
	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	ns := appCtx.EnvContext.Env.ClusterNamespace

	switch appCtx.App.AppType {
	case app.AppTypeDeployment:
		deployment, err := client.AppsV1().Deployments(ns).Get(ctx, appCtx.App.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if deployment.Spec.Template.Labels == nil {
			deployment.Spec.Template.Labels = make(map[string]string)
		}
		deployment.Spec.Template.Labels[kube.LabelDebugging] = "true"

		appContainerName := "app-" + appCtx.App.Slug
		for i := range deployment.Spec.Template.Spec.Containers {
			if deployment.Spec.Template.Spec.Containers[i].Name == appContainerName {
				deployment.Spec.Template.Spec.Containers[i].Command = []string{"sh", "-c", "while true; do sleep 3600; done"}
				break
			}
		}

		if _, err := client.AppsV1().Deployments(ns).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}

	case app.AppTypeStatefulSet:
		statefulSet, err := client.AppsV1().StatefulSets(ns).Get(ctx, appCtx.App.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if statefulSet.Spec.Template.Labels == nil {
			statefulSet.Spec.Template.Labels = make(map[string]string)
		}
		statefulSet.Spec.Template.Labels[kube.LabelDebugging] = "true"

		appContainerName := "app-" + appCtx.App.Slug
		for i := range statefulSet.Spec.Template.Spec.Containers {
			if statefulSet.Spec.Template.Spec.Containers[i].Name == appContainerName {
				statefulSet.Spec.Template.Spec.Containers[i].Command = []string{"sh", "-c", "while true; do sleep 3600; done"}
				break
			}
		}

		if _, err := client.AppsV1().StatefulSets(ns).Update(ctx, statefulSet, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}
	}

	return appCtx, nil
}

func executeDebugOffAction(ctx context.Context, appCtx *models.AppContext) (*models.AppContext, error) {
	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	ns := appCtx.EnvContext.Env.ClusterNamespace

	switch appCtx.App.AppType {
	case app.AppTypeDeployment:
		deployment, err := client.AppsV1().Deployments(ns).Get(ctx, appCtx.App.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if deployment.Spec.Template.Labels != nil {
			delete(deployment.Spec.Template.Labels, kube.LabelDebugging)
		}

		appContainerName := "app-" + appCtx.App.Slug
		for i := range deployment.Spec.Template.Spec.Containers {
			if deployment.Spec.Template.Spec.Containers[i].Name == appContainerName {
				var containerCommand []string
				if appCtx.App.ContainerCommand != "" {
					containerCommand = []string{"sh", "-c", appCtx.App.ContainerCommand}
				}
				deployment.Spec.Template.Spec.Containers[i].Command = containerCommand
				break
			}
		}

		if _, err := client.AppsV1().Deployments(ns).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}

	case app.AppTypeStatefulSet:
		statefulSet, err := client.AppsV1().StatefulSets(ns).Get(ctx, appCtx.App.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if statefulSet.Spec.Template.Labels != nil {
			delete(statefulSet.Spec.Template.Labels, kube.LabelDebugging)
		}

		appContainerName := "app-" + appCtx.App.Slug
		for i := range statefulSet.Spec.Template.Spec.Containers {
			if statefulSet.Spec.Template.Spec.Containers[i].Name == appContainerName {
				statefulSet.Spec.Template.Spec.Containers[i].Command = nil
				break
			}
		}

		if _, err := client.AppsV1().StatefulSets(ns).Update(ctx, statefulSet, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}
	}

	return appCtx, nil
}

func executeDeleteAction(ctx context.Context, appCtx *models.AppContext) (*models.AppContext, error) {
	if err := DeleteApp(ctx, appCtx.App.ID); err != nil {
		return nil, err
	}
	return appCtx, nil
}

func ToAppResponse(c context.Context, appCtx *models.AppContext) models.AppResponse {
	status := GetAppStatus(c, appCtx)

	a := &appCtx.App
	res := models.AppResponse{
		ID:               a.ID,
		Slug:             a.Slug,
		Name:             a.Name,
		Description:      a.Description,
		EnvID:            a.EnvID,
		AppType:          a.AppType,
		CodeRepositoryID: derefString(a.CodeRepositoryID),
		ContainerImage:   a.ContainerImage,
		ContainerCommand: a.ContainerCommand,
		RegistryUsername: a.RegistryUsername,
		RegistryPassword: a.RegistryPassword,
		Replicas:         a.Replicas,
		RequestCPU:       a.RequestCPU,
		RequestMemory:    a.RequestMemory,
		LimitCPU:         a.LimitCPU,
		LimitMemory:      a.LimitMemory,
		Status:           status,
		CreatedAt:        a.CreatedAt,
	}

	if appCtx.AutoScaling != nil {
		res.AutoScaling = &models.AutoScalingSpec{
			MinReplicas:             appCtx.AutoScaling.MinReplicas,
			MaxReplicas:             appCtx.AutoScaling.MaxReplicas,
			TargetCPUUtilization:    appCtx.AutoScaling.TargetCPUUtilization,
			TargetMemoryUtilization: appCtx.AutoScaling.TargetMemoryUtilization,
		}
	}

	if appCtx.SchedulingRule != nil {
		res.SchedulingRule = &models.SchedulingSpec{
			RuleType:     appCtx.SchedulingRule.RuleType,
			NodeName:     appCtx.SchedulingRule.NodeName,
			NodeSelector: appCtx.SchedulingRule.NodeSelector,
			NodeAffinity: appCtx.SchedulingRule.NodeAffinity,
			Tolerations:  appCtx.SchedulingRule.Tolerations,
		}
	}

	if appCtx.EnvContext.Env.ID != "" {
		envResp := ToEnvResponse(&appCtx.EnvContext.Env)
		res.Env = &envResp
	}

	return res
}

func GetAppStatus(c context.Context, appCtx *models.AppContext) string {
	status := appCtx.App.DeployStatus
	if status == "deployed" {
		calculatedStatus, err := core.CalculateAppStatus(c, appCtx)
		if err != nil {
			log.Printf("Failed to calculate app status for app %s: %v", appCtx.App.ID, err)
		}
		status = string(calculatedStatus)
	}
	return status
}

// GetAppListRowStatus calculates the live status for an AppListRow.
// For deployed apps, it queries the Kubernetes cluster using the flat DTO fields.
func GetAppListRowStatus(ctx context.Context, row *models.AppListRow) string {
	status := row.DeployStatus
	if status == "deployed" && row.ClusterID != "" {
		client, err := kube.GlobalClusterStore.GetClient(row.ClusterID)
		if err != nil {
			log.Printf("Failed to get cluster client for app %s: %v", row.ID, err)
			return status
		}
		calculatedStatus, err := core.CalculateAppListStatus(ctx, client, row.ID, row.Slug, row.AppType, row.ClusterNamespace, row.Replicas)
		if err != nil {
			log.Printf("Failed to calculate app status for app %s: %v", row.ID, err)
		}
		status = string(calculatedStatus)
	}
	return status
}

// ToAppListResponse converts a flat AppListRow DTO into the API AppResponse.
func ToAppListResponse(ctx context.Context, row *models.AppListRow) models.AppResponse {
	status := GetAppListRowStatus(ctx, row)
	return models.AppResponse{
		ID:               row.ID,
		Slug:             row.Slug,
		Name:             row.Name,
		Description:      row.Description,
		EnvID:            row.EnvID,
		AppType:          row.AppType,
		CodeRepositoryID: derefString(row.CodeRepositoryID),
		ContainerImage:   row.ContainerImage,
		ContainerCommand: row.ContainerCommand,
		RegistryUsername: row.RegistryUsername,
		RegistryPassword: row.RegistryPassword,
		Replicas:         row.Replicas,
		RequestCPU:       row.RequestCPU,
		RequestMemory:    row.RequestMemory,
		LimitCPU:         row.LimitCPU,
		LimitMemory:      row.LimitMemory,
		Status:           status,
		CreatedAt:        row.CreatedAt,
		Env: &models.EnvResponse{
			ID:               row.EnvID,
			Slug:             row.EnvSlug,
			Name:             row.EnvName,
			ClusterID:        row.ClusterID,
			ClusterNamespace: row.ClusterNamespace,
			IsBuildEnv:       row.IsBuildEnv,
		},
	}
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func GetAppTopology(ctx context.Context, appID string) (*models.AppTopologyResponse, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	return core.GetAppTopology(ctx, client, appCtx)
}

func GetAppTopologyResourceYaml(ctx context.Context, appID string, nodeID string) (string, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return "", err
	}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return "", err
	}

	return core.GetAppTopologyResourceYaml(ctx, client, appCtx, nodeID)
}

// seedAppFromImageMetadata fetches the container image metadata and creates corresponding
// database records for env vars, volumes, gateways, and health check probes.
// Errors are logged and silently skipped to avoid blocking app creation.
func seedAppFromImageMetadata(ctx context.Context, application *entities.App) error {
	meta, err := containerregistry.FetchImageMetadata(
		ctx,
		application.ContainerImage,
		application.RegistryUsername,
		application.RegistryPassword,
	)
	if err != nil {
		return fmt.Errorf("fetch image metadata: %w", err)
	}

	// Seed EnvVars — skip if the key already exists (unique constraint)
	for _, ev := range meta.Env {
		envVar := &entities.AppEnvVar{
			ID:    uuid.New(),
			AppID: application.ID,
			Key:   ev.Key,
			Value: ev.Value,
		}
		// Ignore duplicate key errors; the app may have been pre-populated
		_ = db.DB.Create(envVar).Error
	}

	// Seed Volumes — skip duplicate mount paths
	for _, mountPath := range meta.Volumes {
		slug := strings.Trim(path.Base(mountPath), "/")
		if slug == "" || slug == "." {
			slug = "data"
		}
		volume := &entities.AppVolume{
			ID:         uuid.New(),
			AppID:      application.ID,
			Slug:       slug,
			MountPath:  mountPath,
			VolumeType: app.VolumeTypePVC,
			Capacity:   1,
		}
		// Ignore duplicate mount path errors
		_ = db.DB.Create(volume).Error
	}

	// Seed Gateways — skip duplicate port/protocol combinations
	for _, pi := range meta.ExposedPorts {
		// Map TCP→HTTP and UDP→UDP for gateway protocol
		protocol := "TCP"
		if pi.Protocol == "UDP" {
			protocol = "UDP"
		}
		gateway := &entities.AppGateway{
			ID:       uuid.New(),
			AppID:    application.ID,
			Port:     pi.Port,
			Protocol: protocol,
			Exposed:  false,
			Path:     "/",
		}
		// Ignore duplicate port/protocol errors
		_ = db.DB.Create(gateway).Error
	}

	// Seed health check probe from image HEALTHCHECK instruction
	if hc := meta.HealthCheck; hc != nil && len(hc.Test) > 1 {
		// HEALTHCHECK test slice: ["CMD", "arg1", "arg2"] or ["CMD-SHELL", "cmd string"]
		// Skip index 0 (CMD / CMD-SHELL) and join the rest as the exec command
		execCommand := strings.Join(hc.Test[1:], " ")

		intervalSecs := int(hc.Interval.Seconds())
		if intervalSecs <= 0 {
			intervalSecs = 10
		}
		timeoutSecs := int(hc.Timeout.Seconds())
		if timeoutSecs <= 0 {
			timeoutSecs = 1
		}
		retries := hc.Retries
		if retries <= 0 {
			retries = 3
		}

		probe := &entities.AppProbe{
			ID:               uuid.New(),
			AppID:            application.ID,
			Type:             "liveness",
			ProbeMode:        "exec",
			Enabled:          true,
			ExecCommand:      execCommand,
			PeriodSeconds:    intervalSecs,
			TimeoutSeconds:   timeoutSecs,
			FailureThreshold: retries,
			SuccessThreshold: 1,
		}
		_ = db.DB.Create(probe).Error
	}

	return nil
}
