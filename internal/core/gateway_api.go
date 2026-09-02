package core

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/secrets"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

const (
	// gatewayAPIGroup is the API group for the Gateway API.
	gatewayAPIGroup = "gateway.networking.k8s.io"
	// gatewayAPIVersion is the version used for detection and resource creation.
	gatewayAPIVersion = "v1"
	// defaultGatewayClassName is used when the cluster has no explicit default provider.
	defaultGatewayClassName = "eg"
	// sharedGatewayNamespace is the fixed namespace that hosts the single shared Gateway.
	sharedGatewayNamespace = "ketches-system"
	// sharedGatewayName is the canonical name of the single shared Gateway.
	sharedGatewayName = "ketches-shared-gateway"
)

var sharedGatewayReconcileLocks sync.Map

type sharedGatewayHTTPSListener struct {
	Name                  gatewayv1.SectionName
	Hostname              gatewayv1.Hostname
	CertificateSecretName string
}

type clusterHTTPSGatewayBinding struct {
	Domain string
	CertID *string
	EnvID  string
}

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

	return defaultGatewayClassName, nil
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
// It always exposes one HTTP listener on port 80 and optionally adds HTTPS
// listeners for domains that have a selected TLS certificate.
func BuildSharedGateway(gatewayClassName string, httpsListeners []sharedGatewayHTTPSListener) *gatewayv1.Gateway {
	resolvedGatewayClassName := gatewayv1.ObjectName(gatewayClassName)
	listeners := []gatewayv1.Listener{buildSharedGatewayHTTPListener()}
	listeners = append(listeners, buildSharedGatewayHTTPSListeners(httpsListeners)...)

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
			Listeners:        listeners,
		},
	}
}

func buildSharedGatewayHTTPListener() gatewayv1.Listener {
	httpPort := gatewayv1.PortNumber(80)
	httpProtocol := gatewayv1.HTTPProtocolType

	return gatewayv1.Listener{
		Name:          gatewayv1.SectionName("http"),
		Port:          httpPort,
		Protocol:      httpProtocol,
		AllowedRoutes: buildSharedGatewayAllowedRoutes(),
	}
}

func buildSharedGatewayHTTPSListeners(items []sharedGatewayHTTPSListener) []gatewayv1.Listener {
	if len(items) == 0 {
		return nil
	}

	sortedItems := append([]sharedGatewayHTTPSListener(nil), items...)
	sort.Slice(sortedItems, func(i, j int) bool {
		return sortedItems[i].Hostname < sortedItems[j].Hostname
	})

	listeners := make([]gatewayv1.Listener, 0, len(sortedItems))
	for _, item := range sortedItems {
		hostname := item.Hostname
		httpsPort := gatewayv1.PortNumber(443)
		httpsProtocol := gatewayv1.HTTPSProtocolType
		tlsMode := gatewayv1.TLSModeTerminate
		secretGroup := gatewayv1.Group("")
		secretKind := gatewayv1.Kind("Secret")
		secretNamespace := gatewayv1.Namespace(SharedGatewayNamespace())

		listeners = append(listeners, gatewayv1.Listener{
			Name:     item.Name,
			Hostname: &hostname,
			Port:     httpsPort,
			Protocol: httpsProtocol,
			TLS: &gatewayv1.ListenerTLSConfig{
				Mode: &tlsMode,
				CertificateRefs: []gatewayv1.SecretObjectReference{
					{
						Group:     &secretGroup,
						Kind:      &secretKind,
						Name:      gatewayv1.ObjectName(item.CertificateSecretName),
						Namespace: &secretNamespace,
					},
				},
			},
			AllowedRoutes: buildSharedGatewayAllowedRoutes(),
		})
	}

	return listeners
}

func buildSharedGatewayAllowedRoutes() *gatewayv1.AllowedRoutes {
	fromAll := gatewayv1.NamespacesFromAll
	return &gatewayv1.AllowedRoutes{
		Namespaces: &gatewayv1.RouteNamespaces{
			From: &fromAll,
		},
	}
}

// EnsureSharedGateway creates or updates the single shared Gateway resource in the
// cluster. It is a no-op when the cluster does not have the Gateway API CRDs.
func EnsureSharedGateway(ctx context.Context, clusterID string) error {
	lock := sharedGatewayReconcileLock(clusterID)
	lock.Lock()
	defer lock.Unlock()

	return withDatabaseReconcileLock(ctx, "gateway:"+clusterID, func() error {
		return ensureSharedGateway(ctx, clusterID)
	})
}

func ensureSharedGateway(ctx context.Context, clusterID string) error {
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

	client, err := kube.GlobalClusterStore.GetClient(clusterID)
	if err != nil {
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

	httpsListeners, tlsSecrets, err := loadSharedGatewayHTTPSMaterial(clusterID)
	if err != nil {
		return err
	}
	for _, secret := range tlsSecrets {
		if err := ensureSharedGatewayTLSSecret(ctx, client, secret); err != nil {
			return err
		}
	}

	desired := BuildSharedGateway(gatewayClassName, httpsListeners)

	if err := applySharedGateway(ctx, gwClient, desired); err != nil {
		return err
	}

	// Include a second database snapshot in cleanup protection. A newer request
	// may have committed its TLS choice while this reconcile was updating the
	// Gateway, and deleting that Secret would make the following reconcile fail.
	_, latestTLSSecrets, err := loadSharedGatewayHTTPSMaterial(clusterID)
	if err != nil {
		return err
	}
	legacySecretNames, err := loadLegacySharedGatewayTLSSecretNames(clusterID)
	if err != nil {
		return err
	}
	return cleanupStaleSharedGatewayTLSSecretsWithLegacyNames(
		ctx,
		client,
		gwClient,
		mergeSharedGatewayTLSSecrets(tlsSecrets, latestTLSSecrets),
		legacySecretNames,
	)
}

func sharedGatewayReconcileLock(clusterID string) *sync.Mutex {
	lock := &sync.Mutex{}
	actual, _ := sharedGatewayReconcileLocks.LoadOrStore(clusterID, lock)
	return actual.(*sync.Mutex)
}

func applySharedGateway(ctx context.Context, client gatewayclient.Interface, desired *gatewayv1.Gateway) error {
	return retry.OnError(retry.DefaultRetry, func(err error) bool {
		return apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err)
	}, func() error {
		existing, err := client.GatewayV1().Gateways(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.GatewayV1().Gateways(desired.Namespace).Create(ctx, desired.DeepCopy(), metav1.CreateOptions{})
			return err
		}
		if err != nil {
			return err
		}

		updated := desired.DeepCopy()
		updated.ResourceVersion = existing.ResourceVersion
		_, err = client.GatewayV1().Gateways(updated.Namespace).Update(ctx, updated, metav1.UpdateOptions{})
		return err
	})
}

func loadSharedGatewayHTTPSMaterial(clusterID string) ([]sharedGatewayHTTPSListener, []*corev1.Secret, error) {
	var bindings []clusterHTTPSGatewayBinding
	err := db.DB.Table("app_gateway_http_routes r").
		Select("r.host AS domain, r.cert_id, a.env_id").
		Joins("JOIN app_gateways ag ON ag.id = r.app_gateway_id").
		Joins("JOIN apps a ON a.id = ag.app_id").
		Joins("JOIN envs e ON e.id = a.env_id").
		Where("e.cluster_id = ? AND a.deleted_at IS NULL AND e.deleted_at IS NULL AND r.enabled = ? AND LOWER(r.listener_protocol) = ?", clusterID, true, "https").
		Scan(&bindings).Error
	if err != nil {
		return nil, nil, err
	}
	if len(bindings) == 0 {
		return nil, nil, nil
	}

	certIDs := make([]string, 0, len(bindings))
	seenCertIDs := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		if binding.CertID == nil {
			continue
		}
		certID := strings.TrimSpace(*binding.CertID)
		if certID == "" {
			continue
		}
		if _, exists := seenCertIDs[certID]; exists {
			continue
		}
		seenCertIDs[certID] = struct{}{}
		certIDs = append(certIDs, certID)
	}

	certificatesByID := make(map[string]entities.Certificate, len(certIDs))
	if len(certIDs) > 0 {
		var certificates []entities.Certificate
		if err := db.DB.Where("cluster_id = ? AND id IN ?", clusterID, certIDs).Find(&certificates).Error; err != nil {
			return nil, nil, err
		}
		for _, certificate := range certificates {
			plaintextKey, err := secrets.DecryptStringCompatible(certificate.Key)
			if err != nil {
				return nil, nil, app.WrapErrorf(err, "decrypt certificate %s private key: %w", certificate.ID, err)
			}
			certificate.Key = plaintextKey
			certificatesByID[certificate.ID] = certificate
		}
	}

	listenerByDomain := make(map[string]sharedGatewayHTTPSListener)
	secretByName := make(map[string]*corev1.Secret)
	for _, binding := range bindings {
		domain := strings.TrimSpace(binding.Domain)
		if domain == "" || binding.CertID == nil {
			continue
		}

		certID := strings.TrimSpace(*binding.CertID)
		if certID == "" {
			continue
		}

		certificate, ok := certificatesByID[certID]
		if !ok || !certificateAvailableToEnv(&certificate, binding.EnvID) {
			continue
		}

		normalizedDomain := strings.ToLower(domain)
		secretName := sharedGatewayTLSSecretName(certificate.ID)
		listener := sharedGatewayHTTPSListener{
			Name:                  buildSharedGatewayHTTPSListenerName(normalizedDomain),
			Hostname:              gatewayv1.Hostname(normalizedDomain),
			CertificateSecretName: secretName,
		}
		if existing, ok := listenerByDomain[normalizedDomain]; ok && existing.CertificateSecretName != listener.CertificateSecretName {
			return nil, nil, app.NewErrorf("multiple HTTPS gateways use domain %q with different certificates", normalizedDomain)
		}
		listenerByDomain[normalizedDomain] = listener
		if _, ok := secretByName[secretName]; !ok {
			secretByName[secretName] = buildSharedGatewayTLSSecret(secretName, &certificate)
		}
	}

	listeners := make([]sharedGatewayHTTPSListener, 0, len(listenerByDomain))
	for _, listener := range listenerByDomain {
		listeners = append(listeners, listener)
	}
	secrets := make([]*corev1.Secret, 0, len(secretByName))
	for _, secret := range secretByName {
		secrets = append(secrets, secret)
	}

	return listeners, secrets, nil
}

// loadLegacySharedGatewayTLSSecretNames returns names that older Ketches
// versions may have created without ownership labels. The certificate ID must
// belong to this cluster before the name can be considered safe to remove.
func loadLegacySharedGatewayTLSSecretNames(clusterID string) (map[string]struct{}, error) {
	var certificateIDs []string
	if err := db.DB.Model(&entities.Certificate{}).
		Where("cluster_id = ?", clusterID).
		Pluck("id", &certificateIDs).Error; err != nil {
		return nil, err
	}

	result := make(map[string]struct{}, len(certificateIDs))
	for _, certificateID := range certificateIDs {
		certificateID = strings.TrimSpace(certificateID)
		if certificateID == "" {
			continue
		}
		result[sharedGatewayTLSSecretName(certificateID)] = struct{}{}
	}
	return result, nil
}

func buildSharedGatewayHTTPSListenerName(hostname string) gatewayv1.SectionName {
	normalized := strings.ToLower(strings.TrimSpace(hostname))
	sanitized := make([]rune, 0, len(normalized))
	lastWasDash := false
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sanitized = append(sanitized, r)
			lastWasDash = false
			continue
		}
		if !lastWasDash {
			sanitized = append(sanitized, '-')
			lastWasDash = true
		}
	}

	trimmed := strings.Trim(string(sanitized), "-")
	if trimmed == "" {
		trimmed = "host"
	}

	sum := sha1.Sum([]byte(normalized))
	hashSuffix := hex.EncodeToString(sum[:])[:8]
	if len(trimmed) > 48 {
		trimmed = trimmed[:48]
		trimmed = strings.Trim(trimmed, "-")
		if trimmed == "" {
			trimmed = "host"
		}
	}

	return gatewayv1.SectionName(fmt.Sprintf("https-%s-%s", trimmed, hashSuffix))
}

func sharedGatewayTLSSecretName(certificateID string) string {
	return fmt.Sprintf("ketches-cert-%s", certificateID)
}

func buildSharedGatewayTLSSecret(secretName string, certificate *entities.Certificate) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: SharedGatewayNamespace(),
			Labels: map[string]string{
				kube.LabelManagedBy:     "true",
				kube.LabelComponent:     "gateway-tls",
				kube.LabelCertificateID: certificate.ID,
			},
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte(certificate.Cert),
			corev1.TLSPrivateKeyKey: []byte(certificate.Key),
		},
	}
}

func ensureSharedGatewayTLSSecret(ctx context.Context, client *kubernetes.Clientset, desired *corev1.Secret) error {
	return retry.OnError(retry.DefaultRetry, func(err error) bool {
		return apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err)
	}, func() error {
		existing, err := client.CoreV1().Secrets(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.CoreV1().Secrets(desired.Namespace).Create(ctx, desired.DeepCopy(), metav1.CreateOptions{})
			return err
		}
		if err != nil {
			return err
		}

		updated := desired.DeepCopy()
		updated.ResourceVersion = existing.ResourceVersion
		_, err = client.CoreV1().Secrets(updated.Namespace).Update(ctx, updated, metav1.UpdateOptions{})
		return err
	})
}

func mergeSharedGatewayTLSSecrets(groups ...[]*corev1.Secret) []*corev1.Secret {
	byName := make(map[string]*corev1.Secret)
	for _, group := range groups {
		for _, secret := range group {
			if secret == nil || strings.TrimSpace(secret.Name) == "" {
				continue
			}
			byName[secret.Name] = secret
		}
	}
	result := make([]*corev1.Secret, 0, len(byName))
	for _, secret := range byName {
		result = append(result, secret)
	}
	return result
}

func cleanupStaleSharedGatewayTLSSecrets(
	ctx context.Context,
	client kubernetes.Interface,
	gwClient gatewayclient.Interface,
	desired []*corev1.Secret,
) error {
	return cleanupStaleSharedGatewayTLSSecretsWithLegacyNames(ctx, client, gwClient, desired, nil)
}

func cleanupStaleSharedGatewayTLSSecretsWithLegacyNames(
	ctx context.Context,
	client kubernetes.Interface,
	gwClient gatewayclient.Interface,
	desired []*corev1.Secret,
	legacySecretNames map[string]struct{},
) error {
	desiredNames := make(map[string]struct{}, len(desired))
	for _, secret := range desired {
		if secret == nil || strings.TrimSpace(secret.Name) == "" {
			continue
		}
		desiredNames[secret.Name] = struct{}{}
	}
	secrets, err := client.CoreV1().Secrets(SharedGatewayNamespace()).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for i := range secrets.Items {
		secret := &secrets.Items[i]
		managedTLSSecret := secret.Labels[kube.LabelManagedBy] == "true" && secret.Labels[kube.LabelComponent] == "gateway-tls"
		legacyTLSSecret := trustedLegacySharedGatewayTLSSecret(secret, legacySecretNames)
		if !managedTLSSecret && !legacyTLSSecret {
			continue
		}
		if _, ok := desiredNames[secret.Name]; ok {
			continue
		}

		// Re-read the live Gateway immediately before each delete. Cleanup is
		// intentionally fail-safe: a Secret that is still referenced, or whose
		// reference state cannot be established, is left for a later reconcile.
		gateway, err := gwClient.GatewayV1().Gateways(SharedGatewayNamespace()).Get(
			ctx,
			SharedGatewayName(),
			metav1.GetOptions{},
		)
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if sharedGatewayReferencesTLSSecret(gateway, secret.Name) {
			continue
		}
		if err := client.CoreV1().Secrets(secret.Namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func trustedLegacySharedGatewayTLSSecret(secret *corev1.Secret, allowedNames map[string]struct{}) bool {
	if secret == nil || len(allowedNames) == 0 || !strings.HasPrefix(secret.Name, "ketches-cert-") {
		return false
	}
	if _, ok := allowedNames[secret.Name]; !ok {
		return false
	}
	if secret.Type != corev1.SecretTypeTLS {
		return false
	}
	return len(secret.Data[corev1.TLSCertKey]) > 0 && len(secret.Data[corev1.TLSPrivateKeyKey]) > 0
}

func sharedGatewayReferencesTLSSecret(gateway *gatewayv1.Gateway, secretName string) bool {
	if gateway == nil || strings.TrimSpace(secretName) == "" {
		return false
	}
	for _, listener := range gateway.Spec.Listeners {
		if listener.Protocol != gatewayv1.HTTPSProtocolType || listener.TLS == nil {
			continue
		}
		for _, certificateRef := range listener.TLS.CertificateRefs {
			if certificateRef.Group != nil && *certificateRef.Group != gatewayv1.Group("") {
				continue
			}
			if certificateRef.Kind != nil && *certificateRef.Kind != gatewayv1.Kind("Secret") {
				continue
			}
			if certificateRef.Namespace != nil && string(*certificateRef.Namespace) != gateway.Namespace {
				continue
			}
			if string(certificateRef.Name) == secretName {
				return true
			}
		}
	}
	return false
}

func certificateAvailableToEnv(certificate *entities.Certificate, envID string) bool {
	if certificate == nil {
		return false
	}
	if certificate.Scope == "cluster" {
		return true
	}
	if certificate.Scope != "env" || certificate.EnvID == nil {
		return false
	}
	return strings.TrimSpace(*certificate.EnvID) == strings.TrimSpace(envID)
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
