package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func ListClusters(page, pageSize int, search string) (int64, []entities.Cluster, error) {
	var clusters []entities.Cluster
	var total int64
	query := db.DB.Model(&entities.Cluster{})
	if search != "" {
		query = query.Where("name LIKE ? OR slug LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := query.Order("created_at").Offset((page - 1) * pageSize).Limit(pageSize).Find(&clusters).Error; err != nil {
		return 0, nil, err
	}
	slog.Debug(fmt.Sprintf("Service ListClusters: found %d clusters out of %d total", len(clusters), total))
	return total, clusters, nil
}

func ListClustersSimple() ([]models.SimpleCluster, error) {
	var clusters []models.SimpleCluster
	if err := db.DB.Model(&entities.Cluster{}).Select("id, slug, name, description, connection_status, enabled").Order("created_at").Find(&clusters).Error; err != nil {
		return nil, err
	}
	return clusters, nil
}

func CreateCluster(req *models.CreateClusterRequest) (*entities.Cluster, error) {
	var existing entities.Cluster
	if err := db.DB.Where("slug = ?", req.Slug).First(&existing).Error; err == nil {
		return nil, app.NewErrorf("cluster with slug %s already exists", req.Slug)
	}

	cluster := &entities.Cluster{
		Base:        entities.Base{ID: uuid.New()},
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		GatewayHost: req.GatewayHost,
		Enabled:     true,
	}

	encryptedKubeConfig, err := secrets.EncryptString(req.KubeConfig)
	if err != nil {
		return nil, err
	}
	cluster.KubeConfig = encryptedKubeConfig

	if err := db.DB.Create(cluster).Error; err != nil {
		return nil, err
	}

	if err := kube.GlobalClusterStore.AddClient(cluster.ID, req.KubeConfig); err != nil {
		return nil, err
	}

	return cluster, nil
}

func GetSimpleCluster(clusterID string) (*models.SimpleCluster, error) {
	var cluster models.SimpleCluster
	if err := db.DB.Model(&entities.Cluster{}).Select("id, slug, name, description, connection_status, enabled").Where("id = ?", clusterID).First(&cluster).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

func GetCluster(clusterID string) (*entities.Cluster, error) {
	var cluster entities.Cluster
	if err := db.DB.First(&cluster, "id = ?", clusterID).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

func UpdateCluster(clusterID string, req *models.UpdateClusterRequest) (*entities.Cluster, error) {
	cluster, err := GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	cluster.Name = req.Name
	cluster.Description = req.Description
	cluster.GatewayHost = req.GatewayHost

	encryptedKubeConfig, err := secrets.EncryptString(req.KubeConfig)
	if err != nil {
		return nil, err
	}
	cluster.KubeConfig = encryptedKubeConfig

	if err := db.DB.Save(cluster).Error; err != nil {
		return nil, err
	}

	if cluster.Enabled {
		if err := kube.GlobalClusterStore.AddClient(cluster.ID, req.KubeConfig); err != nil {
			return nil, err
		}
	}
	return cluster, nil
}

func UpdateClusterBasic(clusterID string, req *models.UpdateBasicInfoRequest) (*entities.Cluster, error) {
	cluster, err := GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	cluster.Name = req.Name
	cluster.Description = req.Description

	if err := db.DB.Save(cluster).Error; err != nil {
		return nil, err
	}

	return cluster, nil
}

func UpdateClusterCredentials(clusterID string, req *models.UpdateClusterCredentialsRequest) (*entities.Cluster, error) {
	cluster, err := GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	cluster.GatewayHost = req.GatewayHost

	if req.KubeConfig != "" {
		encryptedKubeConfig, err := secrets.EncryptString(req.KubeConfig)
		if err != nil {
			return nil, err
		}
		cluster.KubeConfig = encryptedKubeConfig
	}

	if err := db.DB.Save(cluster).Error; err != nil {
		return nil, err
	}

	if cluster.Enabled && req.KubeConfig != "" {
		if err := kube.GlobalClusterStore.AddClient(cluster.ID, req.KubeConfig); err != nil {
			return nil, err
		}
	}
	return cluster, nil
}

func DeleteCluster(clusterID string) error {
	if err := db.DB.Delete(&entities.Cluster{}, "id = ?", clusterID).Error; err != nil {
		return err
	}
	kube.GlobalClusterStore.RemoveClient(clusterID)
	return nil
}

func CreateClusterGatewayProvider(clusterID string, req *models.CreateClusterGatewayProviderRequest) (*entities.ClusterGatewayProvider, error) {
	provider := &entities.ClusterGatewayProvider{
		ID:               uuid.New(),
		ClusterID:        clusterID,
		SourceType:       "adopted",
		DisplayName:      firstNonEmpty(strings.TrimSpace(req.DisplayName), strings.TrimSpace(req.GatewayClassName)),
		GatewayClassName: strings.TrimSpace(req.GatewayClassName),
		ControllerName:   strings.TrimSpace(req.ControllerName),
		IsDefault:        req.MakeDefault,
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := upsertClusterGatewayProvider(tx, provider); err != nil {
			return err
		}
		if provider.IsDefault {
			return setDefaultClusterGatewayProvider(tx, clusterID, provider.GatewayClassName)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	var stored entities.ClusterGatewayProvider
	if err := db.DB.Where("cluster_id = ? AND gateway_class_name = ?", clusterID, provider.GatewayClassName).First(&stored).Error; err != nil {
		return nil, err
	}
	return &stored, nil
}

func DeleteClusterGatewayProvider(clusterID, providerID string) error {
	var provider entities.ClusterGatewayProvider
	if err := db.DB.Where("cluster_id = ? AND id = ?", clusterID, providerID).First(&provider).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return app.NewErrorf("gateway provider not found")
		}
		return err
	}
	if provider.IsDefault {
		return app.NewErrorf("cannot remove the default gateway provider; set another provider as default first")
	}
	if provider.SourceType != "adopted" {
		return app.NewErrorf("only adopted gateway providers can be removed directly")
	}
	return db.DB.Delete(&provider).Error
}

func ListClusterGatewayProviders(clusterID string) ([]models.ClusterGatewayProvider, error) {
	var records []entities.ClusterGatewayProvider
	if err := db.DB.Where("cluster_id = ?", clusterID).Order("is_default DESC, created_at ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]models.ClusterGatewayProvider, 0, len(records))
	for _, record := range records {
		result = append(result, toClusterGatewayProviderModel(&record))
	}
	return result, nil
}

func upsertClusterGatewayProvider(tx *gorm.DB, provider *entities.ClusterGatewayProvider) error {
	var existing entities.ClusterGatewayProvider
	err := tx.Where("cluster_id = ? AND gateway_class_name = ?", provider.ClusterID, provider.GatewayClassName).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(provider).Error
	}
	if err != nil {
		return err
	}

	updates := map[string]any{
		"source_type":          provider.SourceType,
		"display_name":         provider.DisplayName,
		"controller_name":      provider.ControllerName,
		"extension_id":         provider.ExtensionID,
		"cluster_extension_id": provider.ClusterExtensionID,
		"is_default":           provider.IsDefault,
	}
	return tx.Model(&existing).Updates(updates).Error
}

func setDefaultClusterGatewayProvider(tx *gorm.DB, clusterID, gatewayClassName string) error {
	if err := tx.Model(&entities.ClusterGatewayProvider{}).Where("cluster_id = ?", clusterID).Update("is_default", false).Error; err != nil {
		return err
	}
	return tx.Model(&entities.ClusterGatewayProvider{}).Where("cluster_id = ? AND gateway_class_name = ?", clusterID, gatewayClassName).Update("is_default", true).Error
}

func RegisterAdoptedGatewayProvider(clusterID, gatewayClassName, controllerName string) (*entities.ClusterGatewayProvider, error) {
	provider := &entities.ClusterGatewayProvider{
		ID:               uuid.New(),
		ClusterID:        clusterID,
		SourceType:       "adopted",
		DisplayName:      gatewayClassName,
		GatewayClassName: gatewayClassName,
		ControllerName:   controllerName,
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&entities.ClusterGatewayProvider{}).Where("cluster_id = ? AND is_default = ?", clusterID, true).Count(&count).Error; err != nil {
			return err
		}
		provider.IsDefault = count == 0
		if err := upsertClusterGatewayProvider(tx, provider); err != nil {
			return err
		}
		if provider.IsDefault {
			return setDefaultClusterGatewayProvider(tx, clusterID, gatewayClassName)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	var stored entities.ClusterGatewayProvider
	if err := db.DB.Where("cluster_id = ? AND gateway_class_name = ?", clusterID, gatewayClassName).First(&stored).Error; err != nil {
		return nil, err
	}
	return &stored, nil
}

func SetDefaultClusterGatewayProvider(clusterID, gatewayClassName, controllerName, sourceType string) (*entities.ClusterGatewayProvider, error) {
	if sourceType == "" {
		sourceType = "adopted"
	}
	if sourceType != "adopted" && sourceType != "managed" {
		return nil, app.NewErrorf("management mode must be one of: adopted, managed")
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		var provider entities.ClusterGatewayProvider
		err := tx.Where("cluster_id = ? AND gateway_class_name = ?", clusterID, gatewayClassName).First(&provider).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			provider = entities.ClusterGatewayProvider{
				ID:               uuid.New(),
				ClusterID:        clusterID,
				SourceType:       sourceType,
				DisplayName:      gatewayClassName,
				GatewayClassName: gatewayClassName,
				ControllerName:   controllerName,
				IsDefault:        true,
			}
			if err := tx.Create(&provider).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			if controllerName != "" {
				provider.ControllerName = controllerName
			}
			provider.SourceType = sourceType
			provider.IsDefault = true
			if err := tx.Model(&provider).Updates(map[string]any{
				"controller_name": provider.ControllerName,
				"source_type":     provider.SourceType,
				"is_default":      true,
			}).Error; err != nil {
				return err
			}
		}
		return setDefaultClusterGatewayProvider(tx, clusterID, gatewayClassName)
	}); err != nil {
		return nil, err
	}
	var stored entities.ClusterGatewayProvider
	if err := db.DB.Where("cluster_id = ? AND gateway_class_name = ?", clusterID, gatewayClassName).First(&stored).Error; err != nil {
		return nil, err
	}
	return &stored, nil
}

func GetDefaultClusterGatewayProvider(clusterID string) (*entities.ClusterGatewayProvider, error) {
	var provider entities.ClusterGatewayProvider
	if err := db.DB.Where("cluster_id = ? AND is_default = ?", clusterID, true).First(&provider).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &provider, nil
}

func ListClusterGatewayClasses(clusterID string) ([]models.GatewayClassSummary, error) {
	gwClient, err := kube.GlobalClusterStore.GetGatewayClient(clusterID)
	if err != nil {
		return nil, err
	}

	defaultProvider, err := GetDefaultClusterGatewayProvider(clusterID)
	if err != nil {
		return nil, err
	}

	items, err := gwClient.GatewayV1().GatewayClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]models.GatewayClassSummary, 0, len(items.Items))
	for _, item := range items.Items {
		accepted := false
		for _, condition := range item.Status.Conditions {
			if condition.Type == string(gatewayv1.GatewayClassConditionStatusAccepted) && condition.Status == metav1.ConditionTrue {
				accepted = true
				break
			}
		}
		result = append(result, models.GatewayClassSummary{
			Name:           item.Name,
			ControllerName: string(item.Spec.ControllerName),
			Accepted:       accepted,
			IsDefault:      defaultProvider != nil && item.Name == defaultProvider.GatewayClassName,
		})
	}

	return result, nil
}

func UpdateClusterDefaultGatewayClass(clusterID string, req *models.UpdateClusterGatewayClassRequest) (*entities.Cluster, error) {
	_, err := SetDefaultClusterGatewayProvider(clusterID, strings.TrimSpace(req.GatewayClassName), strings.TrimSpace(req.GatewayControllerName), strings.TrimSpace(req.ManagementMode))
	if err != nil {
		return nil, err
	}
	return GetCluster(clusterID)
}

func ListClusterNodes(clusterID string) (any, error) {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return nil, err
	}
	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodes.Items, nil
}

func GetClusterNode(clusterID string, nodeName string) (any, error) {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return nil, err
	}
	node, err := client.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return node, nil
}

func UpdateClusterNodeLabels(clusterID string, nodeName string, labels map[string]string) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	node, err := client.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	node.Labels = labels
	_, err = client.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
	return err
}

func UpdateClusterNodeAnnotations(clusterID string, nodeName string, annotations map[string]string) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	node, err := client.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	node.Annotations = annotations
	_, err = client.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
	return err
}

func UpdateClusterNodeTaints(clusterID string, nodeName string, reqTaints []models.NodeTaint) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	node, err := client.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	var taints []corev1.Taint
	for _, t := range reqTaints {
		taints = append(taints, corev1.Taint{
			Key:    t.Key,
			Value:  t.Value,
			Effect: corev1.TaintEffect(t.Effect),
		})
	}

	node.Spec.Taints = taints
	_, err = client.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
	return err
}

func CordonClusterNode(clusterID string, nodeName string, cordon bool) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	node, err := client.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	node.Spec.Unschedulable = cordon
	_, err = client.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
	return err
}

func ListClusterNamespaces(clusterID string) ([]string, error) {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return nil, err
	}
	namespaces, err := client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	res := make([]string, 0, len(namespaces.Items))
	for _, ns := range namespaces.Items {
		res = append(res, ns.Name)
	}
	return res, nil
}

func ListClusterServices(clusterID string, namespace string) ([]string, error) {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return nil, err
	}
	services, err := client.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	res := make([]string, 0, len(services.Items))
	for _, svc := range services.Items {
		res = append(res, svc.Name)
	}
	return res, nil
}

func ListClusterServicesWithPorts(clusterID string, namespace string) ([]models.ClusterServiceResponse, error) {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return nil, err
	}
	serviceList, err := client.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return buildClusterServiceResponses(serviceList.Items), nil
}

func buildClusterServiceResponses(services []corev1.Service) []models.ClusterServiceResponse {
	res := make([]models.ClusterServiceResponse, 0, len(services))
	for _, svc := range services {
		ports := make([]models.ClusterServicePortResponse, 0, len(svc.Spec.Ports))
		for _, svcPort := range svc.Spec.Ports {
			ports = append(ports, models.ClusterServicePortResponse{
				Name:       svcPort.Name,
				Protocol:   string(svcPort.Protocol),
				Port:       svcPort.Port,
				TargetPort: svcPort.TargetPort.String(),
				NodePort:   svcPort.NodePort,
			})
		}

		res = append(res, models.ClusterServiceResponse{
			Name:  svc.Name,
			Ports: ports,
		})
	}

	return res
}

type StorageClassInfo struct {
	Name        string `json:"name"`
	Provisioner string `json:"provisioner"`
	IsDefault   bool   `json:"is_default"`
}

func ListStorageClasses(clusterID string) ([]StorageClassInfo, error) {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return nil, err
	}
	storageClasses, err := client.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	res := make([]StorageClassInfo, 0, len(storageClasses.Items))
	for _, sc := range storageClasses.Items {
		res = append(res, StorageClassInfo{
			Name:        sc.Name,
			Provisioner: sc.Provisioner,
			IsDefault:   sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true",
		})
	}
	return res, nil
}

func ExecClusterNodeTerminal(clusterID string, nodeName string, stdin io.Reader, stdout, stderr io.Writer) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	cluster, err := GetCluster(clusterID)
	if err != nil {
		return err
	}

	plaintextKubeConfig, err := secrets.DecryptString(cluster.KubeConfig)
	if err != nil {
		return err
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(plaintextKubeConfig))
	if err != nil {
		return err
	}

	pods := client.CoreV1().Pods(nodeTerminalNamespace)
	now := nodeTerminalNow()
	pod, err := ensureNodeTerminalPod(context.Background(), pods, nodeName, now)
	if err != nil {
		return app.WrapErrorf(err, "failed to prepare node terminal pod: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), nodeTerminalStartupTimeout)
	defer cancel()

	pod, err = waitForNodeTerminalPodRunning(waitCtx, pods, nodeName, pod.Name)
	if err != nil {
		if deleteErr := deleteNodeTerminalPod(context.Background(), pods, pod.Name); deleteErr != nil {
			slog.Error(fmt.Sprintf("ExecClusterNodeTerminal: failed to delete unhealthy pod %s: %v", pod.Name, deleteErr))
		}
		return err
	}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(nodeTerminalNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: nodeTerminalContainerName,
			Command:   []string{"nsenter", "-t", "1", "-m", "-u", "-i", "-n", "-p", "--", "sh", "-c", "clear; exec $(command -v bash || command -v ash || command -v sh)"},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}

	return exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    true,
	})
}

func boolPtr(b bool) *bool {
	return &b
}

func InitClusters() error {
	var clusters []entities.Cluster
	if err := db.DB.Find(&clusters).Error; err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("InitClusters: found %d clusters to initialize", len(clusters)))

	for _, cluster := range clusters {
		if cluster.Enabled {
			plaintextKubeConfig, err := secrets.DecryptString(cluster.KubeConfig)
			if err != nil {
				if strings.Contains(err.Error(), `ciphertext missing "enc:v1:" prefix`) {
					plaintextKubeConfig = cluster.KubeConfig
					encryptedKubeConfig, encryptErr := secrets.EncryptString(plaintextKubeConfig)
					if encryptErr != nil {
						return encryptErr
					}
					if saveErr := db.DB.Model(&entities.Cluster{}).Where("id = ?", cluster.ID).Update("kube_config", encryptedKubeConfig).Error; saveErr != nil {
						return saveErr
					}
					cluster.KubeConfig = encryptedKubeConfig
				} else {
					kube.GlobalClusterStore.RemoveClient(cluster.ID)
					updateClusterConnectionStatus(cluster.ID, "disconnected", err.Error(), "")
					slog.Error(fmt.Sprintf("InitClusters: failed to decrypt kubeconfig for cluster %s (%s): %v", cluster.Name, cluster.ID, err))
					continue
				}
			}
			if err := kube.GlobalClusterStore.AddClient(cluster.ID, plaintextKubeConfig); err != nil {
				kube.GlobalClusterStore.RemoveClient(cluster.ID)
				updateClusterConnectionStatus(cluster.ID, "disconnected", err.Error(), "")
				slog.Error(fmt.Sprintf("InitClusters: failed to initialize client for cluster %s (%s): %v", cluster.Name, cluster.ID, err))
				continue
			}
		}
	}
	return nil
}

func PingCluster(req *models.PingClusterRequest) error {
	client, err := kube.CreateClientFromKubeConfig(req.KubeConfig)
	if err != nil {
		return err
	}

	_, err = client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{Limit: 1})
	if err != nil {
		return err
	}

	return nil
}

func CheckClusterConnectivity(clusterID string) {
	go func() {
		cluster, err := GetCluster(clusterID)
		if err != nil {
			slog.Error(fmt.Sprintf("CheckClusterConnectivity: failed to get cluster %s: %v", clusterID, err))
			return
		}

		plaintextKubeConfig, decryptErr := secrets.DecryptString(cluster.KubeConfig)
		if decryptErr != nil {
			updateClusterConnectionStatus(clusterID, "disconnected", decryptErr.Error(), "")
			return
		}

		client, err := kube.CreateClientFromKubeConfig(plaintextKubeConfig)
		if err != nil {
			updateClusterConnectionStatus(clusterID, "disconnected", err.Error(), "")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err = client.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
		if err != nil {
			updateClusterConnectionStatus(clusterID, "disconnected", err.Error(), "")
			return
		}

		// Get ApiServer address from config
		config, _ := clientcmd.RESTConfigFromKubeConfig([]byte(plaintextKubeConfig))
		apiServer := ""
		if config != nil {
			apiServer = config.Host
		}

		updateClusterConnectionStatus(clusterID, "connected", "", apiServer)
	}()
}

func updateClusterConnectionStatus(clusterID, status, reason, apiServer string) {
	now := time.Now()
	updates := map[string]any{
		"connection_status":        status,
		"connection_status_reason": reason,
		"last_checked_at":          &now,
	}
	if apiServer != "" {
		updates["api_server"] = apiServer
	}

	if err := db.DB.Model(&entities.Cluster{}).Where("id = ?", clusterID).Updates(updates).Error; err != nil {
		slog.Error(fmt.Sprintf("Failed to update cluster %s connection status: %v", clusterID, err))
	}
}

func CheckAllClustersConnectivity() {
	clusters, err := ListClustersSimple()
	if err != nil {
		slog.Error(fmt.Sprintf("CheckAllClustersConnectivity: failed to list clusters: %v", err))
		return
	}

	for _, cluster := range clusters {
		if cluster.Enabled {
			CheckClusterConnectivity(cluster.ID)
		}
	}
}

func toClusterGatewayProviderModel(provider *entities.ClusterGatewayProvider) models.ClusterGatewayProvider {
	extensionID := ""
	if provider.ExtensionID != nil {
		extensionID = *provider.ExtensionID
	}
	clusterExtensionID := ""
	if provider.ClusterExtensionID != nil {
		clusterExtensionID = *provider.ClusterExtensionID
	}
	return models.ClusterGatewayProvider{
		ID:                 provider.ID,
		ClusterID:          provider.ClusterID,
		SourceType:         provider.SourceType,
		DisplayName:        provider.DisplayName,
		GatewayClassName:   provider.GatewayClassName,
		ControllerName:     provider.ControllerName,
		ExtensionID:        extensionID,
		ClusterExtensionID: clusterExtensionID,
		IsDefault:          provider.IsDefault,
	}
}
