package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

func ListClusters() ([]entities.Cluster, error) {
	var clusters []entities.Cluster
	if err := db.DB.Order("created_at asc").Find(&clusters).Error; err != nil {
		return nil, err
	}
	log.Printf("Service ListClusters: found %d clusters", len(clusters))
	return clusters, nil
}

func CreateCluster(req *models.CreateClusterRequest) (*entities.Cluster, error) {
	var existing entities.Cluster
	if err := db.DB.Where("slug = ?", req.Slug).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("cluster with slug %s already exists", req.Slug)
	}

	cluster := &entities.Cluster{
		Base:        entities.Base{ID: uuid.New()},
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
		KubeConfig:  req.KubeConfig,
		GatewayIP:   req.GatewayIP,
		Enabled:     true,
	}

	if err := db.DB.Create(cluster).Error; err != nil {
		return nil, err
	}

	if err := kube.GlobalClusterStore.AddClient(cluster.ID, cluster.KubeConfig); err != nil {
		return nil, err
	}

	return cluster, nil
}

func GetCluster(clusterID string) (*entities.Cluster, error) {
	var cluster entities.Cluster
	if err := db.DB.First(&cluster, "id = ?", clusterID).Error; err != nil {
		return nil, err
	}
	return &cluster, nil
}

func UpdateCluster(clusterID string, req *models.CreateClusterRequest) (*entities.Cluster, error) {
	cluster, err := GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	if cluster.Slug != req.Slug {
		var existing entities.Cluster
		if err := db.DB.Where("slug = ? AND id != ?", req.Slug, clusterID).First(&existing).Error; err == nil {
			return nil, fmt.Errorf("cluster with slug %s already exists", req.Slug)
		}
	}

	cluster.Slug = req.Slug
	cluster.Name = req.Name
	cluster.Description = req.Description
	cluster.KubeConfig = req.KubeConfig
	cluster.GatewayIP = req.GatewayIP

	if err := db.DB.Save(cluster).Error; err != nil {
		return nil, err
	}

	if cluster.Enabled {
		if err := kube.GlobalClusterStore.AddClient(cluster.ID, cluster.KubeConfig); err != nil {
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

	cluster.KubeConfig = req.KubeConfig
	cluster.GatewayIP = req.GatewayIP

	if err := db.DB.Save(cluster).Error; err != nil {
		return nil, err
	}

	if cluster.Enabled {
		if err := kube.GlobalClusterStore.AddClient(cluster.ID, cluster.KubeConfig); err != nil {
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

type StorageClassInfo struct {
	Name        string
	Provisioner string
	IsDefault   bool
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

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
	if err != nil {
		return err
	}

	podName := fmt.Sprintf("node-terminal-%s", uuid.New()[:8])
	namespace := "default"

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
			Containers: []corev1.Container{
				{
					Name:    "shell",
					Image:   "alpine:latest",
					Command: []string{"sh", "-c", "while true; do sleep 3600; done"},
					SecurityContext: &corev1.SecurityContext{
						Privileged: boolPtr(true),
					},
				},
			},
			HostPID:       true,
			HostNetwork:   true,
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	_, err = client.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create terminal pod: %v", err)
	}

	defer client.CoreV1().Pods(namespace).Delete(context.Background(), podName, metav1.DeleteOptions{})

	for i := 0; i < 30; i++ {
		p, err := client.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err == nil && p.Status.Phase == corev1.PodRunning {
			break
		}
		time.Sleep(1 * time.Second)
	}

	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "shell",
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

	log.Printf("InitClusters: found %d clusters to initialize", len(clusters))

	for _, cluster := range clusters {
		if cluster.Enabled {
			if err := kube.GlobalClusterStore.AddClient(cluster.ID, cluster.KubeConfig); err != nil {
				return err
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
			log.Printf("CheckClusterConnectivity: failed to get cluster %s: %v", clusterID, err)
			return
		}

		client, err := kube.CreateClientFromKubeConfig(cluster.KubeConfig)
		if err != nil {
			updateClusterConnectionStatus(clusterID, "disconnected", err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err = client.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
		if err != nil {
			updateClusterConnectionStatus(clusterID, "disconnected", err.Error())
			return
		}

		updateClusterConnectionStatus(clusterID, "connected", "")
	}()
}

func updateClusterConnectionStatus(clusterID, status, reason string) {
	now := time.Now()
	updates := map[string]any{
		"connection_status":        status,
		"connection_status_reason": reason,
		"last_checked_at":          &now,
	}

	if err := db.DB.Model(&entities.Cluster{}).Where("id = ?", clusterID).Updates(updates).Error; err != nil {
		log.Printf("Failed to update cluster %s connection status: %v", clusterID, err)
	}
}

func CheckAllClustersConnectivity() {
	clusters, err := ListClusters()
	if err != nil {
		log.Printf("CheckAllClustersConnectivity: failed to list clusters: %v", err)
		return
	}

	for _, cluster := range clusters {
		if cluster.Enabled {
			CheckClusterConnectivity(cluster.ID)
		}
	}
}
