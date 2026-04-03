package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	platformUpdateConfigSettingKey            = "platform_update_config"
	platformUpdateNotificationStateSettingKey = "platform_update_notification_state"

	platformUpdateCheckModeAuto   = "auto"
	platformUpdateCheckModeManual = "manual"

	platformUpdateNotificationTitle     = "Platform update available"
	platformUpdateNotificationEventType = "platform_update_available"

	platformUpdateVersionSourceLocal      = "local"
	platformUpdateVersionSourceDeployment = "deployment"

	platformUpdateRolloutPhaseIdle        = "idle"
	platformUpdateRolloutPhaseAvailable   = "available"
	platformUpdateRolloutPhaseProgressing = "progressing"
	platformUpdateRolloutPhaseFailed      = "failed"
	platformUpdateRolloutPhaseBlocked     = "blocked"
)

type platformUpdateKubeClient = kubernetes.Interface

type platformUpdateNotificationState struct {
	LastNotifiedRecommendedVersion string `json:"last_notified_recommended_version"`
}

var (
	platformUpdateListTags = func(repo string) ([]string, error) {
		return crane.ListTags(strings.TrimPrefix(strings.TrimSpace(repo), "oci://"))
	}
	platformUpdateInClusterConfig        = rest.InClusterConfig
	platformUpdateNewKubeClientForConfig = func(cfg *rest.Config) (platformUpdateKubeClient, error) {
		return kubernetes.NewForConfig(cfg)
	}
)

type platformUpdateComponentDetails struct {
	Config     models.PlatformUpdateTargetConfig
	Repository string
	Name       string
}

func GetPlatformUpdateConfig() (*models.PlatformUpdateConfig, error) {
	setting, err := getSystemSetting(platformUpdateConfigSettingKey)
	if err != nil {
		return nil, err
	}
	if setting == nil || strings.TrimSpace(setting.Value) == "" {
		cfg := defaultPlatformUpdateConfig()
		return &cfg, nil
	}

	var cfg models.PlatformUpdateConfig
	if err := json.Unmarshal([]byte(setting.Value), &cfg); err != nil {
		return nil, app.WrapErrorf(err, "failed to decode platform update config: %w", err)
	}

	normalized, err := normalizePlatformUpdateConfig(&cfg)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}

func UpdatePlatformUpdateConfig(cfg *models.PlatformUpdateConfig, _ *app.Claims) (*models.PlatformUpdateConfig, error) {
	normalized, err := normalizePlatformUpdateConfig(cfg)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(normalized)
	if err != nil {
		return nil, app.WrapErrorf(err, "failed to encode platform update config: %w", err)
	}

	setting, err := getSystemSetting(platformUpdateConfigSettingKey)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		if err := db.DB.Create(&entities.SystemSetting{
			Base:  entities.Base{ID: uuid.New()},
			Key:   platformUpdateConfigSettingKey,
			Value: string(payload),
		}).Error; err != nil {
			return nil, err
		}
		return normalized, nil
	}

	if err := db.DB.Model(&entities.SystemSetting{}).
		Where("id = ?", setting.ID).
		Update("value", string(payload)).Error; err != nil {
		return nil, err
	}

	return normalized, nil
}

func GetPlatformUpdateStatus() (*models.PlatformUpdateStatus, error) {
	return buildPlatformUpdateStatus()
}

func CheckPlatformUpdate(req *models.CheckPlatformUpdateRequest, _ *app.Claims) (*models.PlatformUpdateStatus, error) {
	if req == nil {
		return nil, errors.New("check request is required")
	}
	if req.Mode != platformUpdateCheckModeAuto && req.Mode != platformUpdateCheckModeManual {
		return nil, errors.New("check mode must be auto or manual")
	}

	status, err := buildPlatformUpdateStatus()
	if err != nil {
		return nil, err
	}
	if req.Mode != platformUpdateCheckModeAuto {
		return status, nil
	}
	if err := maybeCreatePlatformUpdateNotifications(status); err != nil {
		return nil, err
	}
	return status, nil
}

func buildPlatformUpdateStatus() (*models.PlatformUpdateStatus, error) {
	cfg, err := GetPlatformUpdateConfig()
	if err != nil {
		return nil, err
	}

	status := &models.PlatformUpdateStatus{
		LocalPlatformVersion: app.Version,
		Config:               *cfg,
		RolloutBlockers:      []string{},
		API: models.PlatformUpdateComponentStatus{
			CurrentVersion: app.Version,
			VersionSource:  platformUpdateVersionSourceLocal,
		},
		UI: models.PlatformUpdateComponentStatus{
			CurrentVersion: app.Version,
			VersionSource:  platformUpdateVersionSourceLocal,
		},
	}

	apiVersions, apiErr := platformUpdateAvailableVersions(cfg.API.ImageRepository)
	uiVersions, uiErr := platformUpdateAvailableVersions(cfg.UI.ImageRepository)

	status.API.AvailableVersions = apiVersions
	status.UI.AvailableVersions = uiVersions
	if len(apiVersions) > 0 {
		status.API.LatestVersion = apiVersions[0]
	}
	if len(uiVersions) > 0 {
		status.UI.LatestVersion = uiVersions[0]
	}
	status.RecommendedSharedVersion = recommendedSharedPlatformVersion(apiVersions, uiVersions)

	if apiErr != nil {
		status.API.Error = apiErr.Error()
		status.RolloutBlockers = append(status.RolloutBlockers, "failed to list API image tags")
	}
	if uiErr != nil {
		status.UI.Error = uiErr.Error()
		status.RolloutBlockers = append(status.RolloutBlockers, "failed to list UI image tags")
	}

	inClusterConfig, err := platformUpdateInClusterConfig()
	if err != nil {
		status.API.UpdateAvailable = comparePlatformUpdateVersions(status.API.CurrentVersion, status.API.LatestVersion) < 0
		status.UI.UpdateAvailable = comparePlatformUpdateVersions(status.UI.CurrentVersion, status.UI.LatestVersion) < 0
		status.CanRollout = false
		status.RunningInKubernetes = false
		status.RolloutBlockers = append(status.RolloutBlockers, "current platform is not running in kubernetes")
		status.API.RolloutPhase = platformUpdatePhaseForComponent(status.API, false)
		status.UI.RolloutPhase = platformUpdatePhaseForComponent(status.UI, false)
		return status, nil
	}

	status.RunningInKubernetes = true
	client, err := platformUpdateNewKubeClientForConfig(inClusterConfig)
	if err != nil {
		status.CanRollout = false
		status.RunningInKubernetes = true
		status.RolloutBlockers = append(status.RolloutBlockers, "failed to create in-cluster kubernetes client")
		status.API.RolloutPhase = platformUpdatePhaseForComponent(status.API, false)
		status.UI.RolloutPhase = platformUpdatePhaseForComponent(status.UI, false)
		return status, nil
	}

	apiDeployment, apiErr := platformUpdateLoadDeployment(context.Background(), client, cfg.API)
	uiDeployment, uiErr := platformUpdateLoadDeployment(context.Background(), client, cfg.UI)

	if apiErr != nil {
		status.API.Error = apiErr.Error()
		status.RolloutBlockers = append(status.RolloutBlockers, "failed to inspect API deployment")
	} else {
		status.API.CurrentVersion = extractImageTag(apiDeployment.Spec.Template.Spec.Containers, cfg.API.ContainerName)
		if status.API.CurrentVersion == "" {
			status.API.Error = "failed to determine API deployment image tag"
			status.RolloutBlockers = append(status.RolloutBlockers, "failed to determine API deployment image tag")
		} else {
			status.API.VersionSource = platformUpdateVersionSourceDeployment
		}
	}

	if uiErr != nil {
		status.UI.Error = uiErr.Error()
		status.RolloutBlockers = append(status.RolloutBlockers, "failed to inspect UI deployment")
	} else {
		status.UI.CurrentVersion = extractImageTag(uiDeployment.Spec.Template.Spec.Containers, cfg.UI.ContainerName)
		if status.UI.CurrentVersion == "" {
			status.UI.Error = "failed to determine UI deployment image tag"
			status.RolloutBlockers = append(status.RolloutBlockers, "failed to determine UI deployment image tag")
		} else {
			status.UI.VersionSource = platformUpdateVersionSourceDeployment
		}
	}

	status.API.UpdateAvailable = comparePlatformUpdateVersions(status.API.CurrentVersion, status.API.LatestVersion) < 0
	status.UI.UpdateAvailable = comparePlatformUpdateVersions(status.UI.CurrentVersion, status.UI.LatestVersion) < 0
	status.CanRollout = len(status.RolloutBlockers) == 0
	status.API.RolloutPhase = platformUpdatePhaseForComponent(status.API, status.CanRollout)
	status.UI.RolloutPhase = platformUpdatePhaseForComponent(status.UI, status.CanRollout)

	if apiDeployment != nil {
		status.API.RolloutMessage = platformUpdateRolloutMessage(apiDeployment, cfg.API.ContainerName)
	}
	if uiDeployment != nil {
		status.UI.RolloutMessage = platformUpdateRolloutMessage(uiDeployment, cfg.UI.ContainerName)
	}

	return status, nil
}

func TriggerPlatformRollout(req *models.TriggerPlatformRolloutRequest, claims *app.Claims) (*models.PlatformUpdateRolloutResult, error) {
	if req == nil {
		return nil, errors.New("rollout request is required")
	}

	cfg, err := GetPlatformUpdateConfig()
	if err != nil {
		return nil, err
	}

	apiVersion, uiVersion, err := resolvePlatformRolloutVersions(req)
	if err != nil {
		return nil, err
	}

	inClusterConfig, err := platformUpdateInClusterConfig()
	if err != nil {
		return nil, errors.New("current platform is not running in kubernetes")
	}

	client, err := platformUpdateNewKubeClientForConfig(inClusterConfig)
	if err != nil {
		return nil, app.WrapErrorf(err, "failed to create in-cluster kubernetes client: %w", err)
	}

	apiVersions, err := platformUpdateAvailableVersions(cfg.API.ImageRepository)
	if err != nil {
		return nil, app.WrapErrorf(err, "failed to list API image tags: %w", err)
	}
	uiVersions, err := platformUpdateAvailableVersions(cfg.UI.ImageRepository)
	if err != nil {
		return nil, app.WrapErrorf(err, "failed to list UI image tags: %w", err)
	}
	if !slices.Contains(apiVersions, apiVersion) {
		return nil, app.NewErrorf("API version %q is not available", apiVersion)
	}
	if !slices.Contains(uiVersions, uiVersion) {
		return nil, app.NewErrorf("UI version %q is not available", uiVersion)
	}

	ctx := context.Background()
	apiDeployment, err := platformUpdateLoadDeployment(ctx, client, cfg.API)
	if err != nil {
		return nil, err
	}
	uiDeployment, err := platformUpdateLoadDeployment(ctx, client, cfg.UI)
	if err != nil {
		return nil, err
	}

	apiPreviousImage, err := findContainerImage(apiDeployment.Spec.Template.Spec.Containers, cfg.API.ContainerName)
	if err != nil {
		return nil, err
	}
	uiPreviousImage, err := findContainerImage(uiDeployment.Spec.Template.Spec.Containers, cfg.UI.ContainerName)
	if err != nil {
		return nil, err
	}

	result := &models.PlatformUpdateRolloutResult{
		API: models.PlatformUpdateRolloutTarget{
			Namespace:      cfg.API.Namespace,
			DeploymentName: cfg.API.DeploymentName,
			ContainerName:  cfg.API.ContainerName,
			PreviousImage:  apiPreviousImage,
			TargetImage:    fmt.Sprintf("%s:%s", cfg.API.ImageRepository, apiVersion),
		},
		UI: models.PlatformUpdateRolloutTarget{
			Namespace:      cfg.UI.Namespace,
			DeploymentName: cfg.UI.DeploymentName,
			ContainerName:  cfg.UI.ContainerName,
			PreviousImage:  uiPreviousImage,
			TargetImage:    fmt.Sprintf("%s:%s", cfg.UI.ImageRepository, uiVersion),
		},
	}

	if err := platformUpdatePatchDeploymentImage(ctx, client, cfg.UI, result.UI.TargetImage); err != nil {
		logPlatformRolloutEvent(claims, entities.OperationLogStatusFailure, "failed to patch UI deployment", result, false, false)
		return nil, err
	}

	if err := platformUpdatePatchDeploymentImage(ctx, client, cfg.API, result.API.TargetImage); err != nil {
		result.RollbackAttempted = true
		rollbackErr := platformUpdatePatchDeploymentImage(ctx, client, cfg.UI, uiPreviousImage)
		result.RollbackSucceeded = rollbackErr == nil
		logPlatformRolloutEvent(claims, entities.OperationLogStatusFailure, err.Error(), result, true, rollbackErr == nil)
		if rollbackErr != nil {
			return nil, app.WrapErrorf(err, "%w; UI rollback failed: %v", err, rollbackErr)
		}
		return nil, err
	}

	logPlatformRolloutEvent(claims, entities.OperationLogStatusSuccess, "platform rollout submitted", result, false, false)
	return result, nil
}

func defaultPlatformUpdateConfig() models.PlatformUpdateConfig {
	return models.PlatformUpdateConfig{
		API: models.PlatformUpdateTargetConfig{
			ImageRepository: "ghcr.io/ketches/ketches-api",
			Namespace:       "ketches",
			DeploymentName:  "ketches-api",
			ContainerName:   "ketches-api",
		},
		UI: models.PlatformUpdateTargetConfig{
			ImageRepository: "ghcr.io/ketches/ketches-ui",
			Namespace:       "ketches",
			DeploymentName:  "ketches-ui",
			ContainerName:   "ketches-ui",
		},
	}
}

func normalizePlatformUpdateConfig(cfg *models.PlatformUpdateConfig) (*models.PlatformUpdateConfig, error) {
	if cfg == nil {
		return nil, errors.New("platform update config is required")
	}

	normalized := *cfg
	normalized.API.ImageRepository = strings.TrimSpace(normalized.API.ImageRepository)
	normalized.API.Namespace = strings.TrimSpace(normalized.API.Namespace)
	normalized.API.DeploymentName = strings.TrimSpace(normalized.API.DeploymentName)
	normalized.API.ContainerName = strings.TrimSpace(normalized.API.ContainerName)
	normalized.UI.ImageRepository = strings.TrimSpace(normalized.UI.ImageRepository)
	normalized.UI.Namespace = strings.TrimSpace(normalized.UI.Namespace)
	normalized.UI.DeploymentName = strings.TrimSpace(normalized.UI.DeploymentName)
	normalized.UI.ContainerName = strings.TrimSpace(normalized.UI.ContainerName)

	for _, target := range []platformUpdateComponentDetails{
		{Config: normalized.API, Name: "API"},
		{Config: normalized.UI, Name: "UI"},
	} {
		if target.Config.ImageRepository == "" || target.Config.Namespace == "" || target.Config.DeploymentName == "" || target.Config.ContainerName == "" {
			return nil, app.NewErrorf("%s rollout target is incomplete", target.Name)
		}
	}

	return &normalized, nil
}

func maybeCreatePlatformUpdateNotifications(status *models.PlatformUpdateStatus) error {
	if status == nil {
		return nil
	}

	recommendedVersion := strings.TrimSpace(status.RecommendedSharedVersion)
	if recommendedVersion == "" {
		return nil
	}

	state, err := getPlatformUpdateNotificationState()
	if err != nil {
		return err
	}
	if normalizePlatformVersion(state.LastNotifiedRecommendedVersion) == normalizePlatformVersion(recommendedVersion) {
		return nil
	}

	payload, err := json.Marshal(map[string]string{
		"recommended_version": recommendedVersion,
		"api_latest_version":  status.API.LatestVersion,
		"ui_latest_version":   status.UI.LatestVersion,
	})
	if err != nil {
		return app.WrapErrorf(err, "failed to encode platform update notification payload: %w", err)
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		var admins []entities.User
		if err := tx.Where("role = ?", app.UserRoleAdmin).Find(&admins).Error; err != nil {
			return err
		}

		for _, adminUser := range admins {
			notif := &entities.Notification{
				Base:         entities.Base{ID: uuid.New()},
				RecipientID:  adminUser.ID,
				Category:     "info",
				EventType:    platformUpdateNotificationEventType,
				Title:        platformUpdateNotificationTitle,
				Message:      fmt.Sprintf("Recommended version %s is available for Ketches API and UI.", recommendedVersion),
				Status:       "pending",
				ResourceType: "platform",
				ResourceID:   "platform",
				ActionData:   string(payload),
			}
			if err := tx.Create(notif).Error; err != nil {
				return err
			}
		}

		return savePlatformUpdateNotificationState(tx, platformUpdateNotificationState{
			LastNotifiedRecommendedVersion: recommendedVersion,
		})
	})
}

func getPlatformUpdateNotificationState() (*platformUpdateNotificationState, error) {
	setting, err := getSystemSetting(platformUpdateNotificationStateSettingKey)
	if err != nil {
		return nil, err
	}
	if setting == nil || strings.TrimSpace(setting.Value) == "" {
		return &platformUpdateNotificationState{}, nil
	}

	var state platformUpdateNotificationState
	if err := json.Unmarshal([]byte(setting.Value), &state); err != nil {
		return nil, app.WrapErrorf(err, "failed to decode platform update notification state: %w", err)
	}
	return &state, nil
}

func savePlatformUpdateNotificationState(tx *gorm.DB, state platformUpdateNotificationState) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return app.WrapErrorf(err, "failed to encode platform update notification state: %w", err)
	}

	var setting entities.SystemSetting
	if err := systemSettingLookupQuery(tx, platformUpdateNotificationStateSettingKey).First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&entities.SystemSetting{
				Base:  entities.Base{ID: uuid.New()},
				Key:   platformUpdateNotificationStateSettingKey,
				Value: string(payload),
			}).Error
		}
		return err
	}

	return tx.Model(&entities.SystemSetting{}).
		Where("id = ?", setting.ID).
		Update("value", string(payload)).Error
}

func filterPlatformUpdateVersions(tags []string) []string {
	normalizedToOriginal := make(map[string]string)
	versions := make([]*semver.Version, 0, len(tags))
	for _, tag := range tags {
		v, err := semver.NewVersion(strings.TrimSpace(tag))
		if err != nil {
			continue
		}
		key := v.Original()
		normalized := v.String()
		if _, exists := normalizedToOriginal[normalized]; exists {
			continue
		}
		normalizedToOriginal[normalized] = key
		versions = append(versions, v)
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].GreaterThan(versions[j])
	})

	result := make([]string, 0, len(versions))
	for _, version := range versions {
		result = append(result, normalizedToOriginal[version.String()])
	}
	return result
}

func recommendedSharedPlatformVersion(apiVersions, uiVersions []string) string {
	uiSet := make(map[string]struct{}, len(uiVersions))
	for _, version := range uiVersions {
		normalized := normalizePlatformVersion(version)
		if normalized != "" {
			uiSet[normalized] = struct{}{}
		}
	}
	for _, version := range apiVersions {
		normalized := normalizePlatformVersion(version)
		if normalized == "" {
			continue
		}
		if _, ok := uiSet[normalized]; ok {
			return version
		}
	}
	return ""
}

func platformUpdateAvailableVersions(repository string) ([]string, error) {
	tags, err := platformUpdateListTags(repository)
	if err != nil {
		return []string{}, err
	}
	return filterPlatformUpdateVersions(tags), nil
}

func normalizePlatformVersion(version string) string {
	parsed, err := semver.NewVersion(strings.TrimSpace(version))
	if err != nil {
		return ""
	}
	return parsed.String()
}

func comparePlatformUpdateVersions(current, latest string) int {
	currentVersion, err := semver.NewVersion(strings.TrimSpace(current))
	if err != nil {
		return 0
	}
	latestVersion, err := semver.NewVersion(strings.TrimSpace(latest))
	if err != nil {
		return 0
	}
	return currentVersion.Compare(latestVersion)
}

func platformUpdateLoadDeployment(ctx context.Context, client platformUpdateKubeClient, target models.PlatformUpdateTargetConfig) (*appsv1.Deployment, error) {
	deployment, err := client.AppsV1().Deployments(target.Namespace).Get(ctx, target.DeploymentName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, app.NewErrorf("deployment %s/%s not found", target.Namespace, target.DeploymentName)
		}
		return nil, app.WrapErrorf(err, "failed to get deployment %s/%s: %w", target.Namespace, target.DeploymentName, err)
	}
	if _, err := findContainerImage(deployment.Spec.Template.Spec.Containers, target.ContainerName); err != nil {
		return nil, err
	}
	return deployment, nil
}

func findContainerImage(containers []corev1.Container, containerName string) (string, error) {
	for _, container := range containers {
		if container.Name == containerName {
			return container.Image, nil
		}
	}
	return "", app.NewErrorf("container %q not found", containerName)
}

func extractImageTag(containers []corev1.Container, containerName string) string {
	image, err := findContainerImage(containers, containerName)
	if err != nil {
		return ""
	}
	ref, err := name.ParseReference(image, name.WeakValidation)
	if err != nil {
		return ""
	}
	tag, ok := ref.(name.Tag)
	if !ok {
		return ""
	}
	return tag.TagStr()
}

func platformUpdatePatchDeploymentImage(ctx context.Context, client platformUpdateKubeClient, target models.PlatformUpdateTargetConfig, image string) error {
	deployment, err := platformUpdateLoadDeployment(ctx, client, target)
	if err != nil {
		return err
	}
	for i := range deployment.Spec.Template.Spec.Containers {
		if deployment.Spec.Template.Spec.Containers[i].Name == target.ContainerName {
			deployment.Spec.Template.Spec.Containers[i].Image = image
			_, err = client.AppsV1().Deployments(target.Namespace).Patch(
				ctx,
				target.DeploymentName,
				types.MergePatchType,
				[]byte(fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":"%s","image":"%s"}]}}}}`, target.ContainerName, image)),
				metav1.PatchOptions{},
			)
			return err
		}
	}
	return app.NewErrorf("container %q not found", target.ContainerName)
}

func platformUpdatePhaseForComponent(component models.PlatformUpdateComponentStatus, canRollout bool) string {
	if component.Error != "" || !canRollout {
		return platformUpdateRolloutPhaseBlocked
	}
	if component.RolloutMessage != "" && strings.Contains(strings.ToLower(component.RolloutMessage), "progress") {
		return platformUpdateRolloutPhaseProgressing
	}
	if component.UpdateAvailable {
		return platformUpdateRolloutPhaseAvailable
	}
	return platformUpdateRolloutPhaseIdle
}

func platformUpdateRolloutMessage(deployment *appsv1.Deployment, containerName string) string {
	if deployment == nil {
		return ""
	}
	desiredReplicas := int32(1)
	if deployment.Spec.Replicas != nil {
		desiredReplicas = *deployment.Spec.Replicas
	}
	if deployment.Status.UpdatedReplicas < desiredReplicas || deployment.Status.AvailableReplicas < desiredReplicas {
		return fmt.Sprintf("rollout progressing for %s", containerName)
	}
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing && condition.Message != "" {
			return condition.Message
		}
	}
	return ""
}

func resolvePlatformRolloutVersions(req *models.TriggerPlatformRolloutRequest) (string, string, error) {
	shared := strings.TrimSpace(req.SharedVersion)
	apiVersion := strings.TrimSpace(req.APIVersion)
	uiVersion := strings.TrimSpace(req.UIVersion)

	if apiVersion == "" {
		apiVersion = shared
	}
	if uiVersion == "" {
		uiVersion = shared
	}
	if apiVersion == "" || uiVersion == "" {
		return "", "", errors.New("shared_version or both api_version and ui_version are required")
	}
	return apiVersion, uiVersion, nil
}

func logPlatformRolloutEvent(claims *app.Claims, status, summary string, result *models.PlatformUpdateRolloutResult, rollbackAttempted, rollbackSucceeded bool) {
	if claims == nil || result == nil {
		return
	}
	message := fmt.Sprintf(
		"api:%s -> %s; ui:%s -> %s; rollback_attempted=%t; rollback_succeeded=%t; %s",
		result.API.PreviousImage,
		result.API.TargetImage,
		result.UI.PreviousImage,
		result.UI.TargetImage,
		rollbackAttempted,
		rollbackSucceeded,
		summary,
	)
	_ = CreateOperationLog(CreateOperationLogInput{
		UserID:         claims.UserID,
		Username:       claims.Username,
		Action:         "rollout",
		ResourceType:   "platform_update_rollout",
		ResourceID:     "platform",
		Status:         status,
		StatusCode:     statusCodeForOperationLogStatus(status),
		Sensitivity:    entities.OperationLogSensitivitySensitive,
		RequestSummary: message,
	})
}

func statusCodeForOperationLogStatus(status string) int {
	if status == entities.OperationLogStatusSuccess {
		return 200
	}
	return 500
}
