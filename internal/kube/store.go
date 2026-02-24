package kube

import (
	"sync"

	"github.com/ketches/ketches/internal/app"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

type Clients struct {
	Kube    *kubernetes.Clientset
	Gateway *gatewayclient.Clientset
	Dynamic dynamic.Interface
}

type ClusterStore struct {
	clients sync.Map
}

var GlobalClusterStore = &ClusterStore{}

func (s *ClusterStore) GetClient(clusterID string) (*kubernetes.Clientset, error) {
	if client, ok := s.clients.Load(clusterID); ok {
		return client.(*Clients).Kube, nil
	}
	return nil, app.ErrClusterNotFound
}

func (s *ClusterStore) GetGatewayClient(clusterID string) (*gatewayclient.Clientset, error) {
	if client, ok := s.clients.Load(clusterID); ok {
		return client.(*Clients).Gateway, nil
	}
	return nil, app.ErrClusterNotFound
}

func (s *ClusterStore) GetDynamicClient(clusterID string) (dynamic.Interface, error) {
	if client, ok := s.clients.Load(clusterID); ok {
		return client.(*Clients).Dynamic, nil
	}
	return nil, app.ErrClusterNotFound
}

func (s *ClusterStore) AddClient(clusterID, kubeConfig string) error {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	gatewayClient, err := gatewayclient.NewForConfig(config)
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	s.clients.Store(clusterID, &Clients{
		Kube:    kubeClient,
		Gateway: gatewayClient,
		Dynamic: dynamicClient,
	})
	return nil
}

func (s *ClusterStore) RemoveClient(clusterID string) {
	s.clients.Delete(clusterID)
}

func CreateClientFromKubeConfig(kubeConfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// CreateControllerRuntimeClientFromKubeConfig creates a controller-runtime client from a kubeconfig string.
// This is used by the helm-operator installer which requires a controller-runtime client.
func CreateControllerRuntimeClientFromKubeConfig(kubeConfig string) (crclient.Client, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
	if err != nil {
		return nil, err
	}

	return crclient.New(config, crclient.Options{})
}
