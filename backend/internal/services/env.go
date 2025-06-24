package services

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entity"
	"github.com/ketches/ketches/internal/db/orm"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EnvService interface {
	ListEnvs(ctx context.Context, req *models.ListEnvsRequest) (*models.ListEnvsResponse, app.Error)
	AllEnvRefs(ctx context.Context, req *models.AllEnvRefsRequest) ([]*models.EnvRef, app.Error)
	GetEnv(ctx context.Context, req *models.GetEnvRequest) (*models.EnvModel, app.Error)
	GetEnvRef(ctx context.Context, req *models.GetEnvRefRequest) (*models.EnvRef, app.Error)
	CreateEnv(ctx context.Context, req *models.CreateEnvRequest) (*models.EnvModel, app.Error)
	UpdateEnv(ctx context.Context, req *models.UpdateEnvRequest) (*models.EnvModel, app.Error)
	DeleteEnv(ctx context.Context, req *models.DeleteEnvRequest) app.Error
}

type envService struct {
	Service
}

func NewEnvService() EnvService {
	return &envService{
		Service: LoadService(),
	}
}

func (s *envService) ListEnvs(ctx context.Context, req *models.ListEnvsRequest) (*models.ListEnvsResponse, app.Error) {
	query := db.Instance().Model(&entity.Env{})
	if len(req.ProjectID) > 0 {
		if _, err := s.CheckProjectPermissions(ctx, req.ProjectID); err != nil {
			return nil, err
		}

		query = query.Where("project_id = ?", req.ProjectID)
	}

	if len(req.Query) > 0 {
		query = db.CaseInsensitiveLike(query, req.Query, "slug", "display_name")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("failed to count envs: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	envs := []*entity.Env{}
	if err := req.PagedSQL(query).Find(&envs).Error; err != nil {
		log.Printf("failed to list envs: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	result := &models.ListEnvsResponse{
		Total:   total,
		Records: make([]*models.EnvModel, 0, len(envs)),
	}
	for _, env := range envs {
		result.Records = append(result.Records, &models.EnvModel{
			EnvID:       env.ID,
			Slug:        env.Slug,
			DisplayName: env.DisplayName,
			Description: env.Description,
			ProjectID:   env.ProjectID,
			CreatedAt:   utils.HumanizeTime(env.CreatedAt),
		})
	}

	return result, nil
}

func (s *envService) AllEnvRefs(ctx context.Context, req *models.AllEnvRefsRequest) ([]*models.EnvRef, app.Error) {
	if _, err := s.CheckProjectPermissions(ctx, req.ProjectID); err != nil {
		return nil, err
	}

	refs := []*models.EnvRef{}
	if err := db.Instance().Model(&entity.Env{}).Where("project_id = ?", req.ProjectID).Find(&refs).Error; err != nil {
		log.Printf("failed to list env refs: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return refs, nil
}

func (s *envService) GetEnv(ctx context.Context, req *models.GetEnvRequest) (*models.EnvModel, app.Error) {
	env := new(entity.Env)
	if err := db.Instance().First(env, "id = ?", req.EnvID).Error; err != nil {
		log.Printf("failed to get env %s: %v", req.EnvID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Env not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	if _, err := s.CheckProjectPermissions(ctx, env.ProjectID); err != nil {
		return nil, err
	}

	return &models.EnvModel{
		EnvID:       env.ID,
		Slug:        env.Slug,
		DisplayName: env.DisplayName,
		Description: env.Description,
		ProjectID:   env.ProjectID,
		CreatedAt:   utils.HumanizeTime(env.CreatedAt),
	}, nil
}

func (s *envService) GetEnvRef(ctx context.Context, req *models.GetEnvRefRequest) (*models.EnvRef, app.Error) {
	result := &models.EnvRef{}
	if err := db.Instance().Model(&entity.Env{}).First(result, "id = ?", req.EnvID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Env not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	if _, err := s.CheckProjectPermissions(ctx, result.ProjectID); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *envService) CreateEnv(ctx context.Context, req *models.CreateEnvRequest) (*models.EnvModel, app.Error) {
	projectRole, err := s.CheckProjectPermissions(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	if projectRole != app.ProjectRoleOwner {
		return nil, app.NewError(http.StatusForbidden, "You do not have permission to create env in this project")
	}

	projectSlug, err := orm.GetProjectSlugByID(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}

	clusterSlug, err := orm.GetClusterSlugByID(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	env := &entity.Env{
		Slug:             req.Slug,
		DisplayName:      req.DisplayName,
		Description:      req.Description,
		ProjectID:        req.ProjectID,
		ProjectSlug:      projectSlug,
		ClusterID:        req.ClusterID,
		ClusterSlug:      clusterSlug,
		ClusterNamespace: fmt.Sprintf("%s-%s", projectSlug, req.Slug),
	}

	// Create namespace for the env in the cluster
	if _, err := kube.CreateNamespace(ctx, env.ClusterID, buildNamespace(env)); err != nil {
		return nil, err
	}

	if err := db.Instance().Create(env).Error; err != nil {
		log.Printf("failed to create env: %v", err)
		if db.IsErrDuplicatedKey(err) {
			return nil, app.NewError(http.StatusConflict, "Env with this slug already exists in the project")
		}
		return nil, app.NewError(http.StatusInternalServerError, "Failed to create env")
	}

	return &models.EnvModel{
		EnvID:       env.ID,
		Slug:        env.Slug,
		DisplayName: env.DisplayName,
		Description: env.Description,
		ProjectID:   env.ProjectID,
		ClusterID:   env.ClusterID,
	}, nil
}

func (s *envService) UpdateEnv(ctx context.Context, req *models.UpdateEnvRequest) (*models.EnvModel, app.Error) {
	env := &entity.Env{}
	if err := db.Instance().First(env, "id = ?", req.EnvID).Error; err != nil {
		log.Printf("failed to find env %s: %v", req.EnvID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Env not found")
		}
		return nil, app.NewError(http.StatusInternalServerError, "Failed to get env")
	}

	projectRole, err := s.CheckProjectPermissions(ctx, env.ProjectID)
	if err != nil {
		return nil, err
	}
	if projectRole != app.ProjectRoleOwner {
		return nil, app.NewError(http.StatusForbidden, "You do not have permission to update this env")
	}

	env.DisplayName = req.DisplayName
	env.Description = req.Description

	if err := db.Instance().Updates(&entity.Env{
		UUIDBase:    env.UUIDBase,
		DisplayName: env.DisplayName,
		Description: env.Description,
		AuditBase: entity.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
	}).Error; err != nil {
		log.Printf("failed to update env %s: %v", req.EnvID, err)
		return nil, app.NewError(http.StatusInternalServerError, "Failed to update env")
	}

	return &models.EnvModel{
		EnvID:       env.ID,
		Slug:        env.Slug,
		DisplayName: env.DisplayName,
		Description: env.Description,
		ProjectID:   env.ProjectID,
	}, nil
}

func (s *envService) DeleteEnv(ctx context.Context, req *models.DeleteEnvRequest) app.Error {
	env := &entity.Env{}
	if err := db.Instance().First(env, "id = ?", req.EnvID).Error; err != nil {
		log.Printf("failed to find env %s: %v", req.EnvID, err)
		if db.IsErrRecordNotFound(err) {
			return app.NewError(http.StatusNotFound, "Env not found")
		}
		return app.NewError(http.StatusInternalServerError, "Failed to get env")
	}

	projectRole, err := s.CheckProjectPermissions(ctx, env.ProjectID)
	if err != nil {
		return err
	}
	if projectRole != app.ProjectRoleOwner {
		return app.NewError(http.StatusForbidden, "You do not have permission to delete this env")
	}

	count, err := orm.CountEnvApps(ctx, env.ID)
	if err != nil {
		return err
	}
	if count > 0 {
		return app.NewError(http.StatusConflict, "Cannot delete env with existing apps")
	}

	if err := kube.DeleteNamespace(ctx, env.ClusterID, env.ClusterNamespace); err != nil {
		log.Printf("failed to delete namespace %s: %v", env.ClusterNamespace, err)
		return app.NewError(http.StatusInternalServerError, "Failed to delete env")
	}

	if err := db.Instance().Delete(env).Error; err != nil {
		log.Printf("failed to delete env %s: %v", req.EnvID, err)
		return app.NewError(http.StatusInternalServerError, "Failed to delete env")
	}

	return nil
}

func buildNamespace(env *entity.Env) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: env.ClusterNamespace,
			Labels: map[string]string{
				"ketches/owned":   "true",
				"ketches/env":     env.Slug,
				"ketches/project": env.ProjectSlug,
			},
		},
	}
}
