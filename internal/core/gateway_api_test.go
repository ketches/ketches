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
	gateway := BuildSharedGateway()

	require.Equal(t, SharedGatewayName(), gateway.Name)
	require.Equal(t, SharedGatewayNamespace(), gateway.Namespace)
	require.Len(t, gateway.Spec.Listeners, 1)

	listener := gateway.Spec.Listeners[0]
	assert.Equal(t, gatewayv1.SectionName("http"), listener.Name)
	assert.Equal(t, gatewayv1.PortNumber(80), listener.Port)
	assert.Equal(t, gatewayv1.HTTPProtocolType, listener.Protocol)
	require.NotNil(t, listener.AllowedRoutes)
	require.NotNil(t, listener.AllowedRoutes.Namespaces)
	require.NotNil(t, listener.AllowedRoutes.Namespaces.From)
	assert.Equal(t, gatewayv1.NamespacesFromAll, *listener.AllowedRoutes.Namespaces.From)
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
	assert.Equal(t, gatewayv1.ObjectName(SharedGatewayName()), parentRef.Name)
	assert.Equal(t, gatewayv1.Namespace(SharedGatewayNamespace()), *parentRef.Namespace)
	assert.Equal(t, gatewayv1.SectionName("http"), *parentRef.SectionName)
}
