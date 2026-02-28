package services

import (
	"context"
	"errors"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func ListEnvs(projectID string, page, pageSize int, search string) (int64, []entities.Env, error) {
	var envs []entities.Env
	var total int64
	query := db.DB.Model(&entities.Env{}).Where("project_id = ?", projectID).Order("created_at")
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

func ListEnvsSimple(projectID string) ([]entities.Env, error) {
	var envs []entities.Env
	if err := db.DB.Select("id, slug, name, description, cluster_id, cluster_namespace, is_build_env").Where("project_id = ?", projectID).Order("created_at").Find(&envs).Error; err != nil {
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

	namespaceName := core.GenerateNamespaceName(project.Slug, req.Slug)

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

	if err := core.CreateNamespace(context.Background(), req.ClusterID, namespaceName); err != nil {
		return nil, err
	}

	if err := db.DB.Create(env).Error; err != nil {
		return nil, err
	}

	// If the cluster has Gateway API CRDs, create the env-level Gateway resource.
	// Failure is non-fatal — the env is created either way.
	if gwErr := tryEnsureEnvGateway(context.Background(), env); gwErr != nil {
		_ = gwErr // best-effort
	}

	return env, nil
}

func GetEnv(envID string) (*entities.Env, error) {
	var env entities.Env
	if err := db.DB.Preload("Project").Preload("Cluster").First(&env, "id = ?", envID).Error; err != nil {
		return nil, err
	}
	return &env, nil
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
	if err := db.DB.Unscoped().Preload("Cluster").First(&env, "id = ?", envID).Error; err != nil {
		return err
	}

	// Get all soft-deleted apps in this environment
	var deletedApps []entities.App
	if err := db.DB.Unscoped().Where("env_id = ? AND deleted_at IS NOT NULL", envID).Find(&deletedApps).Error; err != nil {
		return err
	}

	// Permanently delete all soft-deleted apps
	for _, app := range deletedApps {
		if err := PermanentlyDeleteApp(app.ID); err != nil {
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
	if err := db.DB.Preload("Cluster").
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
func tryEnsureEnvGateway(ctx context.Context, env *entities.Env) error {
	var certs []entities.Certificate
	db.DB.Where("env_id = ? AND scope = ?", env.ID, "env").Find(&certs)
	return core.EnsureEnvGateway(ctx, env, certs)
}
