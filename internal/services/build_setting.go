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
	"gorm.io/gorm"
)

// RegistrySummary holds the minimal registry fields joined from container_registries.
// RegistryID is already on entities.BuildSetting, so only name and provider are added here.
type RegistrySummary struct {
	RegName     string `gorm:"column:reg_name"`
	RegProvider string `gorm:"column:reg_provider"`
}

// BuildSettingWithRegistry joins build_settings with a minimal registry projection.
type BuildSettingWithRegistry struct {
	entities.BuildSetting
	RegistrySummary
}

// buildSettingSelectCols selects all build_settings columns plus the two extra registry fields
// the frontend actually renders. RegistryID is already on build_settings so no alias needed.
// Keeping the projection minimal avoids fetching sensitive registry fields (credentials, etc.).
const buildSettingSelectCols = `build_settings.*,
	cr.name     AS reg_name,
	cr.provider AS reg_provider`

func buildSettingQuery() *gorm.DB {
	return db.DB.Table("build_settings").
		Select(buildSettingSelectCols).
		Joins("LEFT JOIN container_registries cr ON cr.id = build_settings.registry_id")
}

// GetAppBuildSetting returns the build setting for an app (scope=app).
func GetAppBuildSetting(appID string) (*BuildSettingWithRegistry, error) {
	var s BuildSettingWithRegistry
	if err := buildSettingQuery().
		Where("build_settings.app_id = ?", appID).
		First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// GetBuildSetting returns a build setting by ID.
func GetBuildSetting(id string) (*BuildSettingWithRegistry, error) {
	var s BuildSettingWithRegistry
	if err := buildSettingQuery().
		Where("build_settings.id = ?", id).
		First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// ListRepoBuildSettings returns all build settings for a code repository.
func ListRepoBuildSettings(repoID string) ([]BuildSettingWithRegistry, error) {
	var settings []BuildSettingWithRegistry
	if err := buildSettingQuery().
		Where("build_settings.code_repository_id = ?", repoID).
		Order("build_settings.created_at").
		Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

// UpsertAppBuildSetting creates or updates the build setting for an app.
func UpsertAppBuildSetting(appID string, req *models.UpsertAppBuildSettingRequest) (*BuildSettingWithRegistry, error) {
	var s entities.BuildSetting
	err := db.DB.Where("app_id = ?", appID).First(&s).Error
	if err != nil {
		// Create
		s = entities.BuildSetting{
			ID:             uuid.New(),
			GitRef:         defaultStr(req.GitRef, "main"),
			DockerfilePath: defaultStr(req.DockerfilePath, "Dockerfile"),
			BuildContext:   defaultStr(req.BuildContext, "."),
			BuildArgs:      req.BuildArgs,
			ImageName:      req.ImageName,
			RegistryID:     req.RegistryID,
		}
		if err := db.DB.Create(&s).Error; err != nil {
			return nil, err
		}
	} else {
		s.GitRef = defaultStr(req.GitRef, "main")
		s.DockerfilePath = defaultStr(req.DockerfilePath, "Dockerfile")
		s.BuildContext = defaultStr(req.BuildContext, ".")
		s.BuildArgs = req.BuildArgs
		s.ImageName = req.ImageName
		s.RegistryID = req.RegistryID
		if err := db.DB.Save(&s).Error; err != nil {
			return nil, err
		}
	}
	return GetAppBuildSetting(appID)
}

// CreateRepoBuildSetting creates a new build setting under a code repository.
func CreateRepoBuildSetting(repoID string, req *models.CreateRepoBuildSettingRequest) (*BuildSettingWithRegistry, error) {
	s := entities.BuildSetting{
		ID:               uuid.New(),
		CodeRepositoryID: &repoID,
		Name:             req.Name,
		GitRef:           defaultStr(req.GitRef, "main"),
		DockerfilePath:   defaultStr(req.DockerfilePath, "Dockerfile"),
		BuildContext:     defaultStr(req.BuildContext, "."),
		BuildArgs:        req.BuildArgs,
		ImageName:        req.ImageName,
		RegistryID:       req.RegistryID,
	}
	if err := db.DB.Create(&s).Error; err != nil {
		return nil, err
	}
	return GetBuildSetting(s.ID)
}

// UpdateRepoBuildSetting updates a repo build setting by ID.
func UpdateRepoBuildSetting(id string, req *models.UpdateRepoBuildSettingRequest) (*BuildSettingWithRegistry, error) {
	// Fetch only the scalar entity for mutation — no registry JOIN needed here.
	var s entities.BuildSetting
	if err := db.DB.First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if req.Name != "" {
		s.Name = req.Name
	}
	if req.GitRef != "" {
		s.GitRef = req.GitRef
	}
	if req.DockerfilePath != "" {
		s.DockerfilePath = req.DockerfilePath
	}
	if req.BuildContext != "" {
		s.BuildContext = req.BuildContext
	}
	if req.ImageName != "" {
		s.ImageName = req.ImageName
	}
	if req.RegistryID != "" {
		s.RegistryID = req.RegistryID
	}
	s.BuildArgs = req.BuildArgs
	if err := db.DB.Save(&s).Error; err != nil {
		return nil, err
	}
	return GetBuildSetting(id)
}

// DeleteAppBuildSetting deletes the build setting for an app.
func DeleteAppBuildSetting(appID string) error {
	return db.DB.Where("app_id = ?", appID).Delete(&entities.BuildSetting{}).Error
}

// DeleteRepoBuildSetting deletes a repo build setting by ID.
func DeleteRepoBuildSetting(id string) error {
	// Prevent deletion if active builds reference this setting.
	var count int64
	if err := db.DB.Model(&entities.Build{}).Where("build_setting_id = ? AND status IN ?", id,
		[]entities.BuildStatus{entities.BuildStatusPending, entities.BuildStatusCloning, entities.BuildStatusBuilding}).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete: %d active build(s) are using this setting", count)
	}
	return db.DB.Delete(&entities.BuildSetting{}, "id = ?", id).Error
}

// ToBuildSettingResponse converts a BuildSettingWithRegistry to its API response model.
func ToBuildSettingResponse(s *BuildSettingWithRegistry) models.BuildSettingResponse {
	resp := models.BuildSettingResponse{
		ID:             s.ID,
		Name:           s.Name,
		GitRef:         s.GitRef,
		DockerfilePath: s.DockerfilePath,
		BuildContext:   s.BuildContext,
		ImageName:      s.ImageName,
		RegistryID:     s.RegistryID,
		BuildArgs:      s.BuildArgs,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
	if s.CodeRepositoryID != nil {
		resp.CodeRepositoryID = *s.CodeRepositoryID
	}
	if s.RegistryID != "" {
		resp.Registry = &models.RegistrySummaryResponse{
			ID:       s.RegistryID,
			Name:     s.RegName,
			Provider: s.RegProvider,
		}
	}
	return resp
}

// ListAvailableRegistriesForApp lists container registries available to an app's cluster/project.
func ListAvailableRegistriesForApp(appID string) ([]entities.ContainerRegistry, error) {
	appCtx, err := GetApp(context.Background(), appID)
	if err != nil {
		return nil, err
	}
	return ListAvailableRegistries(appCtx.EnvContext.Env.ClusterID, appCtx.EnvContext.Project.ID)
}

func TestGitConnection(req *models.TestGitConnectionRequest) *models.TestGitConnectionResponse {
	repoURL := req.GitRepoURL
	if req.GitUsername != "" && req.GitPassword != "" {
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
	if strings.HasPrefix(repoURL, "https://") {
		return fmt.Sprintf("https://%s:%s@%s", username, password, strings.TrimPrefix(repoURL, "https://"))
	}
	if strings.HasPrefix(repoURL, "http://") {
		return fmt.Sprintf("http://%s:%s@%s", username, password, strings.TrimPrefix(repoURL, "http://"))
	}
	return repoURL
}
