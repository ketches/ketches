package services

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

func ListApps(envID string, page, pageSize int, search string) (int64, []entities.App, error) {
	var apps []entities.App
	var total int64
	query := db.DB.Model(&entities.App{}).Where("env_id = ?", envID)
	if search != "" {
		query = query.Where("name LIKE ? OR slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&apps).Error; err != nil {
		return 0, nil, err
	}
	return total, apps, nil
}

func ListAppsSimple(envID string) ([]entities.App, error) {
	var apps []entities.App
	if err := db.DB.Select("id, slug, name, description, code_repository_id").Where("env_id = ?", envID).Order("name").Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

func CreateApp(envID string, req *models.CreateAppRequest) (*entities.App, error) {
	var existing entities.App
	if err := db.DB.Where("env_id = ? AND slug = ?", envID, req.Slug).First(&existing).Error; err == nil {
		return nil, gorm.ErrDuplicatedKey
	}

	application := &entities.App{
		Base:             entities.Base{ID: uuid.New()},
		Slug:             req.Slug,
		Name:             req.Name,
		Description:      req.Description,
		EnvID:            envID,
		ContainerImage:   req.ContainerImage,
		ContainerCommand: req.ContainerCommand,
		RegistryUsername: req.RegistryUsername,
		RegistryPassword: req.RegistryPassword,
		Replicas:         req.Replicas,
		RequestCPU:       req.RequestCPU,
		RequestMemory:    req.RequestMemory,
		LimitCPU:         req.LimitCPU,
		LimitMemory:      req.LimitMemory,
		AppType:          req.AppType,
		DeployStatus:     "undeployed",
	}

	if req.AutoScaling != nil {
		application.AutoScaling = &entities.AppAutoScaling{
			ID:                      uuid.New(),
			MinReplicas:             req.AutoScaling.MinReplicas,
			MaxReplicas:             req.AutoScaling.MaxReplicas,
			TargetCPUUtilization:    req.AutoScaling.TargetCPUUtilization,
			TargetMemoryUtilization: req.AutoScaling.TargetMemoryUtilization,
		}
	}

	if req.SchedulingRule != nil {
		application.SchedulingRule = &entities.AppSchedulingRule{
			ID:           uuid.New(),
			RuleType:     req.SchedulingRule.RuleType,
			NodeName:     req.SchedulingRule.NodeName,
			NodeSelector: req.SchedulingRule.NodeSelector,
			NodeAffinity: req.SchedulingRule.NodeAffinity,
			Tolerations:  req.SchedulingRule.Tolerations,
		}
	}

	if err := db.DB.Create(application).Error; err != nil {
		return nil, err
	}

	var env entities.Env
	if err := db.DB.Preload("Cluster").First(&env, "id = ?", envID).Error; err != nil {
		return nil, err
	}
	application.Env = env

	if req.Deploy {
		if err := core.ApplyApp(context.Background(), application); err != nil {
			return nil, err
		}
		application.DeployStatus = "deployed"
		db.DB.Model(application).Update("deploy_status", "deployed")
	}

	return application, nil
}

func CreateAppFromCodeRepositoryBuild(envID, slug, name, containerImage, registryUsername, registryPassword string, codeRepositoryID *string) (*entities.App, error) {
	var existing entities.App
	if err := db.DB.Where("env_id = ? AND slug = ?", envID, slug).First(&existing).Error; err == nil {
		return nil, gorm.ErrDuplicatedKey
	}

	application := &entities.App{
		Base:             entities.Base{ID: uuid.New()},
		Slug:             slug,
		Name:             name,
		EnvID:            envID,
		ContainerImage:   containerImage,
		RegistryUsername: registryUsername,
		RegistryPassword: registryPassword,
		Replicas:         1,
		RequestCPU:       100,
		RequestMemory:    128,
		LimitCPU:         1000,
		LimitMemory:      512,
		AppType:          "Deployment",
		DeployStatus:     "undeployed",
		CodeRepositoryID: codeRepositoryID,
	}
	if err := db.DB.Create(application).Error; err != nil {
		return nil, err
	}
	var env entities.Env
	if err := db.DB.Preload("Cluster").First(&env, "id = ?", envID).Error; err != nil {
		return nil, err
	}
	application.Env = env
	return application, nil
}

func GetApp(appID string) (*entities.App, error) {
	var application entities.App
	if err := db.DB.Preload("Env.Cluster").
		Preload("EnvVars").
		Preload("Volumes").
		Preload("ConfigFiles").
		Preload("Probes").
		Preload("Gateways").
		Preload("AppPlugins.Plugin").
		Preload("AutoScaling").
		Preload("SchedulingRule").
		First(&application, "id = ?", appID).Error; err != nil {
		return nil, err
	}
	return &application, nil
}

func ApplyApp(application *entities.App) error {
	return core.ApplyApp(context.Background(), application)
}

func UpdateApp(appID string, req *models.CreateAppRequest) (*entities.App, error) {
	application, err := GetApp(appID)
	if err != nil {
		return nil, err
	}

	if application.Slug != req.Slug {
		var existing entities.App
		if err := db.DB.Where("env_id = ? AND slug = ? AND id != ?", application.EnvID, req.Slug, appID).First(&existing).Error; err == nil {
			return nil, gorm.ErrDuplicatedKey
		}
	}

	appBefore := &entities.App{
		ContainerImage: application.ContainerImage,
		Replicas:       application.Replicas,
		RequestCPU:     application.RequestCPU,
		RequestMemory:  application.RequestMemory,
		LimitCPU:       application.LimitCPU,
		LimitMemory:    application.LimitMemory,
	}

	imageChanged := application.ContainerImage != req.ContainerImage

	application.Slug = req.Slug
	application.Name = req.Name
	application.Description = req.Description
	application.AppType = req.AppType
	application.ContainerImage = req.ContainerImage
	application.ContainerCommand = req.ContainerCommand
	application.RegistryUsername = req.RegistryUsername
	if req.RegistryPassword != "" {
		application.RegistryPassword = req.RegistryPassword
	}
	application.Replicas = req.Replicas
	application.RequestCPU = req.RequestCPU
	application.RequestMemory = req.RequestMemory
	application.LimitCPU = req.LimitCPU
	application.LimitMemory = req.LimitMemory

	if req.AutoScaling != nil {
		if application.AutoScaling == nil {
			application.AutoScaling = &entities.AppAutoScaling{ID: uuid.New(), AppID: application.ID}
		}
		application.AutoScaling.MinReplicas = req.AutoScaling.MinReplicas
		application.AutoScaling.MaxReplicas = req.AutoScaling.MaxReplicas
		application.AutoScaling.TargetCPUUtilization = req.AutoScaling.TargetCPUUtilization
		application.AutoScaling.TargetMemoryUtilization = req.AutoScaling.TargetMemoryUtilization
	} else {
		if application.AutoScaling != nil {
			db.DB.Delete(application.AutoScaling)
			application.AutoScaling = nil
		}
	}

	if req.SchedulingRule != nil {
		if application.SchedulingRule == nil {
			application.SchedulingRule = &entities.AppSchedulingRule{ID: uuid.New(), AppID: application.ID}
		}
		application.SchedulingRule.RuleType = req.SchedulingRule.RuleType
		application.SchedulingRule.NodeName = req.SchedulingRule.NodeName
		application.SchedulingRule.NodeSelector = req.SchedulingRule.NodeSelector
		application.SchedulingRule.NodeAffinity = req.SchedulingRule.NodeAffinity
		application.SchedulingRule.Tolerations = req.SchedulingRule.Tolerations
	} else {
		if application.SchedulingRule != nil {
			db.DB.Delete(application.SchedulingRule)
			application.SchedulingRule = nil
		}
	}

	if err := db.DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(application).Error; err != nil {
		return nil, err
	}

	if err := core.ApplyApp(context.Background(), application); err != nil {
		return nil, err
	}

	if imageChanged {
		_ = RecordDeployment(appBefore, application, "manual", "user", "Image updated via UI", nil)
	}

	return application, nil
}

func UpdateAppBasic(appID string, req *models.UpdateBasicInfoRequest) (*entities.App, error) {
	application, err := GetApp(appID)
	if err != nil {
		return nil, err
	}

	application.Name = req.Name
	application.Description = req.Description

	if err := db.DB.Save(application).Error; err != nil {
		return nil, err
	}

	return application, nil
}

func DeleteApp(appID string) error {
	application, err := GetApp(appID)
	if err != nil {
		return err
	}

	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return err
	}

	ctx := context.Background()
	ns := application.Env.ClusterNamespace

	if application.AppType == "StatefulSet" {
		_ = client.AppsV1().StatefulSets(ns).Delete(ctx, application.Slug, metav1.DeleteOptions{})
	} else {
		_ = client.AppsV1().Deployments(ns).Delete(ctx, application.Slug, metav1.DeleteOptions{})
	}

	_ = client.CoreV1().Services(ns).Delete(ctx, application.Slug, metav1.DeleteOptions{})

	return db.DB.Delete(&entities.App{}, "id = ?", appID).Error
}

func PermanentlyDeleteApp(appID string) error {
	var application entities.App
	if err := db.DB.Unscoped().Preload("EnvVars").
		Preload("Volumes").
		Preload("ConfigFiles").
		Preload("Gateways").
		Preload("Probes").
		Preload("AppPlugins").
		Preload("AutoScaling").
		Preload("SchedulingRule").
		First(&application, "id = ?", appID).Error; err != nil {
		return err
	}

	if len(application.EnvVars) > 0 {
		if err := db.DB.Unscoped().Delete(&application.EnvVars).Error; err != nil {
			return err
		}
	}

	if len(application.Volumes) > 0 {
		if err := db.DB.Unscoped().Delete(&application.Volumes).Error; err != nil {
			return err
		}
	}

	if len(application.ConfigFiles) > 0 {
		if err := db.DB.Unscoped().Delete(&application.ConfigFiles).Error; err != nil {
			return err
		}
	}

	if len(application.Gateways) > 0 {
		if err := db.DB.Unscoped().Delete(&application.Gateways).Error; err != nil {
			return err
		}
	}

	if len(application.Probes) > 0 {
		if err := db.DB.Unscoped().Delete(&application.Probes).Error; err != nil {
			return err
		}
	}

	if len(application.AppPlugins) > 0 {
		if err := db.DB.Unscoped().Delete(&application.AppPlugins).Error; err != nil {
			return err
		}
	}

	if application.AutoScaling != nil {
		if err := db.DB.Unscoped().Delete(application.AutoScaling).Error; err != nil {
			return err
		}
	}

	if application.SchedulingRule != nil {
		if err := db.DB.Unscoped().Delete(application.SchedulingRule).Error; err != nil {
			return err
		}
	}

	return db.DB.Unscoped().Delete(&entities.App{}, "id = ?", appID).Error
}

func RestoreApp(appID string) error {
	return db.DB.Unscoped().Model(&entities.App{}).Where("id = ?", appID).Update("deleted_at", nil).Error
}

func ListAppInstances(appID string) ([]models.AppInstanceResponse, error) {
	application, err := GetApp(appID)
	if err != nil {
		return nil, err
	}

	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	pods, err := client.CoreV1().Pods(application.Env.ClusterNamespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "app=" + application.Slug,
	})
	if err != nil {
		return nil, err
	}

	var res []models.AppInstanceResponse
	for _, p := range pods.Items {
		res = append(res, core.ToAppInstanceResponse(&p))
	}
	return res, nil
}

func ListAppInstanceEvents(application *entities.App, instanceName string) ([]models.AppEventResponse, error) {
	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	eventList, err := client.CoreV1().Events(application.Env.ClusterNamespace).List(context.Background(), metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + instanceName + ",involvedObject.kind=Pod",
	})
	if err != nil {
		return nil, err
	}

	var events []models.AppEventResponse
	for _, e := range eventList.Items {
		events = append(events, models.AppEventResponse{
			Type:      e.Type,
			Reason:    e.Reason,
			Message:   e.Message,
			From:      e.Source.Component,
			Count:     e.Count,
			CreatedAt: e.LastTimestamp.Time,
		})
	}

	return events, nil
}

func DeleteAppInstance(application *entities.App, instanceName string) error {
	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return err
	}

	return client.CoreV1().Pods(application.Env.ClusterNamespace).Delete(context.Background(), instanceName, metav1.DeleteOptions{})
}

func StreamAppLogs(ctx context.Context, application *entities.App, instanceName, containerName string, tailLines int64, timestamps bool) (io.ReadCloser, error) {
	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	podLogOptions := &corev1.PodLogOptions{
		Container:  containerName,
		Follow:     true,
		TailLines:  &tailLines,
		Timestamps: timestamps,
	}

	req := client.CoreV1().Pods(application.Env.ClusterNamespace).GetLogs(instanceName, podLogOptions)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open log stream: %w", err)
	}
	return stream, nil
}

func ExecAppContainer(application *entities.App, instanceName, containerName string, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(application.Env.Cluster.KubeConfig))
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(instanceName).
		Namespace(application.Env.ClusterNamespace).
		SubResource("exec").
		VersionedParams(&metav1.GetOptions{}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}

	_ = executor
	return nil
}

func AppAction(appID string, action app.AppAction) (*entities.App, error) {
	application, err := GetApp(appID)
	if err != nil {
		return nil, err
	}

	currentStatus, err := core.CalculateAppStatus(context.Background(), application)
	if err != nil {
		return nil, err
	}

	_, err = core.ValidateStateTransition(currentStatus, action)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	switch action {
	case app.AppActionDeploy:
		return executeDeployAction(ctx, application)
	case app.AppActionStart:
		return executeStartAction(ctx, application)
	case app.AppActionStop:
		return executeStopAction(ctx, application)
	case app.AppActionUpdate:
		return executeUpdateAction(ctx, application)
	case app.AppActionRedeploy:
		return executeRedeployAction(ctx, application)
	case app.AppActionRollback:
		return executeRollbackAction(ctx, application)
	case app.AppActionDebug:
		return executeDebugAction(ctx, application)
	case app.AppActionDebugOff:
		return executeDebugOffAction(ctx, application)
	case app.AppActionDelete:
		return executeDeleteAction(ctx, application)
	default:
		return nil, errors.New("unsupported action")
	}
}

func executeDeployAction(ctx context.Context, application *entities.App) (*entities.App, error) {
	if err := core.ApplyApp(ctx, application); err != nil {
		return nil, err
	}

	application.DeployStatus = "deployed"
	if err := db.DB.Model(application).Update("deploy_status", "deployed").Error; err != nil {
		return nil, err
	}

	return application, nil
}

func executeStartAction(ctx context.Context, application *entities.App) (*entities.App, error) {
	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	switch application.AppType {
	case "Deployment":
		deployment, err := client.AppsV1().Deployments(application.Env.ClusterNamespace).Get(ctx, application.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		replicas := int32(application.Replicas)
		deployment.Spec.Replicas = &replicas

		if _, err := client.AppsV1().Deployments(application.Env.ClusterNamespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}
	case "StatefulSet":
		statefulSet, err := client.AppsV1().StatefulSets(application.Env.ClusterNamespace).Get(ctx, application.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		replicas := int32(application.Replicas)
		statefulSet.Spec.Replicas = &replicas

		if _, err := client.AppsV1().StatefulSets(application.Env.ClusterNamespace).Update(ctx, statefulSet, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}
	}

	return application, nil
}

func executeStopAction(ctx context.Context, application *entities.App) (*entities.App, error) {
	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	replicas := int32(0)

	switch application.AppType {
	case "Deployment":
		deployment, err := client.AppsV1().Deployments(application.Env.ClusterNamespace).Get(ctx, application.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		deployment.Spec.Replicas = &replicas

		if _, err := client.AppsV1().Deployments(application.Env.ClusterNamespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}
	case "StatefulSet":
		statefulSet, err := client.AppsV1().StatefulSets(application.Env.ClusterNamespace).Get(ctx, application.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		statefulSet.Spec.Replicas = &replicas

		if _, err := client.AppsV1().StatefulSets(application.Env.ClusterNamespace).Update(ctx, statefulSet, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}
	}

	return application, nil
}

func executeUpdateAction(ctx context.Context, application *entities.App) (*entities.App, error) {
	if err := core.ApplyApp(ctx, application); err != nil {
		return nil, err
	}

	return application, nil
}

func executeRedeployAction(ctx context.Context, application *entities.App) (*entities.App, error) {
	return executeDeployAction(ctx, application)
}

func executeRollbackAction(ctx context.Context, application *entities.App) (*entities.App, error) {
	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return nil, err
	}
	// TODO: Implement rollback logic, such as using Deployment's revision history or StatefulSet's update strategy to rollback to previous version
	switch application.AppType {
	case "Deployment":
		if _, err := client.AppsV1().Deployments(application.Env.ClusterNamespace).Get(ctx, application.Slug, metav1.GetOptions{}); err != nil {
			return nil, err
		}
	case "StatefulSet":
		if _, err := client.AppsV1().StatefulSets(application.Env.ClusterNamespace).Get(ctx, application.Slug, metav1.GetOptions{}); err != nil {
			return nil, err
		}
	}

	return application, nil
}

func executeDebugAction(ctx context.Context, application *entities.App) (*entities.App, error) {
	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	ns := application.Env.ClusterNamespace

	switch application.AppType {
	case "Deployment":
		deployment, err := client.AppsV1().Deployments(ns).Get(ctx, application.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if deployment.Spec.Template.Labels == nil {
			deployment.Spec.Template.Labels = make(map[string]string)
		}
		deployment.Spec.Template.Labels["app.ketches.cn/debugging"] = "true"

		appContainerName := "app-" + application.Slug
		for i := range deployment.Spec.Template.Spec.Containers {
			if deployment.Spec.Template.Spec.Containers[i].Name == appContainerName {
				deployment.Spec.Template.Spec.Containers[i].Command = []string{"sh", "-c", "while true; do sleep 3600; done"}
				break
			}
		}

		if _, err := client.AppsV1().Deployments(ns).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}

	case "StatefulSet":
		statefulSet, err := client.AppsV1().StatefulSets(ns).Get(ctx, application.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if statefulSet.Spec.Template.Labels == nil {
			statefulSet.Spec.Template.Labels = make(map[string]string)
		}
		statefulSet.Spec.Template.Labels["app.ketches.cn/debugging"] = "true"

		appContainerName := "app-" + application.Slug
		for i := range statefulSet.Spec.Template.Spec.Containers {
			if statefulSet.Spec.Template.Spec.Containers[i].Name == appContainerName {
				statefulSet.Spec.Template.Spec.Containers[i].Command = []string{"sh", "-c", "while true; do sleep 3600; done"}
				break
			}
		}

		if _, err := client.AppsV1().StatefulSets(ns).Update(ctx, statefulSet, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}
	}

	return application, nil
}

func executeDebugOffAction(ctx context.Context, application *entities.App) (*entities.App, error) {
	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	ns := application.Env.ClusterNamespace

	switch application.AppType {
	case "Deployment":
		deployment, err := client.AppsV1().Deployments(ns).Get(ctx, application.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if deployment.Spec.Template.Labels != nil {
			delete(deployment.Spec.Template.Labels, "app.ketches.cn/debugging")
		}

		appContainerName := "app-" + application.Slug
		for i := range deployment.Spec.Template.Spec.Containers {
			if deployment.Spec.Template.Spec.Containers[i].Name == appContainerName {
				var containerCommand []string
				if application.ContainerCommand != "" {
					containerCommand = []string{"sh", "-c", application.ContainerCommand}
				}
				deployment.Spec.Template.Spec.Containers[i].Command = containerCommand
				break
			}
		}

		if _, err := client.AppsV1().Deployments(ns).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}

	case "StatefulSet":
		statefulSet, err := client.AppsV1().StatefulSets(ns).Get(ctx, application.Slug, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if statefulSet.Spec.Template.Labels != nil {
			delete(statefulSet.Spec.Template.Labels, "app.ketches.cn/debugging")
		}

		appContainerName := "app-" + application.Slug
		for i := range statefulSet.Spec.Template.Spec.Containers {
			if statefulSet.Spec.Template.Spec.Containers[i].Name == appContainerName {
				statefulSet.Spec.Template.Spec.Containers[i].Command = nil
				break
			}
		}

		if _, err := client.AppsV1().StatefulSets(ns).Update(ctx, statefulSet, metav1.UpdateOptions{}); err != nil {
			return nil, err
		}
	}

	return application, nil
}

func executeDeleteAction(_ context.Context, application *entities.App) (*entities.App, error) {
	if err := DeleteApp(application.ID); err != nil {
		return nil, err
	}
	return application, nil
}

func ToAppResponse(a *entities.App) models.AppResponse {
	codeRepoID := ""
	if a.CodeRepositoryID != nil {
		codeRepoID = *a.CodeRepositoryID
	}
	res := models.AppResponse{
		ID:               a.ID,
		Slug:             a.Slug,
		Name:             a.Name,
		Description:      a.Description,
		EnvID:            a.EnvID,
		AppType:          a.AppType,
		CodeRepositoryID: codeRepoID,
		ContainerImage:   a.ContainerImage,
		ContainerCommand: a.ContainerCommand,
		RegistryUsername: a.RegistryUsername,
		RegistryPassword: a.RegistryPassword,
		Replicas:         a.Replicas,
		RequestCPU:       a.RequestCPU,
		RequestMemory:    a.RequestMemory,
		LimitCPU:         a.LimitCPU,
		LimitMemory:      a.LimitMemory,
		Status:           a.DeployStatus,
		CreatedAt:        a.CreatedAt,
	}

	if a.AutoScaling != nil {
		res.AutoScaling = &models.AutoScalingSpec{
			MinReplicas:             a.AutoScaling.MinReplicas,
			MaxReplicas:             a.AutoScaling.MaxReplicas,
			TargetCPUUtilization:    a.AutoScaling.TargetCPUUtilization,
			TargetMemoryUtilization: a.AutoScaling.TargetMemoryUtilization,
		}
	}

	if a.SchedulingRule != nil {
		res.SchedulingRule = &models.SchedulingSpec{
			RuleType:     a.SchedulingRule.RuleType,
			NodeName:     a.SchedulingRule.NodeName,
			NodeSelector: a.SchedulingRule.NodeSelector,
			NodeAffinity: a.SchedulingRule.NodeAffinity,
			Tolerations:  a.SchedulingRule.Tolerations,
		}
	}

	if a.Env.ID != "" {
		envResp := ToEnvResponse(&a.Env)
		res.Env = &envResp
	}

	return res
}

func GetAppTopology(appID string) (*models.AppTopologyResponse, error) {
	application, err := GetApp(appID)
	if err != nil {
		return nil, err
	}

	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return nil, err
	}

	return core.GetAppTopology(context.Background(), client, application)
}

func GetAppTopologyResourceYaml(appID string, nodeID string) (string, error) {
	application, err := GetApp(appID)
	if err != nil {
		return "", err
	}

	client, err := kube.GlobalClusterStore.GetClient(application.Env.ClusterID)
	if err != nil {
		return "", err
	}

	return core.GetAppTopologyResourceYaml(context.Background(), client, application, nodeID)
}
