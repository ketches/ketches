package core

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

var (
	clusterHasGatewayAPICRDsForSync = ClusterHasGatewayAPICRDs
	ensureSharedGatewayForSync      = EnsureSharedGateway
)

// SyncGatewaysToK8s synchronizes an application's gateway resources and then
// refreshes the cluster-wide shared Gateway. Application resources are fenced by
// the app reconcile lock; the shared Gateway is deliberately updated after that
// lock is released so advisory database sessions are never nested.
func SyncGatewaysToK8s(ctx context.Context, appCtx *models.AppContext) error {
	var clusterID string
	if err := withAppReconcileLockContext(ctx, appCtx, func(latest *models.AppContext) error {
		clusterID = latest.EnvContext.Env.ClusterID
		return syncAppGatewaysToK8s(ctx, latest)
	}); err != nil {
		return err
	}
	return syncSharedGatewayForApp(ctx, clusterID)
}

// syncAppGatewaysToK8s synchronizes only resources owned by one application.
// Keep shared Gateway reconciliation outside this function: callers may already
// hold the application's advisory lock, while the shared Gateway has its own
// cluster-wide lock.
func syncAppGatewaysToK8s(ctx context.Context, appCtx *models.AppContext) error {
	if appCtx == nil {
		return app.NewErrorf("app gateway reconcile context is nil")
	}
	if appCtx.EnvContext.Env.ClusterID == "" {
		return app.NewErrorf("app environment has no cluster configured")
	}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	metadata := &AppMetadata{AppContext: appCtx}

	if err := syncClusterIPService(ctx, client, metadata, appCtx.Gateways); err != nil {
		return err
	}

	routesByGateway := groupGatewayRoutesForSync(appCtx.GatewayRoutes)
	backendsByRoute := groupGatewayBackendsForSync(appCtx.GatewayBackends)
	var gwClientReady bool
	var gwClientErr error

	// Sync per-gateway NodePort Services and Gateway API routes
	for _, gateway := range appCtx.Gateways {
		npSvcName := fmt.Sprintf("%s-np-%d", appCtx.App.Slug, gateway.Port)
		if gateway.ServiceType == "NodePort" {
			// Create/update per-gateway NodePort Service
			npSvc := metadata.BuildNodePortService(gateway)
			if err := applyService(ctx, client, npSvc); err != nil {
				return err
			}
		} else {
			// Delete per-gateway NodePort Service if it exists (ServiceType changed to ClusterIP)
			if err := client.CoreV1().Services(appCtx.EnvContext.Env.ClusterNamespace).Delete(ctx, npSvcName, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}

		if strings.EqualFold(gateway.Protocol, "http") && len(routesByGateway[gateway.ID]) > 0 {
			if !gwClientReady {
				gwClientErr = ensureGatewayAPIReadyForRouteSync(ctx, appCtx)
				gwClientReady = true
			}
			if gwClientErr != nil {
				return gwClientErr
			}

			gwClient, err := kube.GlobalClusterStore.GetGatewayClient(appCtx.EnvContext.Env.ClusterID)
			if err != nil {
				return err
			}

			for _, routeRow := range routesByGateway[gateway.ID] {
				routeName := buildGatewayHTTPRouteName(appCtx.App.Slug, routeRow.ID)
				if !routeRow.Enabled {
					if err := gwClient.GatewayV1().HTTPRoutes(appCtx.EnvContext.Env.ClusterNamespace).Delete(ctx, routeName, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
						return err
					}
					continue
				}

				route := BuildGatewayHTTPRoute(GatewayHTTPRouteBuildInput{
					AppSlug:   appCtx.App.Slug,
					AppID:     appCtx.App.ID,
					EnvID:     appCtx.EnvContext.Env.ID,
					Namespace: appCtx.EnvContext.Env.ClusterNamespace,
					GatewayID: gateway.ID,
					RouteID:   routeRow.ID,
					Route:     routeEntityToRouteSpecForSync(routeRow),
					Backends:  buildGatewayHTTPBackendInputsForSync(appCtx, gateway.Port, backendsByRoute[routeRow.ID]),
				})
				if route == nil {
					continue
				}
				if err := applyHTTPRoute(ctx, gwClient, route); err != nil {
					return err
				}
			}

		}
	}
	if err := cleanupStaleNodePortServices(ctx, client, appCtx); err != nil {
		return err
	}

	// Gateway API is optional. When it is not installed, there can be no
	// HTTPRoute cleanup to perform; the shared Gateway step will also be a no-op.
	hasGatewayAPI, err := clusterHasGatewayAPICRDsForSync(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}
	if hasGatewayAPI {
		if err := cleanupStaleHTTPRoutes(ctx, appCtx); err != nil {
			return err
		}
	}
	return nil
}

// syncSharedGatewayForApp reconciles the cluster-wide Gateway after an app
// reconcile lock has been released. The discovery hook remains injectable for
// tests and avoids acquiring a database lock when Gateway API is unavailable.
func syncSharedGatewayForApp(ctx context.Context, clusterID string) error {
	if clusterID == "" {
		return app.NewErrorf("app environment has no cluster configured")
	}
	hasGatewayAPI, err := clusterHasGatewayAPICRDsForSync(clusterID)
	if err != nil || !hasGatewayAPI {
		return err
	}
	return ensureSharedGatewayForSync(ctx, clusterID)
}

func cleanupStaleNodePortServices(ctx context.Context, client kubernetes.Interface, appCtx *models.AppContext) error {
	desiredNames := make(map[string]struct{})
	for _, gateway := range appCtx.Gateways {
		if gateway.ServiceType == "NodePort" {
			desiredNames[fmt.Sprintf("%s-np-%d", appCtx.App.Slug, gateway.Port)] = struct{}{}
		}
	}

	services, err := client.CoreV1().Services(appCtx.EnvContext.Env.ClusterNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: appOwnedSelector(appCtx.App.ID),
	})
	if err != nil {
		return err
	}
	legacyPrefix := appCtx.App.Slug + "-np-"
	for i := range services.Items {
		service := &services.Items[i]
		if _, ok := desiredNames[service.Name]; ok {
			continue
		}
		if service.Labels[kube.LabelComponent] != "node-port" && !strings.HasPrefix(service.Name, legacyPrefix) {
			continue
		}
		if err := client.CoreV1().Services(service.Namespace).Delete(ctx, service.Name, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// DeleteGatewayFromK8s removes Gateway API resources from Kubernetes
func DeleteGatewayFromK8s(ctx context.Context, appCtx *models.AppContext, gateway *entities.AppGateway) error {
	if appCtx.EnvContext.Env.ClusterID == "" {
		return app.NewErrorf("app environment has no cluster configured")
	}

	gwClient, err := kube.GlobalClusterStore.GetGatewayClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	protocol := gateway.Protocol
	if protocol == "http" {
		for _, route := range appCtx.GatewayRoutes {
			if route.AppGatewayID != gateway.ID {
				continue
			}
			routeName := buildGatewayHTTPRouteName(appCtx.App.Slug, route.ID)
			if err := gwClient.GatewayV1().HTTPRoutes(appCtx.EnvContext.Env.ClusterNamespace).Delete(ctx, routeName, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	// Delete per-gateway NodePort Service if it existed
	if gateway.ServiceType == "NodePort" {
		npSvcName := fmt.Sprintf("%s-np-%d", appCtx.App.Slug, gateway.Port)
		if err := client.CoreV1().Services(appCtx.EnvContext.Env.ClusterNamespace).Delete(ctx, npSvcName, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	metadata := &AppMetadata{AppContext: appCtx}
	if err := syncClusterIPService(ctx, client, metadata, remainingGatewaysAfterDelete(appCtx.Gateways, gateway.ID)); err != nil {
		return err
	}

	return nil
}

func syncClusterIPService(ctx context.Context, client *kubernetes.Clientset, metadata *AppMetadata, gateways []entities.AppGateway) error {
	namespace := metadata.AppContext.EnvContext.Env.ClusterNamespace
	name := metadata.AppContext.App.Slug
	if len(gateways) == 0 {
		if err := client.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return err
		}
		return nil
	}

	svc := metadata.BuildClusterIPService(gateways)
	return applyService(ctx, client, svc)
}

func remainingGatewaysAfterDelete(gateways []entities.AppGateway, deletedGatewayID string) []entities.AppGateway {
	remaining := make([]entities.AppGateway, 0, len(gateways))
	for _, gateway := range gateways {
		if gateway.ID == deletedGatewayID {
			continue
		}
		remaining = append(remaining, gateway)
	}
	return remaining
}

// BuildClusterIPService builds a shared ClusterIP Service containing all gateway ports.
// This service is used for internal cluster access and as the HTTPRoute backend.
func (m *AppMetadata) BuildClusterIPService(gateways []entities.AppGateway) *corev1.Service {
	var ports []corev1.ServicePort

	for _, gw := range gateways {
		portName := fmt.Sprintf("port-%d", gw.Port)
		if slices.ContainsFunc(ports, func(port corev1.ServicePort) bool {
			return portName == port.Name
		}) {
			continue
		}
		ports = append(ports, corev1.ServicePort{
			Name:       portName,
			Port:       int32(gw.Port),
			TargetPort: intstr.FromInt(gw.Port),
		})
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.AppContext.App.Slug,
			Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels:    withAppComponent(m.getLabels(), "cluster-ip"),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: m.getSelectorLabels(),
			Ports:    ports,
		},
	}
}

// BuildNodePortService builds a per-gateway NodePort Service for external cluster-node access.
// Name: {app.Slug}-np-{port}
func (m *AppMetadata) BuildNodePortService(gw entities.AppGateway) *corev1.Service {
	svcPort := corev1.ServicePort{
		Name:       fmt.Sprintf("port-%d", gw.Port),
		Port:       int32(gw.Port),
		TargetPort: intstr.FromInt(gw.Port),
	}
	if gw.NodePort != 0 {
		svcPort.NodePort = int32(gw.NodePort)
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-np-%d", m.AppContext.App.Slug, gw.Port),
			Namespace: m.AppContext.EnvContext.Env.ClusterNamespace,
			Labels:    withAppComponent(m.getLabels(), "node-port"),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeNodePort,
			Selector: m.getSelectorLabels(),
			Ports:    []corev1.ServicePort{svcPort},
		},
	}
}

// ReadNodePortsFromK8s reads the actual NodePorts assigned by K8s for NodePort gateways.
// It reads from each per-gateway NodePort Service ({slug}-np-{port}).
// Returns a map of containerPort -> nodePort.
func ReadNodePortsFromK8s(ctx context.Context, appCtx *models.AppContext) (map[int]int, error) {
	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return nil, err
	}
	result := make(map[int]int)
	for _, gw := range appCtx.Gateways {
		if gw.ServiceType != "NodePort" {
			continue
		}
		npSvcName := fmt.Sprintf("%s-np-%d", appCtx.App.Slug, gw.Port)
		svc, err := client.CoreV1().Services(appCtx.EnvContext.Env.ClusterNamespace).Get(ctx, npSvcName, metav1.GetOptions{})
		if err != nil {
			continue
		}
		for _, p := range svc.Spec.Ports {
			result[int(p.Port)] = int(p.NodePort)
		}
	}
	return result, nil
}

func ensureGatewayAPIReadyForRouteSync(_ context.Context, appCtx *models.AppContext) error {
	hasGWAPI, err := clusterHasGatewayAPICRDsForSync(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}
	if !hasGWAPI {
		return app.NewErrorf("Gateway API CRDs are not installed on cluster %s", appCtx.EnvContext.Env.ClusterID)
	}
	return nil
}

func applyHTTPRoute(ctx context.Context, gwClient gatewayclient.Interface, route *gatewayv1.HTTPRoute) error {
	if got, err := gwClient.GatewayV1().HTTPRoutes(route.Namespace).Get(ctx, route.Name, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			_, err = gwClient.GatewayV1().HTTPRoutes(route.Namespace).Create(ctx, route, metav1.CreateOptions{})
			return err
		}
		return err
	} else {
		route.ResourceVersion = got.ResourceVersion
		_, err = gwClient.GatewayV1().HTTPRoutes(route.Namespace).Update(ctx, route, metav1.UpdateOptions{})
		return err
	}
}

func cleanupStaleHTTPRoutes(ctx context.Context, appCtx *models.AppContext) error {
	gwClient, err := kube.GlobalClusterStore.GetGatewayClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	routes, err := gwClient.GatewayV1().HTTPRoutes(appCtx.EnvContext.Env.ClusterNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: kube.LabelAppSlug + "=" + appCtx.App.Slug,
	})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	desiredNames := make(map[string]struct{})
	for _, route := range appCtx.GatewayRoutes {
		if !route.Enabled {
			continue
		}
		desiredNames[buildGatewayHTTPRouteName(appCtx.App.Slug, route.ID)] = struct{}{}
	}

	for _, route := range routes.Items {
		if _, ok := desiredNames[route.Name]; ok {
			continue
		}
		if err := gwClient.GatewayV1().HTTPRoutes(route.Namespace).Delete(ctx, route.Name, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func appContextHasEnabledHTTPRoute(appCtx *models.AppContext) bool {
	for _, route := range appCtx.GatewayRoutes {
		if route.Enabled {
			return true
		}
	}
	return false
}

func groupGatewayRoutesForSync(routes []entities.AppGatewayHTTPRoute) map[string][]entities.AppGatewayHTTPRoute {
	grouped := make(map[string][]entities.AppGatewayHTTPRoute)
	for _, route := range routes {
		grouped[route.AppGatewayID] = append(grouped[route.AppGatewayID], route)
	}
	return grouped
}

func groupGatewayBackendsForSync(backends []entities.AppGatewayHTTPRouteBackend) map[string][]entities.AppGatewayHTTPRouteBackend {
	grouped := make(map[string][]entities.AppGatewayHTTPRouteBackend)
	for _, backend := range backends {
		grouped[backend.RouteID] = append(grouped[backend.RouteID], backend)
	}
	return grouped
}

func routeEntityToRouteSpecForSync(route entities.AppGatewayHTTPRoute) models.GatewayRouteSpec {
	spec := models.GatewayRouteSpec{
		ID:               route.ID,
		GatewayID:        route.AppGatewayID,
		Host:             route.Host,
		ListenerProtocol: route.ListenerProtocol,
		Path:             route.Path,
		PathMatchType:    route.PathMatchType,
		Enabled:          route.Enabled,
		SortOrder:        route.SortOrder,
	}
	if route.CertID != nil {
		spec.CertID = *route.CertID
	}
	decodeGatewaySyncJSON(route.MatchesJSON, &spec.Matches)
	decodeGatewaySyncJSON(route.FiltersJSON, &spec.Filters)
	decodeGatewaySyncJSON(route.TimeoutsJSON, &spec.Timeouts)
	decodeGatewaySyncJSON(route.RetryJSON, &spec.Retry)
	decodeGatewaySyncJSON(route.SessionPersistenceJSON, &spec.SessionPersistence)
	decodeGatewaySyncJSON(route.ExtensionJSON, &spec.Extension)
	return spec
}

func decodeGatewaySyncJSON[T any](blob entities.JSONBlob, target **T) {
	if len(blob) == 0 {
		return
	}
	var decoded T
	if err := json.Unmarshal(blob, &decoded); err == nil {
		*target = &decoded
	}
}

func buildGatewayHTTPBackendInputsForSync(appCtx *models.AppContext, gatewayPort int, backends []entities.AppGatewayHTTPRouteBackend) []GatewayHTTPBackendBuildInput {
	if len(backends) == 0 {
		return []GatewayHTTPBackendBuildInput{{
			ServiceName: appCtx.App.Slug,
			Port:        gatewayPort,
			Weight:      1,
		}}
	}
	result := make([]GatewayHTTPBackendBuildInput, 0, len(backends))
	for _, backend := range backends {
		serviceName := backendServiceNameForSync(appCtx, backend.BackendAppID)
		result = append(result, GatewayHTTPBackendBuildInput{
			ServiceName: serviceName,
			Port:        backend.BackendPort,
			Weight:      int32(backend.Weight),
		})
	}
	return result
}

func backendServiceNameForSync(appCtx *models.AppContext, backendAppID string) string {
	if backendAppID == "" || backendAppID == appCtx.App.ID {
		return appCtx.App.Slug
	}
	var backend entities.App
	if err := db.DB.Select("slug").Where("id = ?", backendAppID).First(&backend).Error; err == nil && strings.TrimSpace(backend.Slug) != "" {
		return backend.Slug
	}
	return appCtx.App.Slug
}
