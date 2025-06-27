package kube

import (
	"context"
	"log"
	"net/http"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	clusterClientset     = map[string]kubernetes.Interface{}
	clusterRuntimeClient = map[string]client.Client{}
	clusterKubeConfig    = map[string]*rest.Config{}
	clusterStoreset      = map[string]storeInterface{}
)

func ClusterStore(ctx context.Context, clusterID string) (storeInterface, app.Error) {
	if clusterStore, ok := clusterStoreset[clusterID]; ok {
		return clusterStore, nil
	}

	clientset, err := ClusterClientset(ctx, clusterID, false)
	if err != nil {
		return nil, err
	}

	store := loadStore(clientset)
	clusterStoreset[clusterID] = store
	return store, nil
}

func ClusterClientset(ctx context.Context, clusterID string, refresh bool) (kubernetes.Interface, app.Error) {
	if !refresh {
		if clientset, ok := clusterClientset[clusterID]; ok {
			return clientset, nil
		}
	}

	restConfig, err := RestConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	clusterKubeConfig[clusterID] = restConfig

	clientset, err := clientsetFromRestConfig(restConfig)
	if err != nil {
		return nil, err
	}

	clusterClientset[clusterID] = clientset
	return clientset, nil
}

func ClusterRuntimeClient(ctx context.Context, clusterID string) (client.Client, app.Error) {
	if c, ok := clusterRuntimeClient[clusterID]; ok {
		return c, nil
	}

	restConfig, err := RestConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	clusterKubeConfig[clusterID] = restConfig

	runtimeClient, err := runtimeClientFromRestConfig(restConfig)
	if err != nil {
		return nil, err
	}

	clusterRuntimeClient[clusterID] = runtimeClient
	return runtimeClient, nil
}

func RestConfig(ctx context.Context, clusterID string) (*rest.Config, app.Error) {
	if restConfig, ok := clusterKubeConfig[clusterID]; ok {
		return restConfig, nil
	}

	cluster := &entities.Cluster{}
	if err := db.Instance().First(&cluster, "id = ?", clusterID).Error; err != nil {
		log.Printf("Failed to get cluster %s: %v", clusterID, err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Cluster not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	kubeConfig := cluster.KubeConfig
	if kubeConfig == "" {
		return nil, app.NewError(http.StatusConflict, "KubeConfig is not set for cluster")
	}

	restConfig, err := restConfigFromKubeConfigBytes([]byte(kubeConfig))
	if err != nil {
		return nil, err
	}

	clusterKubeConfig[clusterID] = restConfig
	return restConfig, nil
}

func clientsetFromRestConfig(restConfig *rest.Config) (kubernetes.Interface, app.Error) {
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Printf("Failed to create Kubernetes clientset: %v", err)
		return nil, app.NewError(http.StatusInternalServerError, "Failed to create Kubernetes clientset")
	}

	return clientset, nil
}

func runtimeClientFromRestConfig(restConfig *rest.Config) (client.Client, app.Error) {
	kubeRuntimeClient, err := client.New(restConfig, client.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		log.Printf("Failed to create Kubernetes runtime client: %v", err)
		return nil, app.NewError(http.StatusInternalServerError, "Failed to create Kubernetes runtime client")
	}

	return kubeRuntimeClient, nil
}

func restConfigFromKubeConfigBytes(kubeConfigBytes []byte) (*rest.Config, app.Error) {
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfigBytes)
	if err != nil {
		log.Printf("Failed to parse kubeconfig: %v", err)
		return nil, app.NewError(http.StatusInternalServerError, "Failed to parse kubeconfig")
	}
	return restConfig, nil
}
