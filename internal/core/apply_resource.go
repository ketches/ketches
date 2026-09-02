package core

import (
	"context"
	"maps"
	"path/filepath"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

func ApplyApp(ctx context.Context, appCtx *models.AppContext) error {
	var clusterID string
	if err := withAppReconcileLockContext(ctx, appCtx, func(latest *models.AppContext) error {
		clusterID = latest.EnvContext.Env.ClusterID
		return applyApp(ctx, latest)
	}); err != nil {
		return err
	}
	return syncSharedGatewayForApp(ctx, clusterID)
}

func applyApp(ctx context.Context, appCtx *models.AppContext) error {
	if err := validateAppVolumePolicy(appCtx); err != nil {
		return err
	}
	metadata := &AppMetadata{AppContext: appCtx}
	configRevision, err := metadata.BuildConfigRevision()
	if err != nil {
		return err
	}
	metadata.configRevision = configRevision

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	if err := ApplyResource(ctx, client, metadata.BuildNamespace()); err != nil {
		return err
	}

	hasConfigMap := metadata.hasNonSecretConfigFiles()
	hasConfigSecret := metadata.hasSecretConfigFiles()
	hasEnvSecret := metadata.hasSecretEnvVars()

	// Create/update resources that the desired workload may reference before
	// changing the workload. Obsolete resources are removed only after the new
	// Pod template has been accepted, so a failed rollout cannot leave a live
	// workload pointing at a deleted ConfigMap or Secret.
	if hasConfigMap {
		if err := applyConfigMap(ctx, client, metadata.BuildConfigMap()); err != nil {
			return err
		}
	}
	if hasConfigSecret {
		secret, err := metadata.BuildConfigSecret()
		if err != nil {
			return err
		}
		if err := applySecret(ctx, client, secret); err != nil {
			return err
		}
	}
	if hasEnvSecret {
		secret, err := metadata.BuildEnvSecret()
		if err != nil {
			return err
		}
		if err := applySecret(ctx, client, secret); err != nil {
			return err
		}
	}

	if appCtx.App.RegistryUsername != "" {
		secret, err := metadata.BuildRegistrySecret()
		if err != nil {
			return err
		}
		if err := ApplyResource(ctx, client, secret); err != nil {
			return err
		}
	} else if err := deleteSecretIfExists(ctx, client, appCtx.EnvContext.Env.ClusterNamespace, appCtx.App.Slug+"-registry"); err != nil {
		return err
	}

	for _, volume := range appCtx.Volumes {
		if volume.VolumeType != app.VolumeTypePVC {
			continue
		}
		switch appCtx.App.AppType {
		case app.AppTypeDeployment:
			if err := applyPersistentVolumeClaim(ctx, client, metadata.BuildPVC(volume)); err != nil {
				return err
			}
		case app.AppTypeStatefulSet:
			// StatefulSet controllers create claims from volumeClaimTemplates.
		default:
			return app.NewErrorf("unsupported app type '%s' for volume '%s'", appCtx.App.AppType, volume.Slug)
		}
	}

	if appCtx.App.AppType == app.AppTypeStatefulSet {
		if err := ApplyResource(ctx, client, metadata.BuildStatefulSet()); err != nil {
			return err
		}
		if err := deleteDeploymentIfExists(ctx, client, appCtx.EnvContext.Env.ClusterNamespace, appCtx.App.Slug); err != nil {
			return err
		}
	} else {
		if err := ApplyResource(ctx, client, metadata.BuildDeployment()); err != nil {
			return err
		}
		if err := deleteStatefulSetIfExists(ctx, client, appCtx.EnvContext.Env.ClusterNamespace, appCtx.App.Slug); err != nil {
			return err
		}
	}

	// Now that the workload no longer references obsolete configuration
	// resources, it is safe to remove them. Keeping these deletes after the
	// workload update prevents broken SecretKeyRef/volume references on errors.
	if !hasConfigMap {
		if err := deleteConfigMapIfExists(ctx, client, appCtx.EnvContext.Env.ClusterNamespace, appCtx.App.Slug+"-config"); err != nil {
			return err
		}
	}
	if !hasConfigSecret {
		if err := deleteSecretIfExists(ctx, client, appCtx.EnvContext.Env.ClusterNamespace, appCtx.App.Slug+"-config-secret"); err != nil {
			return err
		}
	}
	if !hasEnvSecret {
		if err := deleteSecretIfExists(ctx, client, appCtx.EnvContext.Env.ClusterNamespace, appCtx.App.Slug+"-env-secret"); err != nil {
			return err
		}
	}

	if appCtx.AutoScaling != nil {
		if err := ApplyResource(ctx, client, metadata.BuildHorizontalPodAutoscaler()); err != nil {
			return err
		}
	} else if err := deleteHPAIfExists(ctx, client, appCtx.EnvContext.Env.ClusterNamespace, appCtx.App.Slug); err != nil {
		return err
	}

	return syncAppGatewaysToK8s(ctx, appCtx)
}

func validateAppVolumePolicy(appCtx *models.AppContext) error {
	for _, volume := range appCtx.Volumes {
		if volume.VolumeType != app.VolumeTypeHostPath {
			continue
		}
		if !app.Config.AllowHostPathVolumes {
			return app.NewErrorf("hostPath volume %q is disabled by server policy", volume.Slug)
		}
		cleanPath := filepath.Clean(strings.TrimSpace(volume.HostPath))
		if !filepath.IsAbs(cleanPath) || cleanPath == "/" {
			return app.NewErrorf("hostPath volume %q must use an absolute path other than root", volume.Slug)
		}
	}
	return nil
}

// ApplyResource creates or updates an application-owned Kubernetes resource.
// Updates always use the latest resourceVersion and preserve Service fields
// allocated by the API server.
func retryOnKubernetesConflict(fn func() error) error {
	return retry.OnError(retry.DefaultRetry, func(err error) bool {
		return apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err)
	}, fn)
}

func ApplyResource(ctx context.Context, client kubernetes.Interface, res runtime.Object) error {
	switch desired := res.(type) {
	case *corev1.Namespace:
		return applyNamespace(ctx, client, desired)
	case *corev1.ConfigMap:
		return applyConfigMap(ctx, client, desired)
	case *corev1.Secret:
		return applySecret(ctx, client, desired)
	case *appsv1.Deployment:
		return applyDeployment(ctx, client, desired)
	case *appsv1.StatefulSet:
		return applyStatefulSet(ctx, client, desired)
	case *autoscalingv2.HorizontalPodAutoscaler:
		return applyHorizontalPodAutoscaler(ctx, client, desired)
	case *corev1.Service:
		return applyService(ctx, client, desired)
	default:
		return app.NewErrorf("unsupported Kubernetes resource type %T", res)
	}
}

func applyNamespace(ctx context.Context, client kubernetes.Interface, desired *corev1.Namespace) error {
	return retryOnKubernetesConflict(func() error {
		existing, err := client.CoreV1().Namespaces().Get(ctx, desired.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.CoreV1().Namespaces().Create(ctx, desired.DeepCopy(), metav1.CreateOptions{})
			return err
		}
		if err != nil {
			return err
		}
		if err := validateNamespaceOwnership(existing, desired.Labels); err != nil {
			return err
		}

		updated := existing.DeepCopy()
		if updated.Labels == nil {
			updated.Labels = make(map[string]string, len(desired.Labels))
		}
		maps.Copy(updated.Labels, desired.Labels)
		_, err = client.CoreV1().Namespaces().Update(ctx, updated, metav1.UpdateOptions{})
		return err
	})
}

func applyConfigMap(ctx context.Context, client kubernetes.Interface, desired *corev1.ConfigMap) error {
	return retryOnKubernetesConflict(func() error {
		existing, err := client.CoreV1().ConfigMaps(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.CoreV1().ConfigMaps(desired.Namespace).Create(ctx, desired.DeepCopy(), metav1.CreateOptions{})
			return err
		}
		if err != nil {
			return err
		}

		updated := desired.DeepCopy()
		updated.ResourceVersion = existing.ResourceVersion
		_, err = client.CoreV1().ConfigMaps(updated.Namespace).Update(ctx, updated, metav1.UpdateOptions{})
		return err
	})
}

func applySecret(ctx context.Context, client kubernetes.Interface, desired *corev1.Secret) error {
	return retryOnKubernetesConflict(func() error {
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

func applyDeployment(ctx context.Context, client kubernetes.Interface, desired *appsv1.Deployment) error {
	return retryOnKubernetesConflict(func() error {
		existing, err := client.AppsV1().Deployments(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.AppsV1().Deployments(desired.Namespace).Create(ctx, desired.DeepCopy(), metav1.CreateOptions{})
			return err
		}
		if err != nil {
			return err
		}

		updated := desired.DeepCopy()
		updated.ResourceVersion = existing.ResourceVersion
		_, err = client.AppsV1().Deployments(updated.Namespace).Update(ctx, updated, metav1.UpdateOptions{})
		return err
	})
}

func applyStatefulSet(ctx context.Context, client kubernetes.Interface, desired *appsv1.StatefulSet) error {
	return retryOnKubernetesConflict(func() error {
		existing, err := client.AppsV1().StatefulSets(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.AppsV1().StatefulSets(desired.Namespace).Create(ctx, desired.DeepCopy(), metav1.CreateOptions{})
			return err
		}
		if err != nil {
			return err
		}
		if err := validateStatefulSetVolumeClaimTemplates(existing, desired); err != nil {
			return err
		}

		updated := desired.DeepCopy()
		updated.ResourceVersion = existing.ResourceVersion
		// VolumeClaimTemplates are immutable after StatefulSet creation. Preserve
		// the API-server representation (including admission defaults) after the
		// compatibility check so unrelated updates cannot resend a divergent form.
		updated.Spec.VolumeClaimTemplates = copyPersistentVolumeClaimTemplates(existing.Spec.VolumeClaimTemplates)
		_, err = client.AppsV1().StatefulSets(updated.Namespace).Update(ctx, updated, metav1.UpdateOptions{})
		return err
	})
}

func updateAppConfigRevision(ctx context.Context, client kubernetes.Interface, appType, namespace, name, revision string) error {
	return updateAppConfigWorkload(ctx, client, appType, namespace, name, nil, revision)
}

// updateAppConfigWorkload updates the configuration-related portion of a
// workload template. Keeping the existing non-configuration fields avoids
// turning a config-file save into an unrelated application spec overwrite.
func updateAppConfigWorkload(
	ctx context.Context,
	client kubernetes.Interface,
	appType, namespace, name string,
	desiredTemplate *corev1.PodTemplateSpec,
	revision string,
) error {
	return retryOnKubernetesConflict(func() error {
		if appType == app.AppTypeStatefulSet {
			existing, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				// A config file may be created before the application is deployed.
				return nil
			}
			if err != nil {
				return err
			}
			if !reconcileConfigPodTemplate(&existing.Spec.Template, desiredTemplate, revision) {
				return nil
			}
			_, err = client.AppsV1().StatefulSets(namespace).Update(ctx, existing, metav1.UpdateOptions{})
			if apierrors.IsNotFound(err) {
				// The application was removed while the config operation was in
				// flight. There is no workload left to reconcile.
				return nil
			}
			return err
		}

		existing, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// A config file may be created before the application is deployed.
			return nil
		}
		if err != nil {
			return err
		}
		if !reconcileConfigPodTemplate(&existing.Spec.Template, desiredTemplate, revision) {
			return nil
		}
		_, err = client.AppsV1().Deployments(namespace).Update(ctx, existing, metav1.UpdateOptions{})
		if apierrors.IsNotFound(err) {
			// The application was removed while the config operation was in
			// flight. There is no workload left to reconcile.
			return nil
		}
		return err
	})
}

func reconcileConfigPodTemplate(existing, desired *corev1.PodTemplateSpec, revision string) bool {
	changed := setConfigRevision(&existing.ObjectMeta, revision)
	if desired == nil {
		return changed
	}

	desiredConfigVolume, hasConfigVolume := findConfigVolume(desired.Spec.Volumes)
	updatedVolumes := replaceConfigVolume(existing.Spec.Volumes, desiredConfigVolume, hasConfigVolume)
	if !apiequality.Semantic.DeepEqual(existing.Spec.Volumes, updatedVolumes) {
		existing.Spec.Volumes = updatedVolumes
		changed = true
	}

	if reconcileConfigContainerMounts(existing.Spec.InitContainers, desired.Spec.InitContainers, hasConfigVolume) {
		changed = true
	}
	if reconcileConfigContainerMounts(existing.Spec.Containers, desired.Spec.Containers, hasConfigVolume) {
		changed = true
	}
	return changed
}

func findConfigVolume(volumes []corev1.Volume) (*corev1.Volume, bool) {
	for i := range volumes {
		if volumes[i].Name == "config-files" {
			volume := volumes[i].DeepCopy()
			return volume, true
		}
	}
	return nil, false
}

func replaceConfigVolume(existing []corev1.Volume, desired *corev1.Volume, include bool) []corev1.Volume {
	result := make([]corev1.Volume, 0, len(existing)+1)
	inserted := false
	for _, volume := range existing {
		if volume.Name == "config-files" {
			if include && !inserted {
				result = append(result, *desired.DeepCopy())
				inserted = true
			}
			continue
		}
		result = append(result, volume)
	}
	if include && !inserted {
		result = append(result, *desired.DeepCopy())
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func reconcileConfigContainerMounts(existing, desired []corev1.Container, include bool) bool {
	desiredByName := make(map[string][]corev1.VolumeMount, len(desired))
	var fallback []corev1.VolumeMount
	for _, container := range desired {
		mounts := configVolumeMounts(container.VolumeMounts)
		if len(mounts) == 0 {
			continue
		}
		desiredByName[container.Name] = mounts
		if fallback == nil {
			fallback = mounts
		}
	}
	if !include {
		fallback = nil
	}

	changed := false
	for i := range existing {
		container := &existing[i]
		mounts := make([]corev1.VolumeMount, 0, len(container.VolumeMounts)+len(fallback))
		for _, mount := range container.VolumeMounts {
			if mount.Name != "config-files" {
				mounts = append(mounts, mount)
			}
		}
		configMounts := desiredByName[container.Name]
		if configMounts == nil && include {
			configMounts = fallback
		}
		mounts = append(mounts, configMounts...)
		if !apiequality.Semantic.DeepEqual(container.VolumeMounts, mounts) {
			container.VolumeMounts = mounts
			changed = true
		}
	}
	return changed
}

func configVolumeMounts(mounts []corev1.VolumeMount) []corev1.VolumeMount {
	result := make([]corev1.VolumeMount, 0, len(mounts))
	for _, mount := range mounts {
		if mount.Name == "config-files" {
			result = append(result, mount)
		}
	}
	return result
}

func setConfigRevision(metadata *metav1.ObjectMeta, revision string) bool {
	if metadata.Annotations != nil && metadata.Annotations[configRevisionAnnotation] == revision {
		return false
	}
	if metadata.Annotations == nil {
		metadata.Annotations = make(map[string]string)
	}
	metadata.Annotations[configRevisionAnnotation] = revision
	return true
}

func applyHorizontalPodAutoscaler(ctx context.Context, client kubernetes.Interface, desired *autoscalingv2.HorizontalPodAutoscaler) error {
	return retryOnKubernetesConflict(func() error {
		existing, err := client.AutoscalingV2().HorizontalPodAutoscalers(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.AutoscalingV2().HorizontalPodAutoscalers(desired.Namespace).Create(ctx, desired.DeepCopy(), metav1.CreateOptions{})
			return err
		}
		if err != nil {
			return err
		}

		updated := desired.DeepCopy()
		updated.ResourceVersion = existing.ResourceVersion
		_, err = client.AutoscalingV2().HorizontalPodAutoscalers(updated.Namespace).Update(ctx, updated, metav1.UpdateOptions{})
		return err
	})
}

func applyService(ctx context.Context, client kubernetes.Interface, desired *corev1.Service) error {
	return retryOnKubernetesConflict(func() error {
		existing, err := client.CoreV1().Services(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.CoreV1().Services(desired.Namespace).Create(ctx, desired.DeepCopy(), metav1.CreateOptions{})
			return err
		}
		if err != nil {
			return err
		}

		updated := desired.DeepCopy()
		updated.ResourceVersion = existing.ResourceVersion
		preserveServiceAllocatedFields(updated, existing)
		_, err = client.CoreV1().Services(updated.Namespace).Update(ctx, updated, metav1.UpdateOptions{})
		return err
	})
}

func preserveServiceAllocatedFields(desired, existing *corev1.Service) {
	desired.Spec.ClusterIP = existing.Spec.ClusterIP
	desired.Spec.ClusterIPs = existing.Spec.ClusterIPs
	desired.Spec.IPFamilies = existing.Spec.IPFamilies
	desired.Spec.IPFamilyPolicy = existing.Spec.IPFamilyPolicy
	desired.Spec.HealthCheckNodePort = existing.Spec.HealthCheckNodePort
	if desired.Spec.LoadBalancerClass == nil {
		desired.Spec.LoadBalancerClass = existing.Spec.LoadBalancerClass
	}
	if desired.Spec.AllocateLoadBalancerNodePorts == nil {
		desired.Spec.AllocateLoadBalancerNodePorts = existing.Spec.AllocateLoadBalancerNodePorts
	}

	for i := range desired.Spec.Ports {
		if desired.Spec.Ports[i].NodePort != 0 {
			continue
		}
		for _, existingPort := range existing.Spec.Ports {
			if desired.Spec.Ports[i].Name == existingPort.Name &&
				desired.Spec.Ports[i].Protocol == existingPort.Protocol {
				desired.Spec.Ports[i].NodePort = existingPort.NodePort
				break
			}
		}
	}
}

func applyPersistentVolumeClaim(ctx context.Context, client kubernetes.Interface, desired *corev1.PersistentVolumeClaim) error {
	return retry.OnError(retry.DefaultRetry, func(err error) bool {
		return apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err)
	}, func() error {
		existing, err := client.CoreV1().PersistentVolumeClaims(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			_, err = client.CoreV1().PersistentVolumeClaims(desired.Namespace).Create(ctx, desired.DeepCopy(), metav1.CreateOptions{})
			return err
		}
		if err != nil {
			return err
		}
		if err := validatePersistentVolumeClaimSpec(existing, desired); err != nil {
			return err
		}

		desiredCapacity := desired.Spec.Resources.Requests[corev1.ResourceStorage]
		currentCapacity := existing.Spec.Resources.Requests[corev1.ResourceStorage]
		if desiredCapacity.Cmp(currentCapacity) <= 0 {
			return nil
		}

		updated := existing.DeepCopy()
		if updated.Spec.Resources.Requests == nil {
			updated.Spec.Resources.Requests = corev1.ResourceList{}
		}
		updated.Spec.Resources.Requests[corev1.ResourceStorage] = desiredCapacity
		_, err = client.CoreV1().PersistentVolumeClaims(updated.Namespace).Update(ctx, updated, metav1.UpdateOptions{})
		return err
	})
}

func appOwnedSelector(appID string) string {
	return labels.Set(map[string]string{
		kube.LabelManagedBy: "true",
		kube.LabelAppID:     appID,
	}).AsSelector().String()
}

func deleteDeploymentIfExists(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	err := client.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func deleteStatefulSetIfExists(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	err := client.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func deleteHPAIfExists(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	err := client.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func deleteConfigMapIfExists(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	err := client.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func deleteSecretIfExists(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	err := client.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// DeleteAppResources removes all non-persistent resources owned by an app.
// PVCs are retained when keepStorageData is true so recycle-bin restore does
// not destroy user data implicitly.
func DeleteAppResources(ctx context.Context, appCtx *models.AppContext, keepStorageData bool) error {
	if appCtx == nil || appCtx.EnvContext.Env.ClusterID == "" {
		return nil
	}
	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}
	namespace := appCtx.EnvContext.Env.ClusterNamespace
	deleteCollection := func(delete func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error) error {
		err := delete(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: appOwnedSelector(appCtx.App.ID)})
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if err := deleteCollection(func(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
		return client.AppsV1().Deployments(namespace).DeleteCollection(ctx, options, listOptions)
	}); err != nil {
		return err
	}
	if err := deleteCollection(func(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
		return client.AppsV1().StatefulSets(namespace).DeleteCollection(ctx, options, listOptions)
	}); err != nil {
		return err
	}
	if err := deleteCollection(func(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
		return client.AutoscalingV2().HorizontalPodAutoscalers(namespace).DeleteCollection(ctx, options, listOptions)
	}); err != nil {
		return err
	}
	if err := deleteCollection(func(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
		return client.CoreV1().ConfigMaps(namespace).DeleteCollection(ctx, options, listOptions)
	}); err != nil {
		return err
	}
	if err := deleteCollection(func(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
		return client.CoreV1().Secrets(namespace).DeleteCollection(ctx, options, listOptions)
	}); err != nil {
		return err
	}
	// Resources created before ownership labels were introduced cannot be found
	// by the collection selectors above. Remove those legacy resources by their
	// stable app-derived names as a compatibility fallback.
	if err := deleteLegacyAppResources(ctx, client, appCtx, keepStorageData); err != nil {
		return err
	}
	if err := deleteOwnedServices(ctx, client, namespace, appCtx.App.ID, appCtx.App.Slug); err != nil {
		return err
	}
	if !keepStorageData {
		if err := deleteCollection(func(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
			return client.CoreV1().PersistentVolumeClaims(namespace).DeleteCollection(ctx, options, listOptions)
		}); err != nil {
			return err
		}
	}

	if gwClient, err := kube.GlobalClusterStore.GetGatewayClient(appCtx.EnvContext.Env.ClusterID); err == nil {
		if err := gwClient.GatewayV1().HTTPRoutes(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: appOwnedSelector(appCtx.App.ID)}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		// Legacy HTTPRoutes did not carry app ownership labels. Their names are
		// deterministic from the app slug and database route ID.
		for _, route := range appCtx.GatewayRoutes {
			name := buildGatewayHTTPRouteName(appCtx.App.Slug, route.ID)
			if err := gwClient.GatewayV1().HTTPRoutes(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
		}
	} else {
		return err
	}
	return nil
}

func deleteLegacyAppResources(ctx context.Context, client kubernetes.Interface, appCtx *models.AppContext, keepStorageData bool) error {
	namespace := appCtx.EnvContext.Env.ClusterNamespace
	slug := appCtx.App.Slug

	if err := deleteDeploymentIfExists(ctx, client, namespace, slug); err != nil {
		return err
	}
	if err := deleteStatefulSetIfExists(ctx, client, namespace, slug); err != nil {
		return err
	}
	if err := deleteHPAIfExists(ctx, client, namespace, slug); err != nil {
		return err
	}
	if err := deleteConfigMapIfExists(ctx, client, namespace, slug+"-config"); err != nil {
		return err
	}
	for _, name := range []string{
		slug + "-config-secret",
		slug + "-env-secret",
		slug + "-registry",
	} {
		if err := deleteSecretIfExists(ctx, client, namespace, name); err != nil {
			return err
		}
	}

	if !keepStorageData {
		persistentVolumeClaims, err := client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, volume := range appCtx.Volumes {
			if volume.VolumeType != app.VolumeTypePVC || strings.TrimSpace(volume.Slug) == "" {
				continue
			}
			if err := deletePersistentVolumeClaimIfExists(ctx, client, namespace, volume.Slug); err != nil {
				return err
			}
			statefulSetClaimPrefix := volume.Slug + "-" + slug + "-"
			for i := range persistentVolumeClaims.Items {
				claim := &persistentVolumeClaims.Items[i]
				if !strings.HasPrefix(claim.Name, statefulSetClaimPrefix) {
					continue
				}
				if err := deletePersistentVolumeClaimIfExists(ctx, client, namespace, claim.Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func deletePersistentVolumeClaimIfExists(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	err := client.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func deleteOwnedServices(ctx context.Context, client kubernetes.Interface, namespace, appID string, appSlugs ...string) error {
	services, err := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{LabelSelector: appOwnedSelector(appID)})
	if err != nil {
		return err
	}
	for i := range services.Items {
		service := &services.Items[i]
		if err := client.CoreV1().Services(namespace).Delete(ctx, service.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	// Legacy services were named after the app but had no ownership labels.
	appSlug := ""
	if len(appSlugs) > 0 {
		appSlug = appSlugs[0]
	}
	if strings.TrimSpace(appSlug) == "" {
		return nil
	}
	services, err = client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	nodePortPrefix := appSlug + "-np-"
	for i := range services.Items {
		service := &services.Items[i]
		if service.Name != appSlug && !strings.HasPrefix(service.Name, nodePortPrefix) {
			continue
		}
		if err := client.CoreV1().Services(namespace).Delete(ctx, service.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
