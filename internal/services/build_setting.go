package services

import (
	"fmt"
	"os/exec"
	"sort"
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

// CreateRepoBuildSetting creates a new build setting under a code repository.
func CreateRepoBuildSetting(repoID string, req *models.CreateBuildSettingRequest) (*BuildSettingWithRegistry, error) {
	buildArgs, _, err := normalizeStructuredBuildArgs(req.BuildArgs, req.BuildArgPairs)
	if err != nil {
		return nil, err
	}

	registryCacheEnabled := true
	if req.RegistryCacheEnabled != nil {
		registryCacheEnabled = *req.RegistryCacheEnabled
	}

	s := entities.BuildSetting{
		ID:                   uuid.New(),
		CodeRepositoryID:     &repoID,
		Name:                 req.Name,
		GitRef:               defaultStr(req.GitRef, "main"),
		DockerfilePath:       defaultStr(req.DockerfilePath, "Dockerfile"),
		BuildContext:         defaultStr(req.BuildContext, "."),
		BuildArgs:            buildArgs,
		ImageName:            req.ImageName,
		RegistryID:           req.RegistryID,
		Platforms:            normalizeBuildSettingPlatforms(req.Platforms),
		RegistryCacheEnabled: &registryCacheEnabled,
		RegistryCacheRef:     strings.TrimSpace(req.RegistryCacheRef),
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

	if req.Platforms != "" {
		s.Platforms = normalizeBuildSettingPlatforms(req.Platforms)
	}
	if req.RegistryCacheEnabled != nil {
		s.RegistryCacheEnabled = req.RegistryCacheEnabled
	}
	if req.RegistryCacheRef != "" {
		s.RegistryCacheRef = strings.TrimSpace(req.RegistryCacheRef)
	}
	if req.BuildArgPairs != nil || req.BuildArgs != "" {
		buildArgs, _, err := normalizeStructuredBuildArgs(req.BuildArgs, req.BuildArgPairs)
		if err != nil {
			return nil, err
		}
		s.BuildArgs = buildArgs
	}
	if err := db.DB.Save(&s).Error; err != nil {
		return nil, err
	}
	return GetBuildSetting(id)
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
		ID:                   s.ID,
		Name:                 s.Name,
		GitRef:               s.GitRef,
		DockerfilePath:       s.DockerfilePath,
		BuildContext:         s.BuildContext,
		ImageName:            s.ImageName,
		RegistryID:           s.RegistryID,
		BuildArgs:            s.BuildArgs,
		Platforms:            normalizeBuildSettingPlatforms(s.Platforms),
		RegistryCacheEnabled: buildSettingRegistryCacheEnabled(s.RegistryCacheEnabled),
		RegistryCacheRef:     s.RegistryCacheRef,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
	if pairs, ok := parseBuildArgPairs(s.BuildArgs); ok && len(pairs) > 0 {
		resp.BuildArgPairs = pairs
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

func normalizeBuildSettingPlatforms(raw string) string {
	switch strings.TrimSpace(raw) {
	case "", "linux/amd64":
		return "linux/amd64"
	case "linux/amd64,linux/arm64":
		return "linux/amd64,linux/arm64"
	default:
		return "linux/amd64"
	}
}

func normalizeStructuredBuildArgs(raw string, pairs []models.BuildArgPair) (string, []models.BuildArgPair, error) {
	if pairs == nil {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return "", nil, nil
		}
		parsed, ok := parseBuildArgPairs(raw)
		if !ok {
			return raw, nil, nil
		}
		return raw, parsed, nil
	}

	normalizedPairs := make([]models.BuildArgPair, 0, len(pairs))
	seen := make(map[string]struct{}, len(pairs))
	for _, pair := range pairs {
		key := strings.TrimSpace(pair.Key)
		if key == "" {
			return "", nil, fmt.Errorf("build arg key is required")
		}
		if _, exists := seen[key]; exists {
			return "", nil, fmt.Errorf("duplicate build arg key %q", key)
		}
		seen[key] = struct{}{}
		normalizedPairs = append(normalizedPairs, models.BuildArgPair{
			Key:   key,
			Value: strings.TrimSpace(pair.Value),
		})
	}

	sort.Slice(normalizedPairs, func(i, j int) bool {
		return normalizedPairs[i].Key < normalizedPairs[j].Key
	})

	lines := make([]string, 0, len(normalizedPairs))
	for _, pair := range normalizedPairs {
		lines = append(lines, fmt.Sprintf("%s=%s", pair.Key, pair.Value))
	}

	return strings.Join(lines, "\n"), normalizedPairs, nil
}

func parseBuildArgPairs(raw string) ([]models.BuildArgPair, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, true
	}

	lines := strings.Split(raw, "\n")
	pairs := make([]models.BuildArgPair, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, false
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, false
		}
		if _, exists := seen[key]; exists {
			return nil, false
		}
		seen[key] = struct{}{}
		pairs = append(pairs, models.BuildArgPair{
			Key:   key,
			Value: strings.TrimSpace(value),
		})
	}

	return pairs, true
}

func buildSettingRegistryCacheEnabled(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
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
