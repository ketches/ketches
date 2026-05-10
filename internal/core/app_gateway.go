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
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

var (
	clusterHasGatewayAPICRDsForSync = ClusterHasGatewayAPICRDs
	ensureSharedGatewayForSync      = EnsureSharedGateway
)

// SyncGatewaysToK8s synchronizes app gateways to Kubernetes cluster
// It creates/updates Service and HTTPRoute/TCPRoute resources as needed
func SyncGatewaysToK8s(ctx context.Context, appCtx *models.AppContext) error {
	if appCtx.EnvContext.Env.ClusterID == "" {
		return app.NewErrorf("app environment has no cluster configured")
	}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	metadata := &AppMetadata{AppContext: appCtx}

	// Always sync shared ClusterIP Service (all gateway ports)
	svc := metadata.BuildClusterIPService(appCtx.Gateways)
	if _, err := client.CoreV1().Services(svc.Namespace).Get(ctx, svc.Name, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			if _, err := client.CoreV1().Services(svc.Namespace).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if _, err := client.CoreV1().Services(svc.Namespace).Update(ctx, svc, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	routesByGateway := groupGatewayRoutesForSync(appCtx.GatewayRoutes)
	backendsByRoute := groupGatewayBackendsForSync(appCtx.GatewayBackends)
	hasEnabledHTTPRoute := appContextHasEnabledHTTPRoute(appCtx)
	var gwClientReady bool
	var gwClientErr error

	// Sync per-gateway NodePort Services and Gateway API routes
	for _, gateway := range appCtx.Gateways {
		npSvcName := fmt.Sprintf("%s-np-%d", appCtx.App.Slug, gateway.Port)
		if gateway.ServiceType == "NodePort" {
			// Create/update per-gateway NodePort Service
			npSvc := metadata.BuildNodePortService(gateway)
			if _, err := client.CoreV1().Services(npSvc.Namespace).Get(ctx, npSvc.Name, metav1.GetOptions{}); err != nil {
				if errors.IsNotFound(err) {
					if _, err := client.CoreV1().Services(npSvc.Namespace).Create(ctx, npSvc, metav1.CreateOptions{}); err != nil {
						return err
					}
				} else {
					return err
				}
			} else {
				if _, err := client.CoreV1().Services(npSvc.Namespace).Update(ctx, npSvc, metav1.UpdateOptions{}); err != nil {
					return err
				}
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

			if err := deleteLegacyHTTPRouteNames(ctx, gwClient, appCtx, gateway); err != nil {
				return err
			}
		}
	}

	if hasEnabledHTTPRoute {
		if err := cleanupStaleHTTPRoutes(ctx, appCtx); err != nil {
			return err
		}
	} else if err := cleanupStaleHTTPRoutes(ctx, appCtx); err != nil {
		return err
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
		if err := deleteLegacyHTTPRouteNames(ctx, gwClient, appCtx, *gateway); err != nil {
			return err
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

	// Rebuild shared ClusterIP Service with remaining gateways
	metadata := &AppMetadata{AppContext: appCtx}
	svc := metadata.BuildClusterIPService(appCtx.Gateways)
	if _, err := client.CoreV1().Services(svc.Namespace).Update(ctx, svc, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
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
			Labels:    m.getLabels(),
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
			Labels:    m.getLabels(),
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

func ensureGatewayAPIReadyForRouteSync(ctx context.Context, appCtx *models.AppContext) error {
	hasGWAPI, err := clusterHasGatewayAPICRDsForSync(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}
	if !hasGWAPI {
		return app.NewErrorf("Gateway API CRDs are not installed on cluster %s", appCtx.EnvContext.Env.ClusterID)
	}
	return ensureSharedGatewayForSync(ctx, appCtx.EnvContext.Env.ClusterID)
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

func deleteLegacyHTTPRouteNames(ctx context.Context, gwClient gatewayclient.Interface, appCtx *models.AppContext, gateway entities.AppGateway) error {
	names := []string{
		buildLegacyAppGatewayHTTPRouteName(appCtx.App.Slug, gateway.Port),
		buildAppGatewayHTTPRouteName(appCtx.App.Slug, gateway),
	}
	for _, name := range names {
		if err := gwClient.GatewayV1().HTTPRoutes(appCtx.EnvContext.Env.ClusterNamespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
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
