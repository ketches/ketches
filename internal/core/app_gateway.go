package core

import (
	"context"
	"fmt"
	"slices"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// SyncGatewaysToK8s synchronizes app gateways to Kubernetes cluster
// It creates/updates Service and HTTPRoute/TCPRoute resources as needed
func SyncGatewaysToK8s(ctx context.Context, appCtx *models.AppContext) error {
	if appCtx.EnvContext.Env.ClusterID == "" {
		return fmt.Errorf("app environment has no cluster configured")
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

		// If gateway is exposed, create/update Gateway API resources
		if gateway.Exposed {
			// Verify that Gateway API CRDs are installed before attempting to create an HTTPRoute.
			hasGWAPI, err := ClusterHasGatewayAPICRDs(appCtx.EnvContext.Env.ClusterID)
			if err != nil {
				return err
			}
			if !hasGWAPI {
				return fmt.Errorf("Gateway API CRDs are not installed on cluster %s", appCtx.EnvContext.Env.ClusterID)
			}
			// Ensure the env-level Gateway exists before creating HTTPRoute.
			if err := EnsureEnvGateway(ctx, &appCtx.EnvContext, nil); err != nil {
				return err
			}

			gwClient, err := kube.GlobalClusterStore.GetGatewayClient(appCtx.EnvContext.Env.ClusterID)
			if err != nil {
				return err
			}

			protocol := gateway.Protocol
			if protocol == "http" || protocol == "https" {
				// Create/Update HTTPRoute using metadata builder
				route := metadata.BuildHTTPRoute(gateway)
				if route != nil {
					if got, err := gwClient.GatewayV1().HTTPRoutes(route.Namespace).Get(ctx, route.Name, metav1.GetOptions{}); err != nil {
						if errors.IsNotFound(err) {
							if _, err := gwClient.GatewayV1().HTTPRoutes(route.Namespace).Create(ctx, route, metav1.CreateOptions{}); err != nil {
								return err
							}
						} else {
							return err
						}
					} else {
						route.ResourceVersion = got.ResourceVersion
						if _, err := gwClient.GatewayV1().HTTPRoutes(route.Namespace).Update(ctx, route, metav1.UpdateOptions{}); err != nil {
							return err
						}
					}
				}
			} else {
				// TCP/UDP - Create/Update TCPRoute or UDPRoute
				// TODO: Implement TCPRoute/UDPRoute when Gateway API supports them
			}
		}
	}

	return nil
}

// DeleteGatewayFromK8s removes Gateway API resources from Kubernetes
func DeleteGatewayFromK8s(ctx context.Context, appCtx *models.AppContext, gateway *entities.AppGateway) error {
	if appCtx.EnvContext.Env.ClusterID == "" {
		return fmt.Errorf("app environment has no cluster configured")
	}

	gwClient, err := kube.GlobalClusterStore.GetGatewayClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	protocol := gateway.Protocol
	if protocol == "http" || protocol == "https" {
		// Delete HTTPRoute
		routeName := fmt.Sprintf("%s-%d", appCtx.App.Slug, gateway.Port)
		err := gwClient.GatewayV1().HTTPRoutes(appCtx.EnvContext.Env.ClusterNamespace).Delete(ctx, routeName, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	} else {
		// Delete TCPRoute/UDPRoute
		// TODO: Implement when Gateway API supports them
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
