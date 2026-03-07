package core

import (
	"context"
	"fmt"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	// gatewayAPIGroup is the API group for the Gateway API.
	gatewayAPIGroup = "gateway.networking.k8s.io"
	// gatewayAPIVersion is the version used for detection and resource creation.
	gatewayAPIVersion = "v1"
	// defaultGatewayClassName is the GatewayClass used when creating env gateways.
	// Envoy Gateway is the default; this can be overridden via cluster configuration in the future.
	defaultGatewayClassName = "eg"
)

// ClusterHasGatewayCRD reports whether the cluster identified by clusterID has
// the Gateway API Gateway CRD installed (gateway.networking.k8s.io/v1).
func ClusterHasGatewayCRD(clusterID string) (bool, error) {
	return clusterHasCRD(clusterID, gatewayAPIGroup+"/"+gatewayAPIVersion)
}

// ClusterHasHTTPRouteCRD reports whether the cluster identified by clusterID has
// the Gateway API HTTPRoute CRD installed.
func ClusterHasHTTPRouteCRD(clusterID string) (bool, error) {
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
		if errors.IsNotFound(err) {
			return false, nil
		}
		// Any other error (e.g. "no kind is registered") is also treated as absent.
		return false, nil
	}
	return true, nil
}

// EnvGatewayName returns the canonical Gateway resource name for an environment.
func EnvGatewayName(envSlug string) string {
	return envSlug + "-gateway-default"
}

// BuildEnvGateway constructs a Gateway resource for the given environment.
// When TLS certificates are present an HTTPS listener is added; an HTTP
// listener is always included.
// BuildEnvGateway creates the env-level Gateway object for a cluster. It adds
// listeners for HTTP (80) and optionally HTTPS (443). The default "eg"
// listener is always included.
func BuildEnvGateway(envCtx *models.EnvContext, certs []entities.Certificate) *gatewayv1.Gateway {
	env := envCtx.Env
	gatewayClassName := gatewayv1.ObjectName(defaultGatewayClassName)

	// HTTP listener is always present.
	httpPort := gatewayv1.PortNumber(80)
	httpProtocol := gatewayv1.HTTPProtocolType
	listeners := []gatewayv1.Listener{
		{
			Name:     "http",
			Port:     httpPort,
			Protocol: httpProtocol,
		},
	}

	// Add HTTPS listener when at least one certificate is attached to this env.
	if len(certs) > 0 {
		httpsPort := gatewayv1.PortNumber(443)
		httpsProtocol := gatewayv1.HTTPSProtocolType
		tlsMode := gatewayv1.TLSModeTerminate

		var certRefs []gatewayv1.SecretObjectReference
		for _, cert := range certs {
			certRefs = append(certRefs, gatewayv1.SecretObjectReference{
				Name: gatewayv1.ObjectName(cert.Name),
			})
		}

		listeners = append(listeners, gatewayv1.Listener{
			Name:     "https",
			Port:     httpsPort,
			Protocol: httpsProtocol,
			TLS: &gatewayv1.ListenerTLSConfig{
				Mode:            &tlsMode,
				CertificateRefs: certRefs,
			},
		})
	}

	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EnvGatewayName(env.Slug),
			Namespace: env.ClusterNamespace,
			Labels: map[string]string{
				kube.LabelEnvID:       env.ID,
				kube.LabelEnvSlug:     env.Slug,
				kube.LabelProjectID:   envCtx.Project.ID,
				kube.LabelProjectSlug: envCtx.Project.Slug,
				kube.LabelManagedBy:      "true",
			},
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayClassName,
			Listeners:        listeners,
		},
	}
}

// EnsureEnvGateway creates or updates the env-level Gateway resource in the
// cluster. It is a no-op when the cluster does not have the Gateway API CRDs.
func EnsureEnvGateway(ctx context.Context, envCtx *models.EnvContext, certs []entities.Certificate) error {
	env := envCtx.Env
	hasGW, err := ClusterHasGatewayCRD(env.ClusterID)
	if err != nil {
		return fmt.Errorf("checking gateway CRD: %w", err)
	}
	if !hasGW {
		// Gateway API not installed — nothing to do.
		return nil
	}

	gwClient, err := kube.GlobalClusterStore.GetGatewayClient(env.ClusterID)
	if err != nil {
		return err
	}

	desired := BuildEnvGateway(envCtx, certs)

	existing, err := gwClient.GatewayV1().Gateways(env.ClusterNamespace).Get(ctx, desired.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = gwClient.GatewayV1().Gateways(env.ClusterNamespace).Create(ctx, desired, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update the existing gateway with the desired spec.
	desired.ResourceVersion = existing.ResourceVersion
	_, err = gwClient.GatewayV1().Gateways(env.ClusterNamespace).Update(ctx, desired, metav1.UpdateOptions{})
	return err
}
