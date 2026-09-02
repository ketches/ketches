package core

import (
	"context"
	"testing"

	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"
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

func TestCleanupStaleSharedGatewayTLSSecretsPreservesLiveGatewayReferences(t *testing.T) {
	const (
		referencedSecret = "ketches-cert-new"
		staleSecret      = "ketches-cert-stale"
	)
	managedTLSSecret := func(name string) *corev1.Secret {
		return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: SharedGatewayNamespace(),
			Labels: map[string]string{
				kube.LabelManagedBy: "true",
				kube.LabelComponent: "gateway-tls",
			},
		}}
	}

	client := fake.NewSimpleClientset(
		managedTLSSecret(referencedSecret),
		managedTLSSecret(staleSecret),
	)
	gateway := BuildSharedGateway("ketches", []sharedGatewayHTTPSListener{{
		Name:                  "https-live",
		Hostname:              "live.example.com",
		CertificateSecretName: referencedSecret,
	}})
	gwClient := gatewayfake.NewSimpleClientset()
	_, err := gwClient.GatewayV1().Gateways(SharedGatewayNamespace()).Create(context.Background(), gateway, metav1.CreateOptions{})
	require.NoError(t, err)

	// The desired slice intentionally represents an obsolete database snapshot.
	// The live Gateway reference must win over cleanup from that snapshot.
	require.NoError(t, cleanupStaleSharedGatewayTLSSecrets(
		context.Background(),
		client,
		gwClient,
		nil,
	))

	_, err = client.CoreV1().Secrets(SharedGatewayNamespace()).Get(context.Background(), referencedSecret, metav1.GetOptions{})
	require.NoError(t, err)
	_, err = client.CoreV1().Secrets(SharedGatewayNamespace()).Get(context.Background(), staleSecret, metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err), "an unreferenced stale Secret should be removed")
}

func TestCleanupStaleSharedGatewayTLSSecretsKeepsSecretsWhenGatewayIsMissing(t *testing.T) {
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      "ketches-cert-orphan",
		Namespace: SharedGatewayNamespace(),
		Labels: map[string]string{
			kube.LabelManagedBy: "true",
			kube.LabelComponent: "gateway-tls",
		},
	}}
	client := fake.NewSimpleClientset(secret)
	gwClient := gatewayfake.NewSimpleClientset()

	require.NoError(t, cleanupStaleSharedGatewayTLSSecrets(context.Background(), client, gwClient, nil))
	_, err := client.CoreV1().Secrets(SharedGatewayNamespace()).Get(context.Background(), secret.Name, metav1.GetOptions{})
	require.NoError(t, err)
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

func TestCleanupStaleSharedGatewayTLSSecretsDoesNotDeleteUntrustedLegacyPrefix(t *testing.T) {
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      "ketches-cert-not-from-this-cluster",
		Namespace: SharedGatewayNamespace(),
	}, Type: corev1.SecretTypeTLS, Data: map[string][]byte{
		corev1.TLSCertKey:       []byte("certificate"),
		corev1.TLSPrivateKeyKey: []byte("private-key"),
	}}
	client := fake.NewSimpleClientset(secret)
	gwClient := gatewayfake.NewSimpleClientset()
	_, err := gwClient.GatewayV1().Gateways(SharedGatewayNamespace()).Create(
		context.Background(),
		BuildSharedGateway("ketches", nil),
		metav1.CreateOptions{},
	)
	require.NoError(t, err)

	require.NoError(t, cleanupStaleSharedGatewayTLSSecrets(context.Background(), client, gwClient, nil))
	_, err = client.CoreV1().Secrets(SharedGatewayNamespace()).Get(context.Background(), secret.Name, metav1.GetOptions{})
	require.NoError(t, err, "an unlabelled Secret must not be trusted solely because of its name")
}

func TestCleanupStaleSharedGatewayTLSSecretsAllowsValidatedLegacyCertificate(t *testing.T) {
	const certificateID = "certificate-current"
	secretName := sharedGatewayTLSSecretName(certificateID)
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      secretName,
		Namespace: SharedGatewayNamespace(),
	}, Type: corev1.SecretTypeTLS, Data: map[string][]byte{
		corev1.TLSCertKey:       []byte("certificate"),
		corev1.TLSPrivateKeyKey: []byte("private-key"),
	}}
	client := fake.NewSimpleClientset(secret)
	gwClient := gatewayfake.NewSimpleClientset()
	_, err := gwClient.GatewayV1().Gateways(SharedGatewayNamespace()).Create(
		context.Background(),
		BuildSharedGateway("ketches", nil),
		metav1.CreateOptions{},
	)
	require.NoError(t, err)

	require.NoError(t, cleanupStaleSharedGatewayTLSSecretsWithLegacyNames(
		context.Background(),
		client,
		gwClient,
		nil,
		map[string]struct{}{secretName: {}},
	))
	_, err = client.CoreV1().Secrets(SharedGatewayNamespace()).Get(context.Background(), secretName, metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err), "a validated legacy Ketches Secret should be removed")
}

func TestTrustedLegacySharedGatewayTLSSecretRequiresTLSMaterial(t *testing.T) {
	allowed := map[string]struct{}{sharedGatewayTLSSecretName("certificate-current"): {}}
	for _, test := range []struct {
		name   string
		secret *corev1.Secret
	}{
		{
			name: "opaque secret",
			secret: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
				Name: sharedGatewayTLSSecretName("certificate-current"),
			}, Type: corev1.SecretTypeOpaque},
		},
		{
			name: "missing key",
			secret: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
				Name: sharedGatewayTLSSecretName("certificate-current"),
			}, Type: corev1.SecretTypeTLS, Data: map[string][]byte{
				corev1.TLSCertKey: []byte("certificate"),
			}},
		},
		{
			name: "different certificate id",
			secret: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
				Name: sharedGatewayTLSSecretName("certificate-other"),
			}, Type: corev1.SecretTypeTLS, Data: map[string][]byte{
				corev1.TLSCertKey:       []byte("certificate"),
				corev1.TLSPrivateKeyKey: []byte("private-key"),
			}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			assert.False(t, trustedLegacySharedGatewayTLSSecret(test.secret, allowed))
		})
	}
}
