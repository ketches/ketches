package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation"
)

func ListEnvs(projectID string, page, pageSize int, search string) (int64, []models.EnvResponse, error) {
	var envs []models.EnvResponse
	var total int64
	query := db.DB.Model(&entities.Env{}).Select("envs.id, envs.name, envs.slug, envs.description, envs.project_id, envs.cluster_id, clusters.name as cluster_name, clusters.connection_status as cluster_connection_status, clusters.connection_status_reason as cluster_connection_status_reason, envs.cluster_namespace, envs.is_build_env, envs.created_at").Where("project_id = ?", projectID).Joins("JOIN clusters ON clusters.id = envs.cluster_id").Order("created_at")
	if search != "" {
		query = query.Where("name LIKE ? OR slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&envs).Error; err != nil {
		return 0, nil, err
	}
	return total, envs, nil
}

func ListEnvsSimple(projectID string) ([]models.EnvResponse, error) {
	var envs []models.EnvResponse
	if err := db.DB.Model(&entities.Env{}).Select("id, slug, name, description, cluster_id, cluster_namespace, is_build_env").Where("project_id = ?", projectID).Order("created_at").Find(&envs).Error; err != nil {
		return nil, err
	}
	return envs, nil
}

func CreateEnv(projectID string, req *models.CreateEnvRequest) (*entities.Env, error) {
	var project entities.Project
	if err := db.DB.First(&project, "id = ?", projectID).Error; err != nil {
		return nil, err
	}

	var existing entities.Env
	if err := db.DB.Where("project_id = ? AND slug = ?", projectID, req.Slug).First(&existing).Error; err == nil {
		return nil, errors.New("environment with this slug already exists in this project")
	}

	namespaceName := strings.TrimSpace(req.ClusterNamespace)
	if namespaceName == "" {
		namespaceName = core.GenerateNamespaceName(project.Slug, req.Slug)
	}

	if err := validateEnvNamespace(namespaceName); err != nil {
		return nil, err
	}

	if err := ensureEnvNamespaceAvailable(req.ClusterID, namespaceName, ""); err != nil {
		return nil, err
	}

	env := &entities.Env{
		Base:             entities.Base{ID: uuid.New()},
		Slug:             req.Slug,
		Name:             req.Name,
		Description:      req.Description,
		ProjectID:        projectID,
		ClusterID:        req.ClusterID,
		ClusterNamespace: namespaceName,
		IsBuildEnv:       req.IsBuildEnv,
	}

	if req.IsBuildEnv {
		// Clear existing build env in same project
		db.DB.Model(&entities.Env{}).Where("project_id = ? AND is_build_env = ?", projectID, true).
			Update("is_build_env", false)
	}

	cluster, err := GetCluster(req.ClusterID)
	if err != nil {
		return nil, err
	}

	envCtx := &models.EnvContext{
		Env:     *env,
		Project: project,
		Cluster: *cluster,
	}

	if err := core.CreateNamespace(context.Background(), req.ClusterID, namespaceName, envCtx); err != nil {
		return nil, err
	}

	if err := db.DB.Create(env).Error; err != nil {
		return nil, err
	}

	if err := CreateDefaultEnvResourceQuota(env.ID); err != nil {
		return nil, err
	}

	// If the cluster has Gateway API CRDs, create the env-level Gateway resource.
	// Failure is non-fatal — the env is created either way.
	if gwErr := tryEnsureEnvGateway(context.Background(), envCtx); gwErr != nil {
		_ = gwErr // best-effort
	}

	return env, nil
}

func validateEnvNamespace(namespace string) error {
	if namespace == "" {
		return errors.New("namespace is required")
	}

	if validationErrors := validation.IsDNS1123Label(namespace); len(validationErrors) > 0 {
		return fmt.Errorf("namespace %q is invalid: %s", namespace, strings.Join(validationErrors, ", "))
	}

	return nil
}

func ensureEnvNamespaceAvailable(clusterID, namespaceName, excludeEnvID string) error {
	res, err := checkEnvNamespaceAvailability(clusterID, namespaceName, excludeEnvID)
	if err != nil {
		return err
	}
	if !res.Available {
		return errors.New(res.Message)
	}

	return nil
}

func CheckEnvNamespaceAvailability(clusterID, namespaceName string) (*models.EnvNamespaceAvailabilityResponse, error) {
	return checkEnvNamespaceAvailability(clusterID, namespaceName, "")
}

func checkEnvNamespaceAvailability(clusterID, namespaceName, excludeEnvID string) (*models.EnvNamespaceAvailabilityResponse, error) {
	namespaceName = strings.TrimSpace(namespaceName)

	if err := validateEnvNamespace(namespaceName); err != nil {
		return &models.EnvNamespaceAvailabilityResponse{
			Available: false,
			Source:    "invalid",
			Message:   err.Error(),
		}, nil
	}

	var existing entities.Env
	query := db.DB.Where("cluster_id = ? AND cluster_namespace = ?", clusterID, namespaceName)
	if excludeEnvID != "" {
		query = query.Where("id != ?", excludeEnvID)
	}

	err := query.First(&existing).Error
	if err == nil {
		return &models.EnvNamespaceAvailabilityResponse{
			Available: false,
			Source:    "database",
			Message:   fmt.Sprintf("namespace %q is already used by another environment in this cluster", namespaceName),
		}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	exists, err := core.NamespaceExists(context.Background(), clusterID, namespaceName)
	if err != nil {
		return nil, err
	}
	if exists {
		return &models.EnvNamespaceAvailabilityResponse{
			Available: false,
			Source:    "cluster",
			Message:   fmt.Sprintf("namespace %q already exists in the selected cluster", namespaceName),
		}, nil
	}

	return &models.EnvNamespaceAvailabilityResponse{
		Available: true,
		Source:    "available",
		Message:   fmt.Sprintf("namespace %q is available", namespaceName),
	}, nil
}

func GetEnv(envID string) (*entities.Env, error) {
	var env entities.Env
	if err := db.DB.First(&env, "id = ?", envID).Error; err != nil {
		return nil, err
	}
	return &env, nil
}

func GetEnvWithProjectName(envID string) (*models.EnvResponse, error) {
	var resp models.EnvResponse
	if err := db.DB.Model(&entities.Env{}).
		Select("envs.id, envs.slug, envs.name, envs.description, envs.project_id, envs.cluster_id, envs.cluster_namespace, envs.is_build_env, envs.created_at, projects.name as project_name, clusters.name as cluster_name, clusters.connection_status as cluster_connection_status, clusters.connection_status_reason as cluster_connection_status_reason").
		Joins("JOIN projects ON projects.id = envs.project_id").
		Joins("JOIN clusters ON clusters.id = envs.cluster_id").
		Where("envs.id = ?", envID).
		First(&resp).Error; err != nil {
		return nil, err
	}

	hasPrometheusIntegration, err := HasPrometheusIntegration(resp.ClusterID)
	if err != nil {
		return nil, err
	}
	resp.HasPrometheusIntegration = hasPrometheusIntegration

	return &resp, nil
}

func GetEnvContext(envID string) (*models.EnvContext, error) {
	env, err := GetEnv(envID)
	if err != nil {
		return nil, err
	}

	project, err := GetProject(env.ProjectID)
	if err != nil {
		return nil, err
	}

	cluster, err := GetCluster(env.ClusterID)
	if err != nil {
		return nil, err
	}

	return &models.EnvContext{
		Env:     *env,
		Project: *project,
		Cluster: *cluster,
	}, nil
}

func UpdateEnv(envID string, req *models.CreateEnvRequest) (*entities.Env, error) {
	env, err := GetEnv(envID)
	if err != nil {
		return nil, err
	}

	if env.Slug != req.Slug {
		var existing entities.Env
		if err := db.DB.Where("project_id = ? AND slug = ? AND id != ?", env.ProjectID, req.Slug, envID).First(&existing).Error; err == nil {
			return nil, errors.New("environment with this slug already exists in this project")
		}
	}

	env.Slug = req.Slug
	env.Name = req.Name
	env.Description = req.Description
	env.ClusterID = req.ClusterID
	env.ClusterNamespace = req.ClusterNamespace

	if err := db.DB.Save(env).Error; err != nil {
		return nil, err
	}
	return env, nil
}

func UpdateEnvBasic(envID string, req *models.UpdateBasicInfoRequest) (*entities.Env, error) {
	env, err := GetEnv(envID)
	if err != nil {
		return nil, err
	}

	env.Name = req.Name
	env.Description = req.Description

	if err := db.DB.Save(env).Error; err != nil {
		return nil, err
	}
	return env, nil
}

func DeleteEnv(envID string) error {
	var appCount int64
	if err := db.DB.Model(&entities.App{}).Where("env_id = ?", envID).Count(&appCount).Error; err != nil {
		return err
	}

	if appCount > 0 {
		return errors.New("cannot delete environment: it contains applications. Please delete all applications first or move them to recycle bin")
	}

	return db.DB.Delete(&entities.Env{}, "id = ?", envID).Error
}

func PermanentlyDeleteEnv(envID string) error {
	var env entities.Env
	if err := db.DB.Unscoped().First(&env, "id = ?", envID).Error; err != nil {
		return err
	}

	// Get all soft-deleted apps in this environment
	var deletedApps []entities.App
	if err := db.DB.Unscoped().Where("env_id = ? AND deleted_at IS NOT NULL", envID).Find(&deletedApps).Error; err != nil {
		return err
	}

	// Permanently delete all soft-deleted apps
	for _, app := range deletedApps {
		if err := PermanentlyDeleteApp(context.Background(), app.ID); err != nil {
			return err
		}
	}

	// Delete environment-level certificates
	var certs []entities.Certificate
	if err := db.DB.Unscoped().Where("env_id = ? AND scope = ?", envID, "env").Find(&certs).Error; err != nil {
		return err
	}
	for _, cert := range certs {
		if err := db.DB.Unscoped().Delete(&cert).Error; err != nil {
			return err
		}
	}

	// Delete the namespace in the cluster (if not already deleted during soft delete)
	if env.ClusterID != "" && env.ClusterNamespace != "" {
		if err := core.DeleteNamespace(context.Background(), env.ClusterID, env.ClusterNamespace); err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return db.DB.Unscoped().Delete(&entities.Env{}, "id = ?", envID).Error
}

func RestoreEnv(envID string) error {
	return db.DB.Unscoped().Model(&entities.Env{}).Where("id = ?", envID).Update("deleted_at", nil).Error
}

func CheckEnvDeletionConflicts(envID string) ([]entities.App, error) {
	var apps []entities.App
	if err := db.DB.Unscoped().Where("env_id = ? AND deleted_at IS NOT NULL", envID).Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

func SetBuildEnv(envID string) (*entities.Env, error) {
	env, err := GetEnv(envID)
	if err != nil {
		return nil, err
	}

	// Clear existing build env in same project
	if err := db.DB.Model(&entities.Env{}).
		Where("project_id = ? AND is_build_env = ?", env.ProjectID, true).
		Update("is_build_env", false).Error; err != nil {
		return nil, err
	}

	env.IsBuildEnv = true
	if err := db.DB.Model(env).Update("is_build_env", true).Error; err != nil {
		return nil, err
	}

	return env, nil
}

func UnsetBuildEnv(envID string) (*entities.Env, error) {
	env, err := GetEnv(envID)
	if err != nil {
		return nil, err
	}

	env.IsBuildEnv = false
	if err := db.DB.Model(env).Update("is_build_env", false).Error; err != nil {
		return nil, err
	}

	return env, nil
}

func GetProjectBuildEnv(projectID string) (*entities.Env, error) {
	var env entities.Env
	if err := db.DB.
		Where("project_id = ? AND is_build_env = ?", projectID, true).
		First(&env).Error; err != nil {
		return nil, err
	}
	return &env, nil
}

func ToEnvResponse(e *entities.Env) models.EnvResponse {
	return models.EnvResponse{
		ID:               e.ID,
		Slug:             e.Slug,
		Name:             e.Name,
		Description:      e.Description,
		ProjectID:        e.ProjectID,
		ClusterID:        e.ClusterID,
		ClusterNamespace: e.ClusterNamespace,
		IsBuildEnv:       e.IsBuildEnv,
		CreatedAt:        e.CreatedAt,
	}
}

// tryEnsureEnvGateway loads the env's certificates and calls core.EnsureEnvGateway.
// It is best-effort: errors are returned but must not block env lifecycle operations.
func tryEnsureEnvGateway(ctx context.Context, envCtx *models.EnvContext) error {
	var certs []entities.Certificate
	db.DB.Where("env_id = ? AND scope = ?", envCtx.Env.ID, "env").Find(&certs)
	return core.EnsureEnvGateway(ctx, envCtx, certs)
}
