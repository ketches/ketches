package core

import (
	"context"
	"fmt"
	"slices"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// SyncGatewaysToK8s synchronizes app gateways to Kubernetes cluster
// It creates/updates Service and HTTPRoute/TCPRoute resources as needed
func SyncGatewaysToK8s(ctx context.Context, app *entities.App) error {
	if app.Env.ClusterID == "" {
		return fmt.Errorf("app environment has no cluster configured")
	}

	client, err := kube.GlobalClusterStore.GetClient(app.Env.ClusterID)
	if err != nil {
		return err
	}

	metadata := &AppMetadata{App: app}

	// Always sync Service (update ports based on all gateways)
	svc := metadata.BuildServiceForGateways(app.Gateways)
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

	for _, gateway := range app.Gateways {
		// If gateway is exposed, create/update Gateway API resources
		if gateway.Exposed {
			// Verify that Gateway API CRDs are installed before attempting to create an HTTPRoute.
			hasGWAPI, err := ClusterHasGatewayAPICRDs(app.Env.ClusterID)
			if err != nil {
				return err
			}
			if !hasGWAPI {
				return fmt.Errorf("Gateway API CRDs are not installed on cluster %s", app.Env.ClusterID)
			}
			// Ensure the env-level Gateway exists before creating HTTPRoute.
			if err := EnsureEnvGateway(ctx, &app.Env, nil); err != nil {
				return err
			}

			gwClient, err := kube.GlobalClusterStore.GetGatewayClient(app.Env.ClusterID)
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
				// For now, just ensure Service has the correct type
			}
		}
	}

	return nil
}

// DeleteGatewayFromK8s removes Gateway API resources from Kubernetes
func DeleteGatewayFromK8s(ctx context.Context, app *entities.App, gateway *entities.AppGateway) error {
	if app.Env.ClusterID == "" {
		return fmt.Errorf("app environment has no cluster configured")
	}

	gwClient, err := kube.GlobalClusterStore.GetGatewayClient(app.Env.ClusterID)
	if err != nil {
		return err
	}

	protocol := gateway.Protocol
	if protocol == "http" || protocol == "https" {
		// Delete HTTPRoute
		routeName := fmt.Sprintf("%s-%d", app.Slug, gateway.Port)
		err := gwClient.GatewayV1().HTTPRoutes(app.Env.ClusterNamespace).Delete(ctx, routeName, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	} else {
		// Delete TCPRoute/UDPRoute
		// TODO: Implement when Gateway API supports them
	}

	// Rebuild Service with remaining gateways
	client, err := kube.GlobalClusterStore.GetClient(app.Env.ClusterID)
	if err != nil {
		return err
	}

	// Load all remaining gateways for this app
	var remainingGateways []entities.AppGateway
	// Note: This assumes app.Gateways is up-to-date after deletion in DB
	// The service layer should reload the app with gateways before calling this
	remainingGateways = app.Gateways

	metadata := &AppMetadata{App: app}
	svc := metadata.BuildServiceForGateways(remainingGateways)

	if _, err := client.CoreV1().Services(svc.Namespace).Update(ctx, svc, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

// BuildServiceForGateways builds a Service with ports from all gateways
func (m *AppMetadata) BuildServiceForGateways(gateways []entities.AppGateway) *corev1.Service {
	var ports []corev1.ServicePort

	// Add a port for each gateway
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
			Name:      m.App.Slug,
			Namespace: m.App.Env.ClusterNamespace,
			Labels:    m.getLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: m.getLabels(),
			Ports:    ports,
		},
	}
}
