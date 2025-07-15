package services

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/db/orm"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayapisv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type EnvService interface {
	ListEnvs(ctx context.Context, req *models.ListEnvsRequest) (*models.ListEnvsResponse, app.Error)
	AllEnvRefs(ctx context.Context, req *models.AllEnvRefsRequest) ([]*models.EnvRef, app.Error)
	CreateEnv(ctx context.Context, req *models.CreateEnvRequest) (*models.EnvModel, app.Error)
	GetEnv(ctx context.Context, req *models.GetEnvRequest) (*models.EnvModel, app.Error)
	GetEnvRef(ctx context.Context, req *models.GetEnvRefRequest) (*models.EnvRef, app.Error)
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
	query := db.Instance().Model(&entities.Env{})
	if len(req.ProjectID) > 0 {
		// Permission check is now handled by middleware when accessing specific project resources
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

	envs := []*entities.Env{}
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
	refs := []*models.EnvRef{}
	if err := db.Instance().Model(&entities.Env{}).Where("project_id = ?", req.ProjectID).Find(&refs).Error; err != nil {
		log.Printf("failed to list env refs: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return refs, nil
}

func (s *envService) CreateEnv(ctx context.Context, req *models.CreateEnvRequest) (*models.EnvModel, app.Error) {
	projectSlug, err := orm.GetProjectSlugByID(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}

	clusterSlug, err := orm.GetClusterSlugByID(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	clusterGatewayIP, err := orm.GetClusterGatewayIPByID(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	env := &entities.Env{
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

	kcli, err := kube.ClusterRuntimeClient(ctx, env.ClusterID)
	if err != nil {
		log.Printf("failed to get cluster runtime client for env %s: %v", env.ID, err)
		return nil, app.NewError(http.StatusInternalServerError, "Failed to get cluster runtime client")
	}

	if installed, _ := core.CheckGatewayAPIInstalled(ctx, kcli); installed {
		core.ApplyResource(ctx, kcli, &gatewayapisv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      env.ClusterNamespace,
				Namespace: env.ClusterNamespace,
				Labels: map[string]string{
					"ketches.cn/owned":     "true",
					"ketches.cn/env":       env.Slug,
					"ketches.cn/envID":     env.ID,
					"ketches.cn/project":   env.ProjectSlug,
					"ketches.cn/projectID": env.ProjectID,
				},
			},
			Spec: gatewayapisv1.GatewaySpec{
				GatewayClassName: gatewayapisv1.ObjectName("ketches"),
				Addresses: []gatewayapisv1.GatewaySpecAddress{
					{
						Type:  utils.Ptr(gatewayapisv1.IPAddressType),
						Value: clusterGatewayIP,
					},
				},
				Listeners: []gatewayapisv1.Listener{
					{
						AllowedRoutes: &gatewayapisv1.AllowedRoutes{
							Kinds: []gatewayapisv1.RouteGroupKind{
								{
									Group: utils.Ptr(gatewayapisv1.Group(gatewayapisv1.GroupName)),
									Kind:  "HTTPRoute",
								},
							},
							Namespaces: &gatewayapisv1.RouteNamespaces{
								From: utils.Ptr(gatewayapisv1.NamespacesFromSame),
							},
						},
						Name:     gatewayapisv1.SectionName("http"),
						Port:     80,
						Protocol: gatewayapisv1.HTTPProtocolType,
					},
				},
			},
		})
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

func (s *envService) GetEnv(ctx context.Context, req *models.GetEnvRequest) (*models.EnvModel, app.Error) {
	env := new(entities.Env)
	if err := db.Instance().First(env, "id = ?", req.EnvID).Error; err != nil {
		log.Printf("failed to get env %s: %v", req.EnvID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Env not found")
		}
		return nil, app.ErrDatabaseOperationFailed
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
	if err := db.Instance().Model(&entities.Env{}).First(result, "id = ?", req.EnvID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Env not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func (s *envService) UpdateEnv(ctx context.Context, req *models.UpdateEnvRequest) (*models.EnvModel, app.Error) {
	env := &entities.Env{}
	if err := db.Instance().First(env, "id = ?", req.EnvID).Error; err != nil {
		log.Printf("failed to find env %s: %v", req.EnvID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Env not found")
		}
		return nil, app.NewError(http.StatusInternalServerError, "Failed to get env")
	}

	env.DisplayName = req.DisplayName
	env.Description = req.Description

	if err := db.Instance().Updates(&entities.Env{
		UUIDBase:    env.UUIDBase,
		DisplayName: env.DisplayName,
		Description: env.Description,
		AuditBase: entities.AuditBase{
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
	env := &entities.Env{}
	if err := db.Instance().First(env, "id = ?", req.EnvID).Error; err != nil {
		log.Printf("failed to find env %s: %v", req.EnvID, err)
		if db.IsErrRecordNotFound(err) {
			return app.NewError(http.StatusNotFound, "Env not found")
		}
		return app.NewError(http.StatusInternalServerError, "Failed to get env")
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

func buildNamespace(env *entities.Env) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: env.ClusterNamespace,
			Labels: map[string]string{
				"ketches.cn/owned":   "true",
				"ketches.cn/env":     env.Slug,
				"ketches.cn/project": env.ProjectSlug,
			},
		},
	}
}
