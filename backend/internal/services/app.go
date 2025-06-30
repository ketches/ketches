package services

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/goccy/go-json"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/db/orm"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/utils"
	"github.com/ketches/ketches/pkg/websocket"
	"github.com/spf13/cast"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/remotecommand"
)

type AppService interface {
	ListApps(ctx context.Context, req *models.ListAppsRequest) (*models.ListAppsResponse, app.Error)
	AllAppRefs(ctx context.Context, req *models.AllAppRefsRequest) ([]*models.AppRef, app.Error)
	CreateApp(ctx context.Context, req *models.CreateAppRequest) (*models.AppModel, app.Error)
	GetApp(ctx context.Context, req *models.GetAppRequest) (*models.AppModel, app.Error)
	GetAppRef(ctx context.Context, req *models.GetAppRefRequest) (*models.AppRef, app.Error)
	UpdateApp(ctx context.Context, req *models.UpdateAppRequest) (*models.AppModel, app.Error)
	DeleteApp(ctx context.Context, req *models.DeleteAppRequest) app.Error
	UpdateAppImage(ctx context.Context, req *models.UpdateAppImageRequest) (*models.AppModel, app.Error)
	SetAppCommand(ctx context.Context, req *models.SetAppCommandRequest) (*models.AppModel, app.Error)
	SetAppResource(ctx context.Context, req *models.SetAppResourceRequest) (*models.AppModel, app.Error)
	AppAction(ctx context.Context, req *models.AppActionRequest) (*models.AppModel, app.Error)
	ListAppInstances(ctx context.Context, req *models.ListAppInstancesRequest) (*models.ListAppInstancesResponse, app.Error)
	GetAppRunningInfo(ctx context.Context, req *models.GetAppRunningInfoRequest) app.Error
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
			Edition:          app.Edition,
			EnvID:            app.EnvID,
			ProjectID:        app.ProjectID,
			ClusterNamespace: app.ClusterNamespace,
			CreatedAt:        utils.HumanizeTime(app.CreatedAt),
		}

		appStatus := core.GetAppStatus(ctx, app)
		resultItem.ActualReplicas, resultItem.Status = appStatus.ActualReplicas, appStatus.Status

		result.Records = append(result.Records, resultItem)
	}

	return result, nil
}

func (s *appService) AllAppRefs(ctx context.Context, req *models.AllAppRefsRequest) ([]*models.AppRef, app.Error) {
	refs := []*models.AppRef{}
	if err := db.Instance().Model(&entities.App{}).Where("env_id = ?", req.EnvID).Find(&refs).Error; err != nil {
		log.Printf("failed to list app refs: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return refs, nil
}

func (s *appService) CreateApp(ctx context.Context, req *models.CreateAppRequest) (*models.AppModel, app.Error) {
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
		Edition:          cast.ToString(time.Now().UnixMilli()),
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
		Edition:          appEntity.Edition,
		EnvID:            appEntity.EnvID,
		EnvSlug:          env.Slug,
		ProjectID:        appEntity.ProjectID,
		ProjectSlug:      env.ProjectSlug,
		ClusterID:        appEntity.ClusterID,
		ClusterSlug:      env.ClusterSlug,
		ClusterNamespace: appEntity.ClusterNamespace,
	}

	if req.Deploy {
		if err := s.deployApp(ctx, appEntity, nil); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s *appService) GetApp(ctx context.Context, req *models.GetAppRequest) (*models.AppModel, app.Error) {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
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
		ContainerCommand: appEntity.ContainerCommand,
		Edition:          appEntity.Edition,
		EnvID:            appEntity.EnvID,
		ProjectID:        appEntity.ProjectID,
		ClusterID:        appEntity.ClusterID,
		ClusterNamespace: appEntity.ClusterNamespace,
		CreatedAt:        utils.HumanizeTime(appEntity.CreatedAt),
	}
	appStatus := core.GetAppStatus(ctx, appEntity)
	result.ActualReplicas, result.ActualEdition, result.Status = appStatus.ActualReplicas, appStatus.ActualEdition, appStatus.Status
	return result, nil
}

func (s *appService) GetAppRef(ctx context.Context, req *models.GetAppRefRequest) (*models.AppRef, app.Error) {
	result := &models.AppRef{}
	if err := db.Instance().Model(&entities.App{}).First(result, "id = ?", req.AppID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "App not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func (s *appService) GetEnvRef(ctx context.Context, envID string) (*models.EnvRef, app.Error) {
	env, err := orm.GetEnvByID(ctx, envID)
	if err != nil {
		return nil, err
	}

	return &models.EnvRef{
		EnvID:       env.ID,
		Slug:        env.Slug,
		DisplayName: env.DisplayName,
	}, nil
}

func (s *appService) UpdateApp(ctx context.Context, req *models.UpdateAppRequest) (*models.AppModel, app.Error) {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	appEntity.DisplayName = req.DisplayName
	appEntity.Description = req.Description

	if err := db.Instance().Model(appEntity).Select(
		"DisplayName", "Description", "UpdatedBy",
	).Updates(entities.App{
		DisplayName: appEntity.DisplayName,
		Description: appEntity.Description,
		AuditBase: entities.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
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

	return s.deleteApp(ctx, appEntity)
}

func (s *appService) UpdateAppImage(ctx context.Context, req *models.UpdateAppImageRequest) (*models.AppModel, app.Error) {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	if req.ContainerImage == appEntity.ContainerImage && req.RegistryUsername == appEntity.RegistryUsername && req.RegistryPassword == appEntity.RegistryPassword {
		return nil, app.NewError(http.StatusBadRequest, "No changes detected in app image or registry credentials")
	}

	result := &models.AppModel{
		AppID:            appEntity.ID,
		Slug:             appEntity.Slug,
		DisplayName:      appEntity.DisplayName,
		Description:      appEntity.Description,
		ContainerImage:   req.ContainerImage,
		RegistryUsername: req.RegistryUsername,
		RegistryPassword: req.RegistryPassword,
		Edition:          cast.ToString(time.Now().UnixMilli()),
		EnvID:            appEntity.EnvID,
		ProjectID:        appEntity.ProjectID,
	}

	if err := db.Instance().Model(appEntity).Select(
		"ContainerImage", "RegistryUsername", "RegistryPassword", "Edition", "UpdatedBy",
	).Updates(entities.App{
		ContainerImage:   result.ContainerImage,
		RegistryUsername: result.RegistryUsername,
		RegistryPassword: result.RegistryPassword,
		Edition:          result.Edition,
		AuditBase: entities.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
	}).Error; err != nil {
		log.Printf("failed to update app image: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func (s *appService) SetAppCommand(ctx context.Context, req *models.SetAppCommandRequest) (*models.AppModel, app.Error) {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	if req.ContainerCommand == appEntity.ContainerCommand {
		return nil, app.NewError(http.StatusBadRequest, "No changes detected in app container command")
	}

	result := &models.AppModel{
		AppID:            appEntity.ID,
		Slug:             appEntity.Slug,
		DisplayName:      appEntity.DisplayName,
		Description:      appEntity.Description,
		ContainerCommand: req.ContainerCommand,
		Edition:          cast.ToString(time.Now().UnixMilli()),
		EnvID:            appEntity.EnvID,
		ProjectID:        appEntity.ProjectID,
	}

	if err := db.Instance().Model(appEntity).
		Select("ContainerCommand", "Edition", "UpdatedBy").
		Updates(entities.App{
			ContainerCommand: result.ContainerCommand,
			Edition:          result.Edition,
			AuditBase: entities.AuditBase{
				UpdatedBy: api.UserID(ctx),
			},
		}).Error; err != nil {
		log.Printf("failed to update app command: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func (s *appService) SetAppResource(ctx context.Context, req *models.SetAppResourceRequest) (*models.AppModel, app.Error) {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	if req.Replicas == appEntity.Replicas &&
		req.RequestCPU == appEntity.RequestCPU &&
		req.RequestMemory == appEntity.RequestMemory &&
		req.LimitCPU == appEntity.LimitCPU &&
		req.LimitMemory == appEntity.LimitMemory {
		return nil, app.NewError(http.StatusBadRequest, "No changes detected in app resources")
	}

	result := &models.AppModel{
		AppID:         appEntity.ID,
		Slug:          appEntity.Slug,
		DisplayName:   appEntity.DisplayName,
		Description:   appEntity.Description,
		Replicas:      req.Replicas,
		RequestCPU:    req.RequestCPU,
		RequestMemory: req.RequestMemory,
		LimitCPU:      req.LimitCPU,
		LimitMemory:   req.LimitMemory,
		Edition:       cast.ToString(time.Now().UnixMilli()),
		EnvID:         appEntity.EnvID,
		ProjectID:     appEntity.ProjectID,
	}

	if err := db.Instance().Model(appEntity).
		Select("Replicas", "RequestCPU", "RequestMemory", "LimitCPU", "LimitMemory", "Edition", "UpdatedBy").
		Updates(entities.App{
			Replicas:      result.Replicas,
			RequestCPU:    result.RequestCPU,
			RequestMemory: result.RequestMemory,
			LimitCPU:      result.LimitCPU,
			LimitMemory:   result.LimitMemory,
			Edition:       result.Edition,
			AuditBase: entities.AuditBase{
				UpdatedBy: api.UserID(ctx),
			},
		}).Error; err != nil {
		log.Printf("failed to update app resources: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func (s *appService) AppAction(ctx context.Context, req *models.AppActionRequest) (*models.AppModel, app.Error) {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	switch req.Action {
	case app.AppActionDeploy, app.AppActionStart, app.AppActionUpdate, app.AppActionDebugOff:
		err = s.deployApp(ctx, appEntity, nil)
	case app.AppActionStop:
		err = s.deployApp(ctx, appEntity, &core.AppDeployOption{
			ZeroReplicas: true,
		})
	case app.AppActionRollback:
		// TODO: Implement rollback logic
	case app.AppActionRedeploy:
		err = s.redeployApp(ctx, appEntity)
	case app.AppActionDebug:
		err = s.deployApp(ctx, appEntity, &core.AppDeployOption{
			DebugMode: true,
		})
	case app.AppActionDelete:
		err = s.deleteApp(ctx, appEntity)
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

	pods, err := kube.ListPods(ctx, appEntity.ClusterID, appEntity.ClusterNamespace, appEntity.Slug)
	if err != nil {
		return nil, err
	}

	result := &models.ListAppInstancesResponse{
		AppID:     appEntity.ID,
		Slug:      appEntity.Slug,
		Edition:   appEntity.Edition,
		Instances: make([]*models.AppInstanceModel, 0, len(pods)),
	}
	for _, pod := range pods {
		mainContainer := kube.MainContainer(appEntity.Slug, pod)
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
			InstanceName:    pod.Name,
			Status:          kube.GetPodStatus(pod),
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
			Edition:         pod.Labels["ketches/edition"],
		}
		result.Instances = append(result.Instances, instance)
	}
	return result, nil
}

func (s *appService) GetAppRunningInfo(ctx context.Context, req *models.GetAppRunningInfoRequest) app.Error {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return err
	}

	w := req.ResponseWriter
	// r := req.Request
	flusher, ok := w.(http.Flusher)
	if !ok {
		return app.NewError(http.StatusInternalServerError, "Streaming unsupported")
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	client := &kube.AppInstanceSSEClient{
		Key:     fmt.Sprintf("%s/%s", appEntity.ClusterNamespace, appEntity.Slug),
		Ctx:     ctx,
		MsgChan: make(chan []*models.AppInstanceModel, 1),
	}

	// Register the client
	kube.AppInstanceSSEClients.Lock()
	kube.AppInstanceSSEClients.List[client] = struct{}{}
	kube.AppInstanceSSEClients.Unlock()

	// Initialize the channel with the current instances
	instances := kube.GetPodList(client.Key)
	client.MsgChan <- instances

	go func() {
		for {
			select {
			case instances, ok := <-client.MsgChan:
				if !ok {
					// Channel closed, clean up
					kube.AppInstanceSSEClients.Lock()
					delete(kube.AppInstanceSSEClients.List, client)
					kube.AppInstanceSSEClients.Unlock()
					return
				}

				for _, instance := range instances {
					instance.RunningDuration = utils.HumanizeTime(instance.CreatedAt)
				}

				runningStatus := core.GetAppStatusFromInstances(ctx, instances)
				runningInfo := &models.GetAppRunningInfoResponse{
					AppID:          appEntity.ID,
					Slug:           appEntity.Slug,
					Replicas:       appEntity.Replicas,
					Edition:        appEntity.Edition,
					Status:         runningStatus.Status,
					ActualReplicas: runningStatus.ActualReplicas,
					ActualEdition:  runningStatus.ActualEdition,
					Instances:      instances,
				}
				data, err := json.Marshal(runningInfo)
				if err != nil {
					log.Printf("failed to marshal running info: %v", err)
					continue
				}
				_, _ = fmt.Fprintf(w, "data: %s\n\n", data) // [data: ] is required for SSE
				flusher.Flush()
			case <-ctx.Done():
				// Context cancelled, clean up
				kube.AppInstanceSSEClients.Lock()
				delete(kube.AppInstanceSSEClients.List, client)
				kube.AppInstanceSSEClients.Unlock()
				return
			}
		}
	}()

	<-ctx.Done()

	kube.AppInstanceSSEClients.Lock()
	delete(kube.AppInstanceSSEClients.List, client)
	kube.AppInstanceSSEClients.Unlock()

	return nil
}

func (s *appService) TerminateAppInstance(ctx context.Context, req *models.TerminateAppInstanceRequest) app.Error {
	appEntity, err := orm.GetAppByID(ctx, req.AppID)
	if err != nil {
		return err
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

func (s *appService) deployApp(ctx context.Context, appEntity *entities.App, options *core.AppDeployOption) app.Error {
	cli, err := kube.ClusterRuntimeClient(ctx, appEntity.ClusterID)
	if err != nil {
		return err
	}

	appMetadata, err := core.NewAppMetadataBuilderFromAppEntity(ctx, appEntity).Build()
	if err != nil {
		return err
	}

	if err := appMetadata.Deploy(ctx, cli, options); err != nil {
		return err
	}

	return nil
}

func (s *appService) undeployApp(ctx context.Context, appEntity *entities.App) app.Error {
	cli, err := kube.ClusterRuntimeClient(ctx, appEntity.ClusterID)
	if err != nil {
		return err
	}

	appMetadata, err := core.NewAppMetadataBuilderFromAppEntity(ctx, appEntity).Build()
	if err != nil {
		return err
	}

	if err := appMetadata.Undeploy(ctx, cli); err != nil {
		return err
	}

	if err := db.Instance().Model(appEntity).Updates(entities.App{
		AuditBase: entities.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
	}).Error; err != nil {
		log.Printf("failed to update app deploy info: %v", err)
		return app.ErrDatabaseOperationFailed
	}

	return nil
}

func (s appService) redeployApp(ctx context.Context, appEntity *entities.App) app.Error {
	// Step 1: undeploy the app
	if err := s.undeployApp(ctx, appEntity); err != nil {
		return err
	}

	go func() {
		// wait for resources to be deleted
		time.Sleep(5 * time.Second)
		s.deployApp(ctx, appEntity, nil)
	}()

	return nil
}

func (s *appService) deleteApp(ctx context.Context, appEntity *entities.App) app.Error {
	// Step 1. undeploy the app
	if err := s.undeployApp(ctx, appEntity); err != nil {
		return err
	}

	// Step 2. delete app in database
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(appEntity).Error; err != nil {
			log.Printf("failed to delete app %s: %v", appEntity.ID, err)
			return err
		}

		if err := tx.Delete(&entities.AppEnvVar{}, "app_id = ?", appEntity.ID).Error; err != nil {
			log.Printf("failed to delete app env vars for app %s: %v", appEntity.ID, err)
			return err
		}

		if err := tx.Delete(&entities.AppVolume{}, "app_id = ?", appEntity.ID).Error; err != nil {
			log.Printf("failed to delete app volumes for app %s: %v", appEntity.ID, err)
			return err
		}

		if err := tx.Delete(&entities.AppGateway{}, "app_id = ?", appEntity.ID).Error; err != nil {
			log.Printf("failed to delete app gateways for app %s: %v", appEntity.ID, err)
			return err
		}

		log.Printf("app %s deleted successfully", appEntity.ID)
		return nil
	}); err != nil {
		return app.ErrDatabaseOperationFailed
	}

	return nil
}
