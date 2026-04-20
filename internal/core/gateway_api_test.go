package core

import (
	"testing"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestBuildSharedGateway_UsesSingleHTTPListenerForAllNamespaces(t *testing.T) {
	gateway := BuildSharedGateway("ketches", nil)

	require.Equal(t, SharedGatewayName(), gateway.Name)
	require.Equal(t, SharedGatewayNamespace(), gateway.Namespace)
	require.Len(t, gateway.Spec.Listeners, 1)

	listener := gateway.Spec.Listeners[0]
	assert.Equal(t, gatewayv1.ObjectName("ketches"), gateway.Spec.GatewayClassName)
	assert.Equal(t, gatewayv1.SectionName("http"), listener.Name)
	assert.Equal(t, gatewayv1.PortNumber(80), listener.Port)
	assert.Equal(t, gatewayv1.HTTPProtocolType, listener.Protocol)
	require.NotNil(t, listener.AllowedRoutes)
	require.NotNil(t, listener.AllowedRoutes.Namespaces)
	require.NotNil(t, listener.AllowedRoutes.Namespaces.From)
	assert.Equal(t, gatewayv1.NamespacesFromAll, *listener.AllowedRoutes.Namespaces.From)
}

func TestBuildSharedGateway_AddsHTTPSListenersWithCertificateRefs(t *testing.T) {
	gateway := BuildSharedGateway("ketches", []sharedGatewayHTTPSListener{
		{
			Name:                  buildSharedGatewayHTTPSListenerName("secure.example.com"),
			Hostname:              gatewayv1.Hostname("secure.example.com"),
			CertificateSecretName: "ketches-cert-cert-1",
		},
	})

	require.Len(t, gateway.Spec.Listeners, 2)
	httpsListener := gateway.Spec.Listeners[1]
	assert.Equal(t, gatewayv1.HTTPSProtocolType, httpsListener.Protocol)
	assert.Equal(t, gatewayv1.PortNumber(443), httpsListener.Port)
	require.NotNil(t, httpsListener.Hostname)
	assert.Equal(t, gatewayv1.Hostname("secure.example.com"), *httpsListener.Hostname)
	require.NotNil(t, httpsListener.TLS)
	require.Len(t, httpsListener.TLS.CertificateRefs, 1)
	assert.Equal(t, gatewayv1.ObjectName("ketches-cert-cert-1"), httpsListener.TLS.CertificateRefs[0].Name)
}

func TestBuildHTTPRoute_UsesSharedGatewayParentRef(t *testing.T) {
	metadata := &AppMetadata{
		AppContext: &models.AppContext{
			App: entities.App{
				Slug: "demo-app",
			},
			EnvContext: models.EnvContext{
				Env: entities.Env{
					ClusterNamespace: "demo-env",
				},
			},
		},
	}

	route := metadata.BuildHTTPRoute(entities.AppGateway{
		Port:     8080,
		Protocol: "http",
		Domain:   "demo.example.com",
		Path:     "/",
		Exposed:  true,
	})

	require.NotNil(t, route)
	require.Len(t, route.Spec.ParentRefs, 1)

	parentRef := route.Spec.ParentRefs[0]
	require.NotNil(t, parentRef.Namespace)
	require.NotNil(t, parentRef.SectionName)
	assert.Equal(t, "demo-app-8080-http", route.Name)
	assert.Equal(t, gatewayv1.ObjectName(SharedGatewayName()), parentRef.Name)
	assert.Equal(t, gatewayv1.Namespace(SharedGatewayNamespace()), *parentRef.Namespace)
	assert.Equal(t, gatewayv1.SectionName("http"), *parentRef.SectionName)
}

func TestBuildHTTPRoute_UsesHTTPSListenerParentRef(t *testing.T) {
	metadata := &AppMetadata{
		AppContext: &models.AppContext{
			App: entities.App{
				Slug: "demo-app",
			},
			EnvContext: models.EnvContext{
				Env: entities.Env{
					ClusterNamespace: "demo-env",
				},
			},
		},
	}

	route := metadata.BuildHTTPRoute(entities.AppGateway{
		Port:     8443,
		Protocol: "https",
		Domain:   "secure.example.com",
		Path:     "/",
		Exposed:  true,
	})

	require.NotNil(t, route)
	require.Len(t, route.Spec.ParentRefs, 1)

	parentRef := route.Spec.ParentRefs[0]
	require.NotNil(t, parentRef.SectionName)
	assert.Equal(t, "demo-app-8443-https", route.Name)
	assert.Equal(t, buildSharedGatewayHTTPSListenerName("secure.example.com"), *parentRef.SectionName)
}

func TestBuildGatewayClass_UsesRequestedControllerName(t *testing.T) {
	gatewayClass := BuildGatewayClass("ketches-envoy-gateway", "gateway.envoyproxy.io/gatewayclass-controller")

	assert.Equal(t, "ketches-envoy-gateway", gatewayClass.Name)
	assert.Equal(t, gatewayv1.GatewayController("gateway.envoyproxy.io/gatewayclass-controller"), gatewayClass.Spec.ControllerName)
}
