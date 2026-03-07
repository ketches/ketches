package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

type AppBuildConfigWithRegistry struct {
	entities.AppBuildConfig
	Registry entities.ContainerRegistry `gorm:"embedded;embeddedPrefix:registry_"`
}

func GetBuildConfig(appID string) (*AppBuildConfigWithRegistry, error) {
	var config AppBuildConfigWithRegistry
	if err := db.DB.Table("app_build_configs").
		Select("app_build_configs.*, container_registries.id AS registry_id, container_registries.created_at AS registry_created_at, container_registries.updated_at AS registry_updated_at, container_registries.name AS registry_name, container_registries.provider AS registry_provider, container_registries.endpoint AS registry_endpoint, container_registries.skip_tls_verify AS registry_skip_tls_verify, container_registries.namespace AS registry_namespace, container_registries.username AS registry_username, container_registries.password AS registry_password, container_registries.scope AS registry_scope, container_registries.cluster_id AS registry_cluster_id, container_registries.project_id AS registry_project_id, container_registries.is_default AS registry_is_default, container_registries.enabled AS registry_enabled, container_registries.description AS registry_description").
		Joins("LEFT JOIN container_registries ON container_registries.id = app_build_configs.registry_id").
		Where("app_build_configs.app_id = ?", appID).
		First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func UpsertBuildConfig(appID string, req *models.UpsertBuildConfigRequest) (*AppBuildConfigWithRegistry, error) {
	var config entities.AppBuildConfig
	err := db.DB.Where("app_id = ?", appID).First(&config).Error

	if err != nil {
		// Create new
		webhookSecret, _ := generateWebhookSecret()
		config = entities.AppBuildConfig{
			ID:             uuid.New(),
			AppID:          appID,
			GitRepoURL:     req.GitRepoURL,
			GitRef:         defaultStr(req.GitRef, "main"),
			GitUsername:    req.GitUsername,
			GitPassword:    req.GitPassword,
			DockerfilePath: defaultStr(req.DockerfilePath, "Dockerfile"),
			BuildContext:   defaultStr(req.BuildContext, "."),
			ImageName:      req.ImageName,
			RegistryID:     req.RegistryID,
			BuildArgs:      req.BuildArgs,
			AutoBuild:      req.AutoBuild,
			AutoDeploy:     req.AutoDeploy,
			WebhookSecret:  webhookSecret,
			WebhookEnabled: req.WebhookEnabled,
		}
		if err := db.DB.Create(&config).Error; err != nil {
			return nil, err
		}
	} else {
		// Update existing
		config.GitRepoURL = req.GitRepoURL
		config.GitRef = defaultStr(req.GitRef, "main")
		config.GitUsername = req.GitUsername
		if req.GitPassword != "" {
			config.GitPassword = req.GitPassword
		}
		config.DockerfilePath = defaultStr(req.DockerfilePath, "Dockerfile")
		config.BuildContext = defaultStr(req.BuildContext, ".")
		config.ImageName = req.ImageName
		config.RegistryID = req.RegistryID
		config.BuildArgs = req.BuildArgs
		config.AutoBuild = req.AutoBuild
		config.AutoDeploy = req.AutoDeploy
		config.WebhookEnabled = req.WebhookEnabled

		if err := db.DB.Save(&config).Error; err != nil {
			return nil, err
		}
	}

	// Reload with registry
	return GetBuildConfig(appID)
}

func DeleteBuildConfig(appID string) error {
	return db.DB.Where("app_id = ?", appID).Delete(&entities.AppBuildConfig{}).Error
}

func TestGitConnection(req *models.TestGitConnectionRequest) *models.TestGitConnectionResponse {
	repoURL := req.GitRepoURL
	if req.GitUsername != "" && req.GitPassword != "" {
		// Inject credentials into URL for testing
		repoURL = injectGitCredentials(repoURL, req.GitUsername, req.GitPassword)
	}

	ref := req.GitRef
	if ref == "" {
		ref = "HEAD"
	}

	cmd := exec.Command("git", "ls-remote", "--exit-code", repoURL, ref)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &models.TestGitConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Git connection failed: %s", strings.TrimSpace(string(output))),
		}
	}

	return &models.TestGitConnectionResponse{
		Success: true,
		Message: "Git repository is accessible",
	}
}

func ListAvailableRegistriesForApp(appID string) ([]entities.ContainerRegistry, error) {
	appCtx, err := GetApp(context.Background(), appID)
	if err != nil {
		return nil, err
	}

	return ListAvailableRegistries(appCtx.EnvContext.Env.ClusterID, appCtx.EnvContext.Project.ID)
}

func ToBuildConfigResponse(config *AppBuildConfigWithRegistry) models.BuildConfigResponse {
	resp := models.BuildConfigResponse{
		ID:             config.ID,
		AppID:          config.AppID,
		GitRepoURL:     config.GitRepoURL,
		GitRef:         config.GitRef,
		GitUsername:    config.GitUsername,
		DockerfilePath: config.DockerfilePath,
		BuildContext:   config.BuildContext,
		ImageName:      config.ImageName,
		RegistryID:     config.RegistryID,
		BuildArgs:      config.BuildArgs,
		AutoBuild:      config.AutoBuild,
		AutoDeploy:     config.AutoDeploy,
		WebhookSecret:  config.WebhookSecret,
		WebhookEnabled: config.WebhookEnabled,
		CreatedAt:      config.CreatedAt,
		UpdatedAt:      config.UpdatedAt,
	}

	if config.Registry.ID != "" {
		regResp := ToContainerRegistryResponse(&config.Registry)
		resp.Registry = &regResp
	}

	return resp
}

func generateWebhookSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func defaultStr(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func injectGitCredentials(repoURL, username, password string) string {
	// https://github.com/user/repo.git -> https://user:pass@github.com/user/repo.git
	if strings.HasPrefix(repoURL, "https://") {
		return fmt.Sprintf("https://%s:%s@%s", username, password, strings.TrimPrefix(repoURL, "https://"))
	}
	if strings.HasPrefix(repoURL, "http://") {
		return fmt.Sprintf("http://%s:%s@%s", username, password, strings.TrimPrefix(repoURL, "http://"))
	}
	return repoURL
}
