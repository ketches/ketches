package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"unsafe"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

func TestSyncGatewaysToK8s_DeletesStaleHTTPRoutes(t *testing.T) {
	originalClusterHasGatewayAPICRDs := clusterHasGatewayAPICRDsForSync
	originalEnsureSharedGateway := ensureSharedGatewayForSync
	clusterHasGatewayAPICRDsForSync = func(string) (bool, error) { return true, nil }
	ensureSharedGatewayForSync = func(context.Context, string) error { return nil }
	t.Cleanup(func() {
		clusterHasGatewayAPICRDsForSync = originalClusterHasGatewayAPICRDs
		ensureSharedGatewayForSync = originalEnsureSharedGateway
	})

	clusterID := "cluster-sync-gateway"
	namespace := "demo-env"
	server := newGatewaySyncAPIServer()
	server.addHTTPRoute(&gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-app-route-stale",
			Namespace: namespace,
			Labels: map[string]string{
				kube.LabelAppSlug:        "demo-app",
				kube.LabelManagedBy:      "true",
				kube.LabelGatewayRouteID: "route-stale",
			},
		},
	})
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	storeTestClusterClients(t, clusterID, httpServer)
	defer kube.GlobalClusterStore.RemoveClient(clusterID)

	appCtx := &models.AppContext{
		App: entities.App{
			Base: entities.Base{
				ID: "app-1",
			},
			Slug: "demo-app",
		},
		EnvContext: models.EnvContext{
			Env: entities.Env{
				Base: entities.Base{
					ID: "env-1",
				},
				Slug:             "demo",
				ClusterID:        clusterID,
				ClusterNamespace: namespace,
			},
			Project: entities.Project{
				Base: entities.Base{
					ID: "project-1",
				},
				Slug: "proj",
			},
		},
		Gateways: []entities.AppGateway{
			{
				ID:          "gateway-1",
				Port:        8080,
				Protocol:    "tcp",
				ServiceType: "ClusterIP",
			},
		},
	}

	err := SyncGatewaysToK8s(context.Background(), appCtx)
	require.NoError(t, err)

	assert.Empty(t, server.listHTTPRoutes(namespace))
}

func TestSyncGatewaysToK8s_CreatesHTTPRoutesFromRouteGraph(t *testing.T) {
	originalClusterHasGatewayAPICRDs := clusterHasGatewayAPICRDsForSync
	originalEnsureSharedGateway := ensureSharedGatewayForSync
	clusterHasGatewayAPICRDsForSync = func(string) (bool, error) { return true, nil }
	ensureSharedGatewayForSync = func(context.Context, string) error { return nil }
	t.Cleanup(func() {
		clusterHasGatewayAPICRDsForSync = originalClusterHasGatewayAPICRDs
		ensureSharedGatewayForSync = originalEnsureSharedGateway
	})

	clusterID := "cluster-sync-routes"
	namespace := "demo-env"
	server := newGatewaySyncAPIServer()
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	storeTestClusterClients(t, clusterID, httpServer)
	defer kube.GlobalClusterStore.RemoveClient(clusterID)

	appCtx := &models.AppContext{
		App: entities.App{
			Base: entities.Base{ID: "app-1"},
			Slug: "demo-app",
		},
		EnvContext: models.EnvContext{
			Env: entities.Env{
				Base:             entities.Base{ID: "env-1"},
				Slug:             "demo",
				ClusterID:        clusterID,
				ClusterNamespace: namespace,
			},
			Project: entities.Project{
				Base: entities.Base{ID: "project-1"},
				Slug: "proj",
			},
		},
		Gateways: []entities.AppGateway{
			{
				ID:          "gateway-1",
				Port:        8080,
				Protocol:    "http",
				ServiceType: "ClusterIP",
			},
		},
		GatewayRoutes: []entities.AppGatewayHTTPRoute{
			{
				ID:               "route-http",
				AppGatewayID:     "gateway-1",
				Host:             "api.example.com",
				ListenerProtocol: "http",
				Path:             "/",
				PathMatchType:    "PathPrefix",
				Enabled:          true,
			},
			{
				ID:               "route-https",
				AppGatewayID:     "gateway-1",
				Host:             "secure.example.com",
				ListenerProtocol: "https",
				Path:             "/secure",
				PathMatchType:    "PathPrefix",
				Enabled:          true,
			},
		},
		GatewayBackends: []entities.AppGatewayHTTPRouteBackend{
			{ID: "backend-http", RouteID: "route-http", BackendAppID: "app-1", BackendPort: 8080, Weight: 100},
			{ID: "backend-https", RouteID: "route-https", BackendAppID: "app-1", BackendPort: 8080, Weight: 100},
		},
	}

	err := SyncGatewaysToK8s(context.Background(), appCtx)
	require.NoError(t, err)

	routes := server.listHTTPRoutes(namespace)
	require.Len(t, routes, 2)
	routeByName := make(map[string]gatewayv1.HTTPRoute, len(routes))
	for _, route := range routes {
		routeByName[route.Name] = route
	}

	httpRoute := routeByName["demo-app-route-http"]
	assert.Equal(t, "route-http", httpRoute.Labels[kube.LabelGatewayRouteID])
	require.Len(t, httpRoute.Spec.ParentRefs, 1)
	require.NotNil(t, httpRoute.Spec.ParentRefs[0].SectionName)
	assert.Equal(t, gatewayv1.SectionName("http"), *httpRoute.Spec.ParentRefs[0].SectionName)

	httpsRoute := routeByName["demo-app-route-https"]
	assert.Equal(t, "route-https", httpsRoute.Labels[kube.LabelGatewayRouteID])
	require.Len(t, httpsRoute.Spec.ParentRefs, 1)
	require.NotNil(t, httpsRoute.Spec.ParentRefs[0].SectionName)
	assert.Equal(t, buildSharedGatewayHTTPSListenerName("secure.example.com"), *httpsRoute.Spec.ParentRefs[0].SectionName)
}

func TestSyncGatewaysToK8s_DoesNotCreateClusterIPServiceWithoutGateways(t *testing.T) {
	clusterID := "cluster-sync-no-gateways"
	namespace := "demo-env"
	server := newGatewaySyncAPIServer()
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	storeTestClusterClients(t, clusterID, httpServer)
	defer kube.GlobalClusterStore.RemoveClient(clusterID)

	appCtx := &models.AppContext{
		App: entities.App{
			Base: entities.Base{ID: "app-1"},
			Slug: "demo-app",
		},
		EnvContext: models.EnvContext{
			Env: entities.Env{
				Base:             entities.Base{ID: "env-1"},
				Slug:             "demo",
				ClusterID:        clusterID,
				ClusterNamespace: namespace,
			},
			Project: entities.Project{
				Base: entities.Base{ID: "project-1"},
				Slug: "proj",
			},
		},
	}

	err := SyncGatewaysToK8s(context.Background(), appCtx)
	require.NoError(t, err)

	assert.Empty(t, server.listServices(namespace))
}

func TestDeleteGatewayFromK8s_DeletesClusterIPServiceWhenRemovingLastGateway(t *testing.T) {
	clusterID := "cluster-delete-last-gateway"
	namespace := "demo-env"
	server := newGatewaySyncAPIServer()
	server.addService(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-app",
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{Name: "port-8080", Port: 8080},
			},
		},
	})
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	storeTestClusterClients(t, clusterID, httpServer)
	defer kube.GlobalClusterStore.RemoveClient(clusterID)

	gateway := entities.AppGateway{
		ID:          "gateway-1",
		Port:        8080,
		Protocol:    "tcp",
		ServiceType: "ClusterIP",
	}
	appCtx := &models.AppContext{
		App: entities.App{
			Base: entities.Base{ID: "app-1"},
			Slug: "demo-app",
		},
		EnvContext: models.EnvContext{
			Env: entities.Env{
				Base:             entities.Base{ID: "env-1"},
				Slug:             "demo",
				ClusterID:        clusterID,
				ClusterNamespace: namespace,
			},
			Project: entities.Project{
				Base: entities.Base{ID: "project-1"},
				Slug: "proj",
			},
		},
		Gateways: []entities.AppGateway{gateway},
	}

	err := DeleteGatewayFromK8s(context.Background(), appCtx, &gateway)
	require.NoError(t, err)

	assert.Empty(t, server.listServices(namespace))
}

type gatewaySyncAPIServer struct {
	mu         sync.Mutex
	services   map[string]*corev1.Service
	httpRoutes map[string]*gatewayv1.HTTPRoute
}

func newGatewaySyncAPIServer() *gatewaySyncAPIServer {
	return &gatewaySyncAPIServer{
		services:   make(map[string]*corev1.Service),
		httpRoutes: make(map[string]*gatewayv1.HTTPRoute),
	}
}

func (s *gatewaySyncAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/api/v1/namespaces/") && strings.Contains(r.URL.Path, "/services"):
		s.handleServices(w, r)
	case strings.HasPrefix(r.URL.Path, "/apis/gateway.networking.k8s.io/v1/namespaces/") && strings.Contains(r.URL.Path, "/httproutes"):
		s.handleHTTPRoutes(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *gatewaySyncAPIServer) handleServices(w http.ResponseWriter, r *http.Request) {
	namespace, name, collection := parseNamespacedResourcePath(r.URL.Path, "services")
	switch r.Method {
	case http.MethodGet:
		s.mu.Lock()
		service, ok := s.services[namespacedKey(namespace, name)]
		s.mu.Unlock()
		if !ok || collection {
			writeAPIError(w, apierrors.NewNotFound(schema.GroupResource{Resource: "services"}, name))
			return
		}
		writeJSON(w, service)
	case http.MethodPost:
		if !collection {
			http.NotFound(w, r)
			return
		}
		var service corev1.Service
		if !requireRequestDecode(w, r, &service) {
			return
		}
		s.mu.Lock()
		s.services[namespacedKey(namespace, service.Name)] = service.DeepCopy()
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(&service)
	case http.MethodPut:
		var service corev1.Service
		if !requireRequestDecode(w, r, &service) {
			return
		}
		s.mu.Lock()
		s.services[namespacedKey(namespace, name)] = service.DeepCopy()
		s.mu.Unlock()
		writeJSON(w, &service)
	case http.MethodDelete:
		s.mu.Lock()
		delete(s.services, namespacedKey(namespace, name))
		s.mu.Unlock()
		writeJSON(w, &metav1.Status{Status: metav1.StatusSuccess})
	default:
		http.NotFound(w, r)
	}
}

func (s *gatewaySyncAPIServer) handleHTTPRoutes(w http.ResponseWriter, r *http.Request) {
	namespace, name, collection := parseNamespacedResourcePath(r.URL.Path, "httproutes")
	switch r.Method {
	case http.MethodGet:
		if collection {
			s.mu.Lock()
			items := make([]gatewayv1.HTTPRoute, 0)
			for _, route := range s.httpRoutes {
				if route.Namespace != namespace {
					continue
				}
				if matchesLabelSelector(route.Labels, r.URL.Query().Get("labelSelector")) {
					items = append(items, *route.DeepCopy())
				}
			}
			s.mu.Unlock()
			writeJSON(w, &gatewayv1.HTTPRouteList{Items: items})
			return
		}
		s.mu.Lock()
		route, ok := s.httpRoutes[namespacedKey(namespace, name)]
		s.mu.Unlock()
		if !ok {
			writeAPIError(w, apierrors.NewNotFound(schema.GroupResource{Group: gatewayv1.GroupName, Resource: "httproutes"}, name))
			return
		}
		writeJSON(w, route)
	case http.MethodPost:
		if !collection {
			http.NotFound(w, r)
			return
		}
		var route gatewayv1.HTTPRoute
		if !requireRequestDecode(w, r, &route) {
			return
		}
		s.mu.Lock()
		s.httpRoutes[namespacedKey(namespace, route.Name)] = route.DeepCopy()
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(&route)
	case http.MethodPut:
		var route gatewayv1.HTTPRoute
		if !requireRequestDecode(w, r, &route) {
			return
		}
		s.mu.Lock()
		s.httpRoutes[namespacedKey(namespace, name)] = route.DeepCopy()
		s.mu.Unlock()
		writeJSON(w, &route)
	case http.MethodDelete:
		s.mu.Lock()
		delete(s.httpRoutes, namespacedKey(namespace, name))
		s.mu.Unlock()
		writeJSON(w, &metav1.Status{Status: metav1.StatusSuccess})
	default:
		http.NotFound(w, r)
	}
}

func (s *gatewaySyncAPIServer) addHTTPRoute(route *gatewayv1.HTTPRoute) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.httpRoutes[namespacedKey(route.Namespace, route.Name)] = route.DeepCopy()
}

func (s *gatewaySyncAPIServer) addService(service *corev1.Service) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[namespacedKey(service.Namespace, service.Name)] = service.DeepCopy()
}

func (s *gatewaySyncAPIServer) listServices(namespace string) []corev1.Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]corev1.Service, 0)
	for _, service := range s.services {
		if service.Namespace == namespace {
			items = append(items, *service.DeepCopy())
		}
	}
	return items
}

func (s *gatewaySyncAPIServer) listHTTPRoutes(namespace string) []gatewayv1.HTTPRoute {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]gatewayv1.HTTPRoute, 0)
	for _, route := range s.httpRoutes {
		if route.Namespace == namespace {
			items = append(items, *route.DeepCopy())
		}
	}
	return items
}

func parseNamespacedResourcePath(path, resource string) (namespace, name string, collection bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for idx := range parts {
		if parts[idx] != resource {
			continue
		}
		if idx >= 2 {
			namespace = parts[idx-1]
		}
		if idx == len(parts)-1 {
			return namespace, "", true
		}
		return namespace, parts[idx+1], false
	}
	return "", "", false
}

func namespacedKey(namespace, name string) string {
	return namespace + "/" + name
}

func matchesLabelSelector(labels map[string]string, selector string) bool {
	if selector == "" {
		return true
	}
	for _, requirement := range strings.Split(selector, ",") {
		parts := strings.SplitN(requirement, "=", 2)
		if len(parts) != 2 {
			return false
		}
		if labels[parts[0]] != parts[1] {
			return false
		}
	}
	return true
}

func requireRequestDecode(w http.ResponseWriter, r *http.Request, out any) bool {
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeAPIError(w http.ResponseWriter, err error) {
	status := metav1.Status{
		Status:  metav1.StatusFailure,
		Message: err.Error(),
		Reason:  metav1.StatusReasonUnknown,
		Code:    http.StatusInternalServerError,
	}
	if statusErr, ok := err.(*apierrors.StatusError); ok {
		status = statusErr.ErrStatus
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(status.Code))
	_ = json.NewEncoder(w).Encode(&status)
}

func writeJSON(w http.ResponseWriter, obj any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(obj)
}

func storeTestClusterClients(t *testing.T, clusterID string, server *httptest.Server) {
	t.Helper()

	config := &rest.Config{
		Host: server.URL,
		ContentConfig: rest.ContentConfig{
			AcceptContentTypes: runtime.ContentTypeJSON,
			ContentType:        runtime.ContentTypeJSON,
		},
	}

	httpClient := server.Client()

	kubeClient, err := kubernetes.NewForConfigAndClient(config, httpClient)
	require.NoError(t, err)

	gwClient, err := gatewayclient.NewForConfigAndClient(config, httpClient)
	require.NoError(t, err)

	clientsField := reflect.ValueOf(kube.GlobalClusterStore).Elem().FieldByName("clients")
	clientsMap := (*sync.Map)(unsafe.Pointer(clientsField.UnsafeAddr()))
	clientsMap.Store(clusterID, &kube.Clients{
		Kube:      kubeClient,
		Gateway:   gwClient,
		Dynamic:   dynamic.NewForConfigOrDie(config),
		HTTPProxy: httpClient,
	})
}
