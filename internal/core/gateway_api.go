package core

import (
	"context"
	"errors"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	// gatewayAPIGroup is the API group for the Gateway API.
	gatewayAPIGroup = "gateway.networking.k8s.io"
	// gatewayAPIVersion is the version used for detection and resource creation.
	gatewayAPIVersion = "v1"
	// fallbackGatewayClassName preserves the legacy default until a cluster-level default is configured.
	fallbackGatewayClassName = "eg"
	// sharedGatewayNamespace is the fixed namespace that hosts the single shared Gateway.
	sharedGatewayNamespace = "ketches-system"
	// sharedGatewayName is the canonical name of the single shared Gateway.
	sharedGatewayName = "ketches-shared-gateway"
)

// ClusterHasGatewayCRD reports whether the cluster identified by clusterID has
// the Gateway API Gateway CRD installed (gateway.networking.k8s.io/v1).
func ClusterHasGatewayCRD(clusterID string) (bool, error) {
	return clusterHasCRD(clusterID, gatewayAPIGroup+"/"+gatewayAPIVersion)
}

// ClusterHasGatewayAPICRDs checks whether both the Gateway and HTTPRoute CRDs
// are installed on the cluster. Both resources belong to the same group/version,
// so a single discovery call is used.
func ClusterHasGatewayAPICRDs(clusterID string) (bool, error) {
	return clusterHasCRD(clusterID, gatewayAPIGroup+"/"+gatewayAPIVersion)
}

// clusterHasCRD uses the Kubernetes discovery API to check whether the given
// groupVersion is registered on the cluster.
func clusterHasCRD(clusterID string, groupVersion string) (bool, error) {
	kubeClient, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return false, err
	}

	_, err = kubeClient.Discovery().ServerResourcesForGroupVersion(groupVersion)
	if err != nil {
		// NotFound from the discovery API means the group/version is absent.
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		// Any other error (e.g. "no kind is registered") is also treated as absent.
		return false, nil
	}
	return true, nil
}

func resolveClusterGatewayClassName(clusterID string) (string, error) {
	var provider entities.ClusterGatewayProvider
	if err := db.DB.Where("cluster_id = ? AND is_default = ?", clusterID, true).First(&provider).Error; err == nil {
		return provider.GatewayClassName, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	return fallbackGatewayClassName, nil
}

func BuildGatewayClass(name, controllerName string) *gatewayv1.GatewayClass {
	return &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				kube.LabelManagedBy: "true",
			},
		},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: gatewayv1.GatewayController(controllerName),
		},
	}
}

func EnsureGatewayClass(ctx context.Context, clusterID, name, controllerName string) error {
	gwClient, err := kube.GlobalClusterStore.GetGatewayClient(clusterID)
	if err != nil {
		return err
	}

	desired := BuildGatewayClass(name, controllerName)
	existing, err := gwClient.GatewayV1().GatewayClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = gwClient.GatewayV1().GatewayClasses().Create(ctx, desired, metav1.CreateOptions{})
			return err
		}
		return err
	}

	if existing.Spec.ControllerName != desired.Spec.ControllerName {
		return app.NewErrorf("gateway class %q already exists with controller %q", name, existing.Spec.ControllerName)
	}

	return nil
}

// SharedGatewayName returns the canonical name of the single shared Gateway.
func SharedGatewayName() string {
	return sharedGatewayName
}

// SharedGatewayNamespace returns the namespace that hosts the single shared Gateway.
func SharedGatewayNamespace() string {
	return sharedGatewayNamespace
}

// BuildSharedGateway constructs the single shared Gateway resource for a cluster.
// It exposes one HTTP listener on port 80 and allows routes from all namespaces.
func BuildSharedGateway(gatewayClassName string) *gatewayv1.Gateway {
	resolvedGatewayClassName := gatewayv1.ObjectName(gatewayClassName)
	listenerName := gatewayv1.SectionName("http")
	fromAll := gatewayv1.NamespacesFromAll

	httpPort := gatewayv1.PortNumber(80)
	httpProtocol := gatewayv1.HTTPProtocolType

	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SharedGatewayName(),
			Namespace: SharedGatewayNamespace(),
			Labels: map[string]string{
				kube.LabelManagedBy: "true",
			},
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: resolvedGatewayClassName,
			Listeners: []gatewayv1.Listener{
				{
					Name:     listenerName,
					Port:     httpPort,
					Protocol: httpProtocol,
					AllowedRoutes: &gatewayv1.AllowedRoutes{
						Namespaces: &gatewayv1.RouteNamespaces{
							From: &fromAll,
						},
					},
				},
			},
		},
	}
}

// EnsureSharedGateway creates or updates the single shared Gateway resource in the
// cluster. It is a no-op when the cluster does not have the Gateway API CRDs.
func EnsureSharedGateway(ctx context.Context, clusterID string) error {
	hasGW, err := ClusterHasGatewayCRD(clusterID)
	if err != nil {
		return app.WrapErrorf(err, "checking gateway CRD: %w", err)
	}
	if !hasGW {
		return nil
	}

	if err := ensureSharedGatewayNamespace(ctx, clusterID); err != nil {
		return err
	}

	gwClient, err := kube.GlobalClusterStore.GetGatewayClient(clusterID)
	if err != nil {
		return err
	}

	gatewayClassName, err := resolveClusterGatewayClassName(clusterID)
	if err != nil {
		return err
	}

	desired := BuildSharedGateway(gatewayClassName)

	existing, err := gwClient.GatewayV1().Gateways(SharedGatewayNamespace()).Get(ctx, desired.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = gwClient.GatewayV1().Gateways(SharedGatewayNamespace()).Create(ctx, desired, metav1.CreateOptions{})
			return err
		}
		return err
	}

	desired.ResourceVersion = existing.ResourceVersion
	_, err = gwClient.GatewayV1().Gateways(SharedGatewayNamespace()).Update(ctx, desired, metav1.UpdateOptions{})
	return err
}

func ensureSharedGatewayNamespace(ctx context.Context, clusterID string) error {
	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
		return err
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: SharedGatewayNamespace(),
			Labels: map[string]string{
				kube.LabelManagedBy: "true",
			},
		},
	}

	_, err = client.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
