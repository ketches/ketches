package kube

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ketches/ketches/internal/app"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

type Clients struct {
	Kube      *kubernetes.Clientset
	Gateway   *gatewayclient.Clientset
	Dynamic   dynamic.Interface
	HTTPProxy *http.Client
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

// GetHTTPProxyClient returns the cached *http.Client for proxying requests to
// the cluster's K8s apiserver service proxy sub-resource. The client reuses
// TCP/TLS connections and never follows redirects so Location headers can be
// rewritten by the caller.
func (s *ClusterStore) GetHTTPProxyClient(clusterID string) (*http.Client, error) {
	if client, ok := s.clients.Load(clusterID); ok {
		return client.(*Clients).HTTPProxy, nil
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

	httpProxyClient, err := newHTTPProxyClient(config)
	if err != nil {
		return err
	}

	s.clients.Store(clusterID, &Clients{
		Kube:      kubeClient,
		Gateway:   gatewayClient,
		Dynamic:   dynamicClient,
		HTTPProxy: httpProxyClient,
	})
	return nil
}

func (s *ClusterStore) RemoveClient(clusterID string) {
	s.clients.Delete(clusterID)
}

// newHTTPProxyClient builds an *http.Client configured with the cluster's TLS
// settings. It does not follow redirects so that the proxy handler can inspect
// and rewrite Location headers before forwarding them to the browser.
func newHTTPProxyClient(restConfig *rest.Config) (*http.Client, error) {
	tlsCfg, err := rest.TLSConfigFor(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build TLS config for HTTP proxy client: %w", err)
	}
	if tlsCfg == nil {
		tlsCfg = &tls.Config{} // plain HTTP cluster — still use a typed transport
	}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:   tlsCfg,
			DisableKeepAlives: true, // force connection close so EOF is detectable (prevents nginx keep-alive hang)
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, nil
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
