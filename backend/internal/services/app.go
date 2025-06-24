package services

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entity"
	"github.com/ketches/ketches/internal/db/orm"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/utils"
	"github.com/ketches/ketches/pkg/websocket"
	"github.com/spf13/cast"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/remotecommand"
)

type AppService interface {
	ListApps(ctx context.Context, req *models.ListAppsRequest) (*models.ListAppsResponse, app.Error)
	AllAppRefs(ctx context.Context, req *models.AllAppRefsRequest) ([]*models.AppRef, app.Error)
	GetApp(ctx context.Context, req *models.GetAppRequest) (*models.AppModel, app.Error)
	GetAppRef(ctx context.Context, req *models.GetAppRefRequest) (*models.AppRef, app.Error)
	CreateApp(ctx context.Context, req *models.CreateAppRequest) (*models.AppModel, app.Error)
	UpdateApp(ctx context.Context, req *models.UpdateAppRequest) (*models.AppModel, app.Error)
	DeleteApp(ctx context.Context, req *models.DeleteAppRequest) app.Error
	UpdateAppImage(ctx context.Context, req *models.UpdateAppImageRequest) (*models.AppModel, app.Error)
	AppAction(ctx context.Context, req *models.AppActionRequest) (*models.AppModel, app.Error)
	ListAppInstances(ctx context.Context, req *models.ListAppInstancesRequest) (*models.ListAppInstancesResponse, app.Error)
	TerminateAppInstance(ctx context.Context, req *models.TerminateAppInstanceRequest) app.Error
	ViewAppContainerLogs(ctx context.Context, req *models.ViewAppContainerLogsRequest) app.Error
	ExecAppContainerTerminal(ctx context.Context, req *models.ExecAppContainerTerminalRequest) app.Error
}

type appService struct {
	Service
}

var appServiceInstance = &appService{
	Service: LoadService(),
}

func NewAppService() AppService {
	return appServiceInstance
}

func (s *appService) ListApps(ctx context.Context, req *models.ListAppsRequest) (*models.ListAppsResponse, app.Error) {
	query := db.Instance().Model(&entity.App{})
	if len(req.ProjectID) > 0 {
		if _, err := s.CheckProjectPermissions(ctx, req.ProjectID); err != nil {
			return nil, err
		}

		query = query.Where("project_id = ?", req.ProjectID)
	}

	if len(req.EnvID) > 0 {
		env, err := orm.GetEnvByID(ctx, req.EnvID)
		if err != nil {
			return nil, err
		}
		if _, err := s.CheckProjectPermissions(ctx, env.ProjectID); err != nil {
			return nil, err
		}

		query = query.Where("env_id = ?", req.EnvID)
	}

	if req.Query != "" {
		query = db.CaseInsensitiveLike(query, req.Query, "slug", "display_name")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("failed to count apps: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	apps := []*entity.App{}
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

func (s *appService) AllAppRefs(ctx context.Context, req *models.AllAppRefsRequest) ([]*models.AppRef, app.Error) {
	projectID, err := orm.GetProjectIDByEnvID(ctx, req.EnvID)
	if err != nil {
		return nil, err
	}

	if _, err := s.CheckProjectPermissions(ctx, projectID); err != nil {
		return nil, err
	}

	refs := []*models.AppRef{}
	if err := db.Instance().Model(&entity.App{}).Where("env_id = ?", req.EnvID).Find(&refs).Error; err != nil {
		log.Printf("failed to list app refs: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return refs, nil
}

func (s *appService) GetApp(ctx context.Context, req *models.GetAppRequest) (*models.AppModel, app.Error) {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	if _, err := s.CheckProjectPermissions(ctx, appEntity.ProjectID); err != nil {
		return nil, err
	}

	result := &models.AppModel{
		AppID:            appEntity.ID,
		Slug:             appEntity.Slug,
		DisplayName:      appEntity.DisplayName,
		Description:      appEntity.Description,
		WorkloadType:     appEntity.WorkloadType,
		Replicas:         appEntity.Replicas,
		RequestCPU:       appEntity.RequestCPU,
		RequestMemory:    appEntity.RequestMemory,
		LimitCPU:         appEntity.LimitCPU,
		LimitMemory:      appEntity.LimitMemory,
		ContainerImage:   appEntity.ContainerImage,
		RegistryUsername: appEntity.RegistryUsername,
		RegistryPassword: appEntity.RegistryPassword,
		Deployed:         appEntity.Deployed,
		DeployVersion:    appEntity.DeployVersion,
		EnvID:            appEntity.EnvID,
		ProjectID:        appEntity.ProjectID,
		ClusterID:        appEntity.ClusterID,
		ClusterNamespace: appEntity.ClusterNamespace,
		CreatedAt:        utils.HumanizeTime(appEntity.CreatedAt),
	}
	result.ActualReplicas, result.Status = getAppStatus(ctx, appEntity)
	return result, nil
}

func (s *appService) GetAppRef(ctx context.Context, req *models.GetAppRefRequest) (*models.AppRef, app.Error) {
	result := &models.AppRef{}
	if err := db.Instance().Model(&entity.App{}).First(result, "id = ?", req.AppID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "App not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	if _, err := s.CheckProjectPermissions(ctx, result.ProjectID); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *appService) GetEnvRef(ctx context.Context, envID string) (*models.EnvRef, app.Error) {
	env, err := orm.GetEnvByID(ctx, envID)
	if err != nil {
		return nil, err
	}

	if _, err := s.CheckProjectPermissions(ctx, env.ProjectID); err != nil {
		return nil, err
	}

	return &models.EnvRef{
		EnvID:       env.ID,
		Slug:        env.Slug,
		DisplayName: env.DisplayName,
	}, nil
}

func (s *appService) CreateApp(ctx context.Context, req *models.CreateAppRequest) (*models.AppModel, app.Error) {
	env, err := orm.GetEnvByID(ctx, req.EnvID)
	if err != nil {
		return nil, err
	}

	if projectRole, err := s.CheckProjectPermissions(ctx, env.ProjectID); err != nil {
		return nil, err
	} else if projectRole == app.ProjectRoleViewer {
		return nil, app.ErrPermissionDenied
	}

	appEntity := &entity.App{
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
		AuditBase: entity.AuditBase{
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
		if err := db.Instance().Updates(entity.App{
			UUIDBase:      appEntity.UUIDBase,
			Deployed:      appEntity.Deployed,
			DeployVersion: appEntity.DeployVersion,
			AuditBase: entity.AuditBase{
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

func (s *appService) UpdateApp(ctx context.Context, req *models.UpdateAppRequest) (*models.AppModel, app.Error) {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	if projectRole, err := s.CheckProjectPermissions(ctx, appEntity.ProjectID); err != nil {
		return nil, err
	} else if projectRole == app.ProjectRoleViewer {
		return nil, app.ErrPermissionDenied
	}

	appEntity.DisplayName = req.DisplayName
	appEntity.Description = req.Description

	if err := db.Instance().Model(appEntity).Updates(entity.App{
		DisplayName: appEntity.DisplayName,
		Description: appEntity.Description,
	}).Error; err != nil {
		log.Printf("failed to update app: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	result := &models.AppModel{
		AppID:       appEntity.ID,
		Slug:        appEntity.Slug,
		DisplayName: appEntity.DisplayName,
		Description: appEntity.Description,
	}

	return result, nil
}

func (s *appService) DeleteApp(ctx context.Context, req *models.DeleteAppRequest) app.Error {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return err
	}

	if projectRole, err := s.CheckProjectPermissions(ctx, appEntity.ProjectID); err != nil {
		return err
	} else if projectRole == app.ProjectRoleViewer {
		return app.ErrPermissionDenied
	}

	// 1. delete app in cluster
	if appEntity.Deployed {
		if err := db.Instance().Model(appEntity).Update("deployed", false).Where("id = ?", appEntity.ID).Error; err != nil {
			log.Printf("failed to update app deploy info: %v", err)
			return app.ErrDatabaseOperationFailed
		}

		if err := deleteClusterAppResources(ctx, appEntity); err != nil {
			return err
		}
	}

	// 2. delete app in database
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(appEntity).Error; err != nil {
			log.Printf("failed to delete app %s: %v", req.AppID, err)
			return err
		}

		if err := tx.Delete(&entity.AppEnvVar{}, "app_id = ?", appEntity.ID).Error; err != nil {
			log.Printf("failed to delete app env vars for app %s: %v", req.AppID, err)
			return err
		}

		if err := tx.Delete(&entity.AppPort{}, "app_id = ?", appEntity.ID).Error; err != nil {
			log.Printf("failed to delete app ports for app %s: %v", req.AppID, err)
			return err
		}

		if err := tx.Delete(&entity.AppGateway{}, "app_id = ?", appEntity.ID).Error; err != nil {
			log.Printf("failed to delete app gateways for app %s: %v", req.AppID, err)
			return err
		}

		log.Printf("app %s deleted successfully", req.AppID)
		return nil
	}); err != nil {
		return app.ErrDatabaseOperationFailed
	}

	return nil
}

func (s *appService) UpdateAppImage(ctx context.Context, req *models.UpdateAppImageRequest) (*models.AppModel, app.Error) {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	if req.ContainerImage == appEntity.ContainerImage && req.RegistryUsername == appEntity.RegistryUsername && req.RegistryPassword == appEntity.RegistryPassword {
		return nil, app.NewError(http.StatusBadRequest, "No changes detected in app image or registry credentials")
	}

	if projectRole, err := s.CheckProjectPermissions(ctx, appEntity.ProjectID); err != nil {
		return nil, err
	} else if projectRole == app.ProjectRoleViewer {
		return nil, app.ErrPermissionDenied
	}

	appEntity.ContainerImage = req.ContainerImage
	appEntity.RegistryUsername = req.RegistryUsername
	appEntity.RegistryPassword = req.RegistryPassword

	if err := db.Instance().Model(appEntity).Updates(entity.App{
		ContainerImage:   appEntity.ContainerImage,
		RegistryUsername: appEntity.RegistryUsername,
		RegistryPassword: appEntity.RegistryPassword,
	}).Error; err != nil {
		log.Printf("failed to update app image: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	if appEntity.Deployed {
		switch appEntity.WorkloadType {
		case app.WorkloadTypeDeployment:
			deployment, err := kube.GetDeployment(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
			if err != nil {
				if err.Code() != http.StatusNotFound {
					return nil, err
				}
				// Deployment not found, we need to redeploy the app
				if err := s.deployApp(ctx, appEntity, true); err != nil {
					return nil, err
				}
			} else {
				for i := range deployment.Spec.Template.Spec.Containers {
					if deployment.Spec.Template.Spec.Containers[i].Name == mainContainerName(appEntity.Slug) {
						deployment.Spec.Template.Spec.Containers[i].Image = appEntity.ContainerImage
					}
				}
				if _, err := kube.UpdateDeployment(ctx, appEntity.ClusterID, deployment); err != nil {
					return nil, err
				}
			}
		case app.WorkloadTypeStatefulSet:
			statefulSet, err := kube.GetStatefulSet(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
			if err != nil {
				if err.Code() != http.StatusNotFound {
					return nil, err
				}
				// StatefulSet not found, we need to redeploy the app
				if err := s.deployApp(ctx, appEntity, true); err != nil {
					return nil, err
				}
			} else {
				for i := range statefulSet.Spec.Template.Spec.Containers {
					if statefulSet.Spec.Template.Spec.Containers[i].Name == mainContainerName(appEntity.Slug) {
						statefulSet.Spec.Template.Spec.Containers[i].Image = appEntity.ContainerImage
					}
				}
				if _, err := kube.UpdateStatefulSet(ctx, appEntity.ClusterID, statefulSet); err != nil {
					return nil, err
				}
			}
		}
	}

	result := &models.AppModel{
		AppID:            appEntity.ID,
		Slug:             appEntity.Slug,
		DisplayName:      appEntity.DisplayName,
		Description:      appEntity.Description,
		ContainerImage:   appEntity.ContainerImage,
		RegistryUsername: appEntity.RegistryUsername,
		RegistryPassword: appEntity.RegistryPassword,
		EnvID:            appEntity.EnvID,
		Deployed:         appEntity.Deployed,
		ProjectID:        appEntity.ProjectID,
	}

	return result, nil
}

func (s *appService) AppAction(ctx context.Context, req *models.AppActionRequest) (*models.AppModel, app.Error) {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	if projectRole, err := s.CheckProjectPermissions(ctx, appEntity.ProjectID); err != nil {
		return nil, err
	} else if projectRole == app.ProjectRoleViewer {
		return nil, app.ErrPermissionDenied
	}

	// TODO: execute action based on req.Action
	switch req.Action {
	case app.AppActionDeploy:
		err = s.deployApp(ctx, appEntity, true)
	case app.AppActionStart:
		err = s.startApp(ctx, appEntity)
	case app.AppActionStop:
		err = s.stopApp(ctx, appEntity)
	case app.AppActionRollingUpdate:
		err = s.rollingUpdateApp(ctx, appEntity)
	case app.AppActionRollback:
		// TODO: Implement rollback logic
	case app.AppActionRedeploy:
		s.redeployApp(ctx, appEntity, true)
	default:
		return nil, app.NewError(http.StatusBadRequest, "Unknown app action")
	}

	if err != nil {
		return nil, err
	}

	result := &models.AppModel{
		AppID: appEntity.ID,
		Slug:  appEntity.Slug,
	}

	return result, nil
}

func (s *appService) ListAppInstances(ctx context.Context, req *models.ListAppInstancesRequest) (*models.ListAppInstancesResponse, app.Error) {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}
	if projectRole, err := s.CheckProjectPermissions(ctx, appEntity.ProjectID); err != nil {
		return nil, err
	} else if projectRole == app.ProjectRoleViewer {
		return nil, app.ErrPermissionDenied
	}

	pods, err := kube.ListPods(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
	if err != nil {
		return nil, err
	}

	result := &models.ListAppInstancesResponse{
		DeployVersion: appEntity.DeployVersion,
		Instances:     make([]*models.AppInstanceModel, 0, len(pods)),
	}
	for _, pod := range pods {
		mainContainer := mainContainer(appEntity.Slug, pod)
		containers := make([]*models.AppInstanceContainerModel, 0, len(pod.Spec.Containers))
		initContainers := make([]*models.AppInstanceContainerModel, 0, len(pod.Spec.InitContainers))
		for _, container := range pod.Status.ContainerStatuses {
			containers = append(containers, &models.AppInstanceContainerModel{
				ContainerName: container.Name,
				Image:         container.Image,
				Status:        kube.GetContainerStatus(&container),
			})
		}
		// let mainContainer always be the first in the list
		sort.Slice(containers, func(i, j int) bool {
			return containers[i].ContainerName == mainContainer.Name
		})
		for _, container := range pod.Status.InitContainerStatuses {
			initContainers = append(initContainers, &models.AppInstanceContainerModel{
				ContainerName: container.Name,
				Image:         container.Image,
				Status:        kube.GetContainerStatus(&container),
			})
		}
		instance := &models.AppInstanceModel{
			AppID:           appEntity.ID,
			InstanceName:    pod.Name,
			Status:          string(pod.Status.Phase),
			RunningDuration: utils.HumanizeTime(pod.CreationTimestamp.Time),
			InstanceIP:      pod.Status.PodIP,
			Containers:      containers,
			InitContainers:  initContainers,
			NodeName:        pod.Spec.NodeName,
			NodeIP:          pod.Status.HostIP,
			ContainerCount:  len(pod.Status.ContainerStatuses),
			RequestCPU:      mainContainer.Resources.Requests.Cpu().String(),
			RequestMemory:   mainContainer.Resources.Requests.Memory().String(),
			LimitCPU:        mainContainer.Resources.Limits.Cpu().String(),
			LimitMemory:     mainContainer.Resources.Limits.Memory().String(),
			DeployVersion:   pod.Labels["ketches/deploy-version"],
		}
		result.Instances = append(result.Instances, instance)
	}
	return result, nil
}

func (s *appService) TerminateAppInstance(ctx context.Context, req *models.TerminateAppInstanceRequest) app.Error {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return err
	}

	if projectRole, err := s.CheckProjectPermissions(ctx, appEntity.ProjectID); err != nil {
		return err
	} else if projectRole == app.ProjectRoleViewer {
		return app.ErrPermissionDenied
	}

	if err := kube.DeletePod(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, req.InstanceName); err != nil {
		return err
	}

	return nil
}

func (s *appService) ViewAppContainerLogs(ctx context.Context, req *models.ViewAppContainerLogsRequest) app.Error {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return err
	}

	if projectRole, err := s.CheckProjectPermissions(ctx, appEntity.ProjectID); err != nil {
		return err
	} else if projectRole == app.ProjectRoleViewer {
		return app.ErrPermissionDenied
	}

	w := req.ResponseWriter
	r := req.Request
	flusher, ok := w.(http.Flusher)
	if !ok {
		return app.NewError(http.StatusInternalServerError, "Streaming unsupported")
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if req.TailLines <= 0 {
		req.TailLines = 1000 // Default to last 1000 lines if not specified
	}

	clientset, err := kube.ClusterClientset(ctx, appEntity.ClusterID, false)
	if err != nil {
		return err
	}

	logsReq := clientset.CoreV1().Pods(appEntity.ClusterNamespace).GetLogs(req.InstanceName, &corev1.PodLogOptions{
		Container:  req.ContainerName,  // container name
		Follow:     req.Follow,         // follow logs
		TailLines:  &req.TailLines,     // fetch last N lines
		Timestamps: req.ShowTimestamps, // show timestamps
		// SinceSeconds: &req.SinceSeconds,
		// SinceTime: &metav1.Time{
		// 	Time: req.SinceTime, // fetch logs since a specific time
		// }, // fetch logs since a specific time
		// Previous:   req.Previous,         // fetch previous logs
		// LimitBytes: utils.Ptr(req.Limit), // limit log size
	})

	stream, e := logsReq.Stream(r.Context())
	if e != nil {
		log.Println("Error streaming pod logs:", e)
		return app.NewError(http.StatusInternalServerError, "Failed to stream pod logs")
	}
	defer stream.Close()

	scaner := bufio.NewScanner(stream)

	for {
		select {
		case <-r.Context().Done():
			return nil
		default:
			if scaner.Scan() {
				txt := scaner.Text()
				fmt.Fprintf(w, "data: %s\n\n", txt) // [data: ] is required for SSE
				flusher.Flush()
			}
		}
	}
}

func (s *appService) ExecAppContainerTerminal(ctx context.Context, req *models.ExecAppContainerTerminalRequest) app.Error {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return err
	}

	if projectRole, err := s.CheckProjectPermissions(ctx, appEntity.ProjectID); err != nil {
		return err
	} else if projectRole == app.ProjectRoleViewer {
		return app.ErrPermissionDenied
	}

	w := req.ResponseWriter
	r := req.Request
	conn, err := websocket.NewConn(w, r)
	if err != nil {
		log.Printf("Error upgrading connection to WebSocket: %v", err)
		return app.NewError(http.StatusInternalServerError, "Failed to upgrade connection to WebSocket")
	}
	defer conn.Close()

	clientset, err := kube.ClusterClientset(ctx, appEntity.ClusterID, false)
	if err != nil {
		return err
	}

	restConfig, err := kube.RestConfig(ctx, appEntity.ClusterID)
	if err != nil {
		return err
	}

	execReq := clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(req.InstanceName).
		Namespace(appEntity.ClusterNamespace).
		SubResource("exec").
		Param("container", req.ContainerName).
		Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "true").
		Param("tty", "true").
		Param("command", "sh").
		Param("command", "-c").
		Param("command", "clear; exec $(command -v bash || command -v ash || command -v sh)")

	// 创建SPDY executor
	executor, e := remotecommand.NewSPDYExecutor(restConfig, "POST", execReq.URL())
	if e != nil {
		log.Printf("Error creating SPDY Executor: %v", e)
		return app.NewError(http.StatusInternalServerError, "Failed to create SPDY Executor")
	}

	// 创建远程命令的流选项
	streamOptions := remotecommand.StreamOptions{
		Stdin:  websocket.NewReader(conn),
		Stdout: websocket.NewWriter(conn),
		Stderr: websocket.NewWriter(conn),
		Tty:    true,
	}

	// 执行远程命令
	if e := executor.StreamWithContext(context.Background(), streamOptions); e != nil {
		log.Printf("Error streaming to Pod: %v", e)
		return app.NewError(http.StatusInternalServerError, "Failed to stream to Pod")
	}

	return nil
}

func (s appService) deployApp(ctx context.Context, appEntity *entity.App, start bool) app.Error {
	envVars, err := orm.AllAppEnvVars(appEntity.ID)
	if err != nil {
		return err
	}
	appEntity.Deployed = true
	appEntity.DeployVersion = newDeployVersion()

	switch appEntity.WorkloadType {
	case app.WorkloadTypeDeployment:
		deployment := buildDeployment(appEntity, envVars, start)
		if _, err := kube.CreateDeployment(ctx, appEntity.ClusterID, deployment); err != nil {
			return err
		}

	case app.WorkloadTypeStatefulSet:
		statefulSet := buildStatefulSet(appEntity, envVars, start)
		if _, err := kube.CreateStatefulSet(ctx, appEntity.ClusterID, statefulSet); err != nil {
			return err
		}
	}

	if err := db.Instance().Updates(&entity.App{
		UUIDBase: entity.UUIDBase{
			ID: appEntity.ID,
		},
		Deployed:      appEntity.Deployed,
		DeployVersion: appEntity.DeployVersion,
		AuditBase: entity.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
	}).Error; err != nil {
		log.Printf("failed to update app info on deploy: %v", err)
		return app.ErrDatabaseOperationFailed
	}

	return nil
}

func (s appService) startApp(ctx context.Context, appEntity *entity.App) app.Error {
	switch appEntity.WorkloadType {
	case app.WorkloadTypeDeployment:
		deployment, err := kube.GetDeployment(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
		if err != nil {
			if err.Code() != http.StatusNotFound {
				return err
			}
			// Deployment not found, we need to redeploy the app
			return s.deployApp(ctx, appEntity, true)
		} else {
			deployment.Spec.Replicas = &appEntity.Replicas
			if _, err := kube.UpdateDeployment(ctx, appEntity.ClusterID, deployment); err != nil {
				return err
			}
		}
	case app.WorkloadTypeStatefulSet:
		statefulSet, err := kube.GetStatefulSet(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
		if err != nil {
			if err.Code() != http.StatusNotFound {
				return err
			}
			// StatefulSet not found, we need to redeploy the app
			return s.deployApp(ctx, appEntity, true)
		} else {
			statefulSet.Spec.Replicas = &appEntity.Replicas
			if _, err := kube.UpdateStatefulSet(ctx, appEntity.ClusterID, statefulSet); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s appService) stopApp(ctx context.Context, appEntity *entity.App) app.Error {
	switch appEntity.WorkloadType {
	case app.WorkloadTypeDeployment:
		deployment, err := kube.GetDeployment(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
		if err != nil {
			if err.Code() != http.StatusNotFound {
				return err
			}
			return s.deployApp(ctx, appEntity, false)
		} else {
			deployment.Spec.Replicas = utils.Ptr[int32](0) // Set replicas to 0 to stop the app
			if _, err := kube.UpdateDeployment(ctx, appEntity.ClusterID, deployment); err != nil {
				return err
			}
		}
	case app.WorkloadTypeStatefulSet:
		statefulSet, err := kube.GetStatefulSet(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
		if err != nil {
			if err.Code() != http.StatusNotFound {
				return err
			}
			return s.deployApp(ctx, appEntity, false)
		} else {
			statefulSet.Spec.Replicas = utils.Ptr[int32](0) // Set replicas to 0 to stop the app
			if _, err := kube.UpdateStatefulSet(ctx, appEntity.ClusterID, statefulSet); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s appService) redeployApp(ctx context.Context, appEntity *entity.App, start bool) app.Error {
	// delete existing resources
	if err := deleteClusterAppResources(ctx, appEntity); err != nil {
		return err
	}

	go func() {
		// wait for resources to be deleted
		time.Sleep(5 * time.Second)
		s.deployApp(ctx, appEntity, start)
		// TODO: create associated resources like services, ingress, etc.
	}()

	return nil
}

func (s appService) rollingUpdateApp(ctx context.Context, appEntity *entity.App) app.Error {
	switch appEntity.WorkloadType {
	case app.WorkloadTypeDeployment:
		deployment, err := kube.GetDeployment(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
		if err != nil {
			if err.Code() != http.StatusNotFound {
				return err
			}
			// Deployment not found, we need to redeploy the app
			return s.deployApp(ctx, appEntity, true)
		} else {
			if deployment.Spec.Template.Annotations == nil {
				deployment.Spec.Template.Annotations = make(map[string]string)
			}
			deployment.Spec.Template.Annotations["ketches/rolling-update"] = fmt.Sprintf("%d", time.Now().UnixMilli())
			if _, err := kube.UpdateDeployment(ctx, appEntity.ClusterID, deployment); err != nil {
				return err
			}
		}
	case app.WorkloadTypeStatefulSet:
		statefulSet, err := kube.GetStatefulSet(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
		if err != nil {
			if err.Code() != http.StatusNotFound {
				return err
			}
			// StatefulSet not found, we need to redeploy the app
			return s.deployApp(ctx, appEntity, true)
		} else {
			if statefulSet.Spec.Template.Annotations == nil {
				statefulSet.Spec.Template.Annotations = make(map[string]string)
			}
			statefulSet.Spec.Template.Annotations["ketches/rolling-update"] = fmt.Sprintf("%d", time.Now().UnixMilli())
			if _, err := kube.UpdateStatefulSet(ctx, appEntity.ClusterID, statefulSet); err != nil {
				return err
			}
		}
	}

	return nil
}

func getAppStatus(ctx context.Context, appEntity *entity.App) (int32, string) {
	if !appEntity.Deployed {
		return 0, app.AppStatusUndeployed
	}

	pods, err := kube.ListPods(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
	if err != nil {
		return 0, app.AppStatusUnknown
	}

	if len(pods) == 0 {
		return 0, app.AppStatusStopped
	}

	actualReplicas := int32(len(pods))

	var (
		rollingUpdate       bool
		runningPodCount     int32
		pendingPodCount     int32
		terminatingPodCount int32
	)
	for _, pod := range pods {
		if kube.IsPodAbnormal(pod) {
			return actualReplicas, app.AppStatusAbnormal
		}

		deployVersion := pod.Labels["ketches/deploy-version"]
		if deployVersion != appEntity.DeployVersion {
			rollingUpdate = true
			continue
		}

		if pod.DeletionTimestamp != nil {
			terminatingPodCount++
			continue
		}

		switch pod.Status.Phase {
		case corev1.PodRunning:
			runningPodCount++
		case corev1.PodPending:
			pendingPodCount++
		}
	}

	if rollingUpdate {
		return actualReplicas, app.AppStatusRollingUpdate
	}

	if terminatingPodCount > 0 {
		if runningPodCount == 0 && pendingPodCount == 0 {
			return actualReplicas, app.AppStatusStopped
		} else {
			return actualReplicas, app.AppStatusStopping
		}
	}

	if pendingPodCount > 0 {
		return actualReplicas, app.AppStatusStarting
	}

	if runningPodCount == actualReplicas {
		return actualReplicas, app.AppStatusRunning
	}

	return actualReplicas, app.AppStatusUnknown
}

// buildDeployment creates a Kubernetes Deployment resource based on the app entity and environment variables.
// TODO: 1. Registry credentials should be handled securely, not as plain text.
//  2. Consider adding volume mounts and persistent storage if needed.
func buildDeployment(app *entity.App, appEnvVars []*entity.AppEnvVar, start bool) *appsv1.Deployment {
	envs := make([]corev1.EnvVar, 0, len(appEnvVars))
	for _, envVar := range appEnvVars {
		envs = append(envs, corev1.EnvVar{
			Name:  envVar.Key,
			Value: envVar.Value,
		})
	}

	replicas := app.Replicas
	if !start {
		replicas = 0
	}

	labels := buildWorkloadLabels(app)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Slug,
			Namespace: app.ClusterNamespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            mainContainerName(app.Slug),
							Image:           app.ContainerImage,
							ImagePullPolicy: corev1.PullAlways,
							Env:             envs,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", app.RequestCPU)),
									corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", app.RequestMemory)),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", app.LimitCPU)),
									corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", app.LimitMemory)),
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildStatefulSet(app *entity.App, appEnvVars []*entity.AppEnvVar, start bool) *appsv1.StatefulSet {
	envs := make([]corev1.EnvVar, 0, len(appEnvVars))
	for _, envVar := range appEnvVars {
		envs = append(envs, corev1.EnvVar{
			Name:  envVar.Key,
			Value: envVar.Value,
		})
	}

	replicas := app.Replicas
	if !start {
		replicas = 0
	}

	labels := buildWorkloadLabels(app)

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Slug,
			Namespace: app.ClusterNamespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			// VolumeClaimTemplates: []corev1.PersistentVolumeClaim{},
			ServiceName: app.Slug,
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            mainContainerName(app.Slug),
							Image:           app.ContainerImage,
							ImagePullPolicy: corev1.PullAlways,
							Env:             envs,
							// VolumeMounts: []corev1.VolumeMount{},
						},
					},
					// Volumes: []corev1.Volume{},
				},
			},
			PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
				WhenDeleted: appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
			},
		},
	}
}

func buildWorkloadLabels(app *entity.App) map[string]string {
	return map[string]string{
		"ketches/owned":          "true",
		"ketches/app":            app.Slug,
		"ketches/env":            app.EnvSlug,
		"ketches/project":        app.ProjectSlug,
		"ketches/deploy-version": app.DeployVersion,
	}
}

func deleteClusterAppResources(ctx context.Context, appEntity *entity.App) app.Error {
	switch appEntity.WorkloadType {
	case app.WorkloadTypeDeployment:
		// Delete deployment
		if err := kube.DeleteDeployment(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug); err != nil {
			log.Printf("failed to delete deployment for app %s: %v", appEntity.Slug, err)
			return err
		}
	case app.WorkloadTypeStatefulSet:
		// Delete statefulset
		if err := kube.DeleteStatefulSet(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug); err != nil {
			log.Printf("failed to delete statefulset for app %s: %v", appEntity.Slug, err)
			return err
		}
	}

	// Delete services
	services, err := kube.ListServices(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
	if err != nil {
		return err
	}
	for _, service := range services {
		if err := kube.DeleteService(ctx, appEntity.ClusterID, service.Namespace, service.Name); err != nil {
			log.Printf("failed to delete service for app %s: %v", appEntity.Slug, err)
			return err
		}
	}

	return nil
}

func mainContainer(appSlug string, pod *corev1.Pod) *corev1.Container {
	for _, container := range pod.Spec.Containers {
		if container.Name == mainContainerName(appSlug) {
			return &container
		}
	}
	return nil
}

func mainContainerName(appSlug string) string {
	return fmt.Sprintf("mc-%s", appSlug)
}

func newDeployVersion() string {
	return cast.ToString(time.Now().UnixMilli())
}
