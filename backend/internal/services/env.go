package services

import (
	"context"
	"log"
	"net/http"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/db/orm"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EnvService interface {
	GetEnv(ctx context.Context, req *models.GetEnvRequest) (*models.EnvModel, app.Error)
	GetEnvRef(ctx context.Context, req *models.GetEnvRefRequest) (*models.EnvRef, app.Error)
	UpdateEnv(ctx context.Context, req *models.UpdateEnvRequest) (*models.EnvModel, app.Error)
	DeleteEnv(ctx context.Context, req *models.DeleteEnvRequest) app.Error
	ListApps(ctx context.Context, req *models.ListAppsRequest) (*models.ListAppsResponse, app.Error)
	AllAppRefs(ctx context.Context, req *models.AllAppRefsRequest) ([]*models.AppRef, app.Error)
	CreateApp(ctx context.Context, req *models.CreateAppRequest) (*models.AppModel, app.Error)
}

type envService struct {
	Service
}

func NewEnvService() EnvService {
	return &envService{
		Service: LoadService(),
	}
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

	// Permission check is now handled by middleware

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

	// Permission check is now handled by middleware

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

	// Permission check is now handled by middleware

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

	// Permission check is now handled by middleware

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

func (s *envService) ListApps(ctx context.Context, req *models.ListAppsRequest) (*models.ListAppsResponse, app.Error) {
	query := db.Instance().Model(&entities.App{}).Where("env_id = ?", req.EnvID)

	if req.Query != "" {
		query = db.CaseInsensitiveLike(query, req.Query, "slug", "display_name")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("failed to count apps: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	apps := []*entities.App{}
	if err := req.PagedSQL(query).Find(&apps).Error; err != nil {
		log.Printf("failed to list apps: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	result := &models.ListAppsResponse{
		Total:   total,
		Records: make([]*models.AppModel, 0, len(apps)),
	}
	for _, app := range apps {
		resultItem := &models.AppModel{
			AppID:            app.ID,
			Slug:             app.Slug,
			DisplayName:      app.DisplayName,
			WorkloadType:     app.WorkloadType,
			Replicas:         app.Replicas,
			ContainerImage:   app.ContainerImage,
			Deployed:         app.Deployed,
			DeployVersion:    app.DeployVersion,
			EnvID:            app.EnvID,
			ProjectID:        app.ProjectID,
			ClusterNamespace: app.ClusterNamespace,
			CreatedAt:        utils.HumanizeTime(app.CreatedAt),
		}

		resultItem.ActualReplicas, resultItem.Status = getAppStatus(ctx, app)

		result.Records = append(result.Records, resultItem)
	}

	return result, nil
}

func (s *envService) AllAppRefs(ctx context.Context, req *models.AllAppRefsRequest) ([]*models.AppRef, app.Error) {
	refs := []*models.AppRef{}
	if err := db.Instance().Model(&entities.App{}).Where("env_id = ?", req.EnvID).Find(&refs).Error; err != nil {
		log.Printf("failed to list app refs: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return refs, nil
}

func (s *envService) CreateApp(ctx context.Context, req *models.CreateAppRequest) (*models.AppModel, app.Error) {
	env, err := orm.GetEnvByID(ctx, req.EnvID)
	if err != nil {
		return nil, err
	}

	appEntity := &entities.App{
		Slug:             req.Slug,
		DisplayName:      req.DisplayName,
		Description:      req.Description,
		ContainerImage:   req.ContainerImage,
		WorkloadType:     req.WorkloadType,
		Replicas:         req.Replicas,
		RegistryUsername: req.RegistryUsername,
		RegistryPassword: req.RegistryPassword,
		RequestCPU:       req.RequestCPU,
		RequestMemory:    req.RequestMemory,
		LimitCPU:         req.LimitCPU,
		LimitMemory:      req.LimitMemory,
		EnvID:            req.EnvID,
		EnvSlug:          env.Slug,
		ProjectID:        env.ProjectID,
		ProjectSlug:      env.ProjectSlug,
		ClusterID:        env.ClusterID,
		ClusterSlug:      env.ClusterSlug,
		ClusterNamespace: env.ClusterNamespace,
		AuditBase: entities.AuditBase{
			CreatedBy: api.UserID(ctx),
			UpdatedBy: api.UserID(ctx),
		},
	}

	if err := db.Instance().Create(appEntity).Error; err != nil {
		log.Printf("failed to create app: %v", err)

		if db.IsErrDuplicatedKey(err) {
			return nil, app.NewError(http.StatusConflict, "App with this slug already exists in the env")
		}
		return nil, app.NewError(http.StatusInternalServerError, "Failed to create app")
	}

	result := &models.AppModel{
		AppID:            appEntity.ID,
		Slug:             appEntity.Slug,
		DisplayName:      appEntity.DisplayName,
		Description:      appEntity.Description,
		WorkloadType:     appEntity.WorkloadType,
		Replicas:         appEntity.Replicas,
		ContainerImage:   appEntity.ContainerImage,
		RegistryUsername: appEntity.RegistryUsername,
		RegistryPassword: appEntity.RegistryPassword,
		RequestCPU:       appEntity.RequestCPU,
		RequestMemory:    appEntity.RequestMemory,
		LimitCPU:         appEntity.LimitCPU,
		LimitMemory:      appEntity.LimitMemory,
		EnvID:            appEntity.EnvID,
		EnvSlug:          env.Slug,
		ProjectID:        appEntity.ProjectID,
		ProjectSlug:      env.ProjectSlug,
		ClusterID:        appEntity.ClusterID,
		ClusterSlug:      env.ClusterSlug,
		ClusterNamespace: appEntity.ClusterNamespace,
	}

	if req.Deploy {
		appEntity.Deployed = true
		appEntity.DeployVersion = newDeployVersion()

		// Step 1: deploy app in cluster
		switch appEntity.WorkloadType {
		case app.WorkloadTypeDeployment:
			// For Deployment, we create a Deployment resource in the cluster
			deployment := buildDeployment(appEntity, nil, true)
			if _, err := kube.CreateDeployment(ctx, env.ClusterID, deployment); err != nil {
				log.Printf("failed to create deployment [%s/%s] for app: %v", deployment.Namespace, deployment.Name, err)
				return nil, err
			}
		case app.WorkloadTypeStatefulSet:
			statefulSet := buildStatefulSet(appEntity, nil, true)
			// For StatefulSet, we create a StatefulSet resource in the cluster
			if _, err := kube.CreateStatefulSet(ctx, env.ClusterID, statefulSet); err != nil {
				log.Printf("failed to create statefulset [%s/%s] for app: %v", statefulSet.Namespace, statefulSet.Name, err)
				return nil, err
			}
		}

		// Step 2: update app deploy flag
		if err := db.Instance().Updates(entities.App{
			UUIDBase:      appEntity.UUIDBase,
			Deployed:      appEntity.Deployed,
			DeployVersion: appEntity.DeployVersion,
			AuditBase: entities.AuditBase{
				UpdatedBy: api.UserID(ctx),
			},
		}).Error; err != nil {
			log.Printf("failed to update app deploy info: %v", err)
			return nil, app.ErrDatabaseOperationFailed
		}
		result.Deployed = appEntity.Deployed
		result.DeployVersion = appEntity.DeployVersion
		result.ActualReplicas, result.Status = getAppStatus(ctx, appEntity)
	}

	return result, nil
}

func buildNamespace(env *entities.Env) *corev1.Namespace {
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
