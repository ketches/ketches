package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/app"
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

	if err := core.CreateNamespace(context.Background(), req.ClusterID, namespaceName, &models.EnvContext{
		Env:     *env,
		Project: project,
	}); err != nil {
		return nil, err
	}

	if err := db.DB.Create(env).Error; err != nil {
		return nil, err
	}

	if err := CreateDefaultEnvResourceQuota(env.ID); err != nil {
		return nil, err
	}

	return env, nil
}

func validateEnvNamespace(namespace string) error {
	if namespace == "" {
		return errors.New("namespace is required")
	}

	if validationErrors := validation.IsDNS1123Label(namespace); len(validationErrors) > 0 {
		return app.NewErrorf("namespace %q is invalid: %s", namespace, strings.Join(validationErrors, ", "))
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

	requestedNamespace := strings.TrimSpace(req.ClusterNamespace)
	if requestedNamespace == "" {
		requestedNamespace = env.ClusterNamespace
	}
	if req.ClusterID != env.ClusterID || requestedNamespace != env.ClusterNamespace {
		return nil, errors.New("changing an environment cluster or namespace requires the dedicated migration workflow")
	}

	env.Slug = req.Slug
	env.Name = req.Name
	env.Description = req.Description

	// Update only mutable environment fields. ClusterNamespace may be NULL in
	// legacy databases, and saving the whole entity would turn that NULL into
	// an empty string when GORM scans it into the string field.
	if err := db.DB.Model(&entities.Env{}).Where("id = ?", env.ID).Updates(map[string]any{
		"slug":        env.Slug,
		"name":        env.Name,
		"description": env.Description,
	}).Error; err != nil {
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

	// Do not use Save here: it writes every column, including a nullable
	// legacy ClusterNamespace that was scanned into an empty string.
	if err := db.DB.Model(&entities.Env{}).Where("id = ?", env.ID).Updates(map[string]any{
		"name":        env.Name,
		"description": env.Description,
	}).Error; err != nil {
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

func PermanentlyDeleteEnv(envID string, actors ...RecycleBinActor) error {
	actor, err := recycleBinActorFromArgs(actors)
	if err != nil {
		return err
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := loadRecycleBinEnv(tx, envID, actor); err != nil {
			return err
		}
		if err := claimRecycleBinDeletionTargets(tx, newRecycleBinDeletionTarget(
			recycleBinResourceEnvironment, envID, &entities.Env{}, "environment",
		)); err != nil {
			return err
		}
		return validateEnvPermanentDeletionTx(tx, envID)
	}); err != nil {
		return err
	}
	if err := cleanupDeletedEnvNamespace(context.Background(), envID); err != nil {
		return err
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := loadRecycleBinEnv(tx, envID, actor); err != nil {
			return err
		}
		return permanentlyDeleteEnvTx(context.Background(), tx, envID)
	})
}

func permanentlyDeleteEnvTx(ctx context.Context, tx *gorm.DB, envID string) error {
	var env entities.Env
	if err := tx.Unscoped().First(&env, "id = ?", envID).Error; err != nil {
		return err
	}
	if !env.DeletedAt.Valid {
		return app.WrapErrorf(ErrRecycleBinResourceActive, "environment %s", envID)
	}

	if err := deleteEnvOwnedRecordsTx(ctx, tx, envID); err != nil {
		return err
	}

	if err := tx.Unscoped().Delete(&entities.Env{}, "id = ?", envID).Error; err != nil {
		return err
	}
	return deleteRecycleBinDeletionClaim(tx, recycleBinResourceEnvironment, envID)
}

func cleanupDeletedEnvNamespace(ctx context.Context, envID string) error {
	var env entities.Env
	if err := db.DB.Unscoped().First(&env, "id = ?", envID).Error; err != nil {
		return err
	}
	if env.ClusterID == "" || env.ClusterNamespace == "" {
		return nil
	}
	err := core.DeleteNamespace(ctx, env.ClusterID, env.ClusterNamespace)
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func RestoreEnv(envID string) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		return restoreEnvTx(tx, envID)
	})
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
