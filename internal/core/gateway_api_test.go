package core

import (
	"testing"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
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

func TestBuildGatewayHTTPRoute_UsesRouteIDAndHTTPSListener(t *testing.T) {
	route := BuildGatewayHTTPRoute(GatewayHTTPRouteBuildInput{
		AppSlug:   "demo-app",
		AppID:     "app-1",
		EnvID:     "env-1",
		Namespace: "demo-env",
		GatewayID: "gateway-1",
		RouteID:   "route-1",
		Route: models.GatewayRouteSpec{
			Host:             "secure.example.com",
			ListenerProtocol: "https",
			Path:             "/api",
			PathMatchType:    "PathPrefix",
			Enabled:          true,
			Timeouts: &models.GatewayRouteTimeouts{
				Request:        "30s",
				BackendRequest: "25s",
			},
			Filters: &models.GatewayRouteFilters{
				RequestHeaders: &models.GatewayHeaderModifier{
					Set: []models.GatewayHeaderValue{{Name: "X-App", Value: "demo"}},
				},
			},
		},
		Backends: []GatewayHTTPBackendBuildInput{
			{ServiceName: "demo-app", Port: 8080, Weight: 90},
			{ServiceName: "demo-app-canary", Port: 8080, Weight: 10},
		},
	})

	require.NotNil(t, route)
	assert.Equal(t, "demo-app-route-1", route.Name)
	assert.Equal(t, "demo-env", route.Namespace)
	assert.Equal(t, "app-1", route.Labels[kube.LabelAppID])
	assert.Equal(t, "gateway-1", route.Labels[kube.LabelGatewayID])
	assert.Equal(t, "route-1", route.Labels[kube.LabelGatewayRouteID])
	require.Len(t, route.Spec.ParentRefs, 1)
	require.NotNil(t, route.Spec.ParentRefs[0].SectionName)
	assert.Equal(t, buildSharedGatewayHTTPSListenerName("secure.example.com"), *route.Spec.ParentRefs[0].SectionName)
	require.Len(t, route.Spec.Hostnames, 1)
	assert.Equal(t, gatewayv1.Hostname("secure.example.com"), route.Spec.Hostnames[0])

	require.Len(t, route.Spec.Rules, 1)
	rule := route.Spec.Rules[0]
	require.Len(t, rule.Matches, 1)
	require.NotNil(t, rule.Matches[0].Path)
	require.NotNil(t, rule.Matches[0].Path.Value)
	assert.Equal(t, "/api", *rule.Matches[0].Path.Value)
	require.NotNil(t, rule.Timeouts)
	require.NotNil(t, rule.Timeouts.Request)
	require.NotNil(t, rule.Timeouts.BackendRequest)
	assert.Equal(t, gatewayv1.Duration("30s"), *rule.Timeouts.Request)
	assert.Equal(t, gatewayv1.Duration("25s"), *rule.Timeouts.BackendRequest)
	require.Len(t, rule.Filters, 1)
	require.NotNil(t, rule.Filters[0].RequestHeaderModifier)
	assert.Equal(t, gatewayv1.HTTPHeaderName("X-App"), rule.Filters[0].RequestHeaderModifier.Set[0].Name)

	require.Len(t, rule.BackendRefs, 2)
	assert.Equal(t, gatewayv1.ObjectName("demo-app"), rule.BackendRefs[0].Name)
	require.NotNil(t, rule.BackendRefs[0].Weight)
	assert.Equal(t, int32(90), *rule.BackendRefs[0].Weight)
	assert.Equal(t, gatewayv1.ObjectName("demo-app-canary"), rule.BackendRefs[1].Name)
	require.NotNil(t, rule.BackendRefs[1].Weight)
	assert.Equal(t, int32(10), *rule.BackendRefs[1].Weight)
}
