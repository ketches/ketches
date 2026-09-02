package core

import (
	"context"
	"sort"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SyncVolumeToK8s synchronizes a volume to Kubernetes (creates or safely
// updates a PVC for persistent volumes).
func SyncVolumeToK8s(ctx context.Context, appCtx *models.AppContext, volume *entities.AppVolume) error {
	if appCtx == nil || volume == nil {
		return app.NewErrorf("volume sync requires an application context and volume")
	}
	return withAppReconcileLockContext(ctx, appCtx, func(latest *models.AppContext) error {
		desired := volume
		if latest.App.ID != "" {
			desired = nil
			for i := range latest.Volumes {
				if latest.Volumes[i].ID == volume.ID {
					desired = &latest.Volumes[i]
					break
				}
			}
			if desired == nil {
				return app.NewErrorf("volume %s no longer exists in application %s", volume.ID, latest.App.ID)
			}
		}
		return syncVolumeToK8s(ctx, latest, desired)
	})
}

func syncVolumeToK8s(ctx context.Context, appCtx *models.AppContext, volume *entities.AppVolume) error {
	if appCtx.EnvContext.Env.ClusterID == "" {
		return app.NewErrorf("app environment has no cluster configured")
	}

	// Only create PVC for pvc type volumes.
	if volume.VolumeType != app.VolumeTypePVC {
		return nil
	}
	if appCtx.App.AppType == app.AppTypeStatefulSet {
		// StatefulSet controllers create claims from immutable claim templates.
		// Creating a same-named standalone PVC would not back the StatefulSet.
		return nil
	}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	metadata := &AppMetadata{AppContext: appCtx}
	return applyPersistentVolumeClaim(ctx, client, metadata.BuildPVC(*volume))
}

// ValidateAppVolumeUpdate checks PVC identity and immutable fields before the
// database row is changed. This prevents an API response from claiming a
// Kubernetes PVC specification that cannot be applied.
func ValidateAppVolumeUpdate(ctx context.Context, appCtx *models.AppContext, current, desired *entities.AppVolume) error {
	if current == nil || desired == nil {
		return app.NewErrorf("volume update is missing its current or desired value")
	}
	if current.VolumeType != app.VolumeTypePVC && desired.VolumeType != app.VolumeTypePVC {
		return nil
	}
	if appCtx.EnvContext.Env.ClusterID == "" {
		return app.NewErrorf("app environment has no cluster configured")
	}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}
	if appCtx.App.AppType == app.AppTypeStatefulSet {
		return validateStatefulSetAppVolumeUpdate(ctx, client, appCtx, current, desired)
	}
	pvcName := desired.Slug
	if current.VolumeType == app.VolumeTypePVC {
		pvcName = current.Slug
	}
	existing, err := client.CoreV1().PersistentVolumeClaims(appCtx.EnvContext.Env.ClusterNamespace).Get(ctx, pvcName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if current.VolumeType != app.VolumeTypePVC {
		return app.NewErrorf("cannot reuse existing PVC %q for a non-PVC volume", existing.Name)
	}
	if desired.VolumeType != app.VolumeTypePVC {
		return app.NewErrorf("cannot change PVC volume %q to type %q while its PVC exists; delete the volume first", current.Slug, desired.VolumeType)
	}
	if current.Slug != desired.Slug {
		return app.NewErrorf("cannot rename PVC-backed volume %q while its PVC exists", current.Slug)
	}

	metadata := &AppMetadata{AppContext: appCtx}
	return validatePersistentVolumeClaimSpec(existing, metadata.BuildPVC(*desired))
}

// ValidateAppVolumeCreate rejects PVC-template additions that Kubernetes
// cannot apply to an existing StatefulSet.
func ValidateAppVolumeCreate(ctx context.Context, appCtx *models.AppContext, desired *entities.AppVolume) error {
	if desired == nil || desired.VolumeType != app.VolumeTypePVC || appCtx.App.AppType != app.AppTypeStatefulSet {
		return nil
	}
	if appCtx.EnvContext.Env.ClusterID == "" {
		return app.NewErrorf("app environment has no cluster configured")
	}
	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}
	existing, err := client.AppsV1().StatefulSets(appCtx.EnvContext.Env.ClusterNamespace).Get(
		ctx,
		appCtx.App.Slug,
		metav1.GetOptions{},
	)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return app.NewErrorf(
		"cannot add PVC-backed volume %q while StatefulSet %q exists; recreate the StatefulSet to change volume claim templates",
		desired.Slug,
		existing.Name,
	)
}

// ValidateAppVolumeDelete rejects removal of a claim template from an existing
// StatefulSet. Deleting only the database row would make the desired and live
// specifications diverge permanently.
func ValidateAppVolumeDelete(ctx context.Context, appCtx *models.AppContext, current *entities.AppVolume) error {
	if current == nil || current.VolumeType != app.VolumeTypePVC || appCtx.App.AppType != app.AppTypeStatefulSet {
		return nil
	}
	if appCtx.EnvContext.Env.ClusterID == "" {
		return app.NewErrorf("app environment has no cluster configured")
	}
	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}
	existing, err := client.AppsV1().StatefulSets(appCtx.EnvContext.Env.ClusterNamespace).Get(
		ctx,
		appCtx.App.Slug,
		metav1.GetOptions{},
	)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for i := range existing.Spec.VolumeClaimTemplates {
		if existing.Spec.VolumeClaimTemplates[i].Name == current.Slug {
			return app.NewErrorf(
				"cannot delete PVC-backed volume %q while StatefulSet %q uses its immutable volume claim template; recreate the StatefulSet first",
				current.Slug,
				existing.Name,
			)
		}
	}
	return nil
}

func validateStatefulSetAppVolumeUpdate(
	ctx context.Context,
	client kubernetes.Interface,
	appCtx *models.AppContext,
	current, desired *entities.AppVolume,
) error {
	existing, err := client.AppsV1().StatefulSets(appCtx.EnvContext.Env.ClusterNamespace).Get(
		ctx,
		appCtx.App.Slug,
		metav1.GetOptions{},
	)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	desiredAppCtx := *appCtx
	desiredAppCtx.Volumes = append([]entities.AppVolume(nil), appCtx.Volumes...)
	updated := false
	for i := range desiredAppCtx.Volumes {
		if desiredAppCtx.Volumes[i].ID != current.ID {
			continue
		}
		desiredAppCtx.Volumes[i] = *desired
		updated = true
		break
	}
	if !updated {
		return app.NewErrorf("volume %q is missing from its application context", current.Slug)
	}

	desiredStatefulSet := (&AppMetadata{AppContext: &desiredAppCtx}).BuildStatefulSet()
	return validateStatefulSetVolumeClaimTemplates(existing, desiredStatefulSet)
}

func validateStatefulSetVolumeClaimTemplates(existing, desired *appsv1.StatefulSet) error {
	existingByName := make(map[string]*corev1.PersistentVolumeClaim, len(existing.Spec.VolumeClaimTemplates))
	for i := range existing.Spec.VolumeClaimTemplates {
		claim := &existing.Spec.VolumeClaimTemplates[i]
		if _, duplicate := existingByName[claim.Name]; duplicate {
			return app.NewErrorf("StatefulSet %q has duplicate volume claim template %q", existing.Name, claim.Name)
		}
		existingByName[claim.Name] = claim
	}
	if len(existingByName) != len(desired.Spec.VolumeClaimTemplates) {
		return app.NewErrorf(
			"StatefulSet %q volume claim templates are immutable; recreate the StatefulSet to add or remove persistent volumes",
			existing.Name,
		)
	}

	for i := range desired.Spec.VolumeClaimTemplates {
		desiredClaim := &desired.Spec.VolumeClaimTemplates[i]
		existingClaim, ok := existingByName[desiredClaim.Name]
		if !ok {
			return app.NewErrorf(
				"StatefulSet %q volume claim templates are immutable; cannot add or rename template %q",
				existing.Name,
				desiredClaim.Name,
			)
		}
		if err := validateStatefulSetVolumeClaimTemplateSpec(existing.Name, existingClaim, desiredClaim); err != nil {
			return err
		}
	}
	return nil
}

func validateStatefulSetVolumeClaimTemplateSpec(
	statefulSetName string,
	existing, desired *corev1.PersistentVolumeClaim,
) error {
	if desired.Spec.StorageClassName != nil && normalizePVCStorageClass(existing.Spec.StorageClassName) != normalizePVCStorageClass(desired.Spec.StorageClassName) {
		return app.NewErrorf("StatefulSet %q PVC template %q storage class is immutable", statefulSetName, desired.Name)
	}
	if normalizePVCVolumeMode(existing.Spec.VolumeMode) != normalizePVCVolumeMode(desired.Spec.VolumeMode) {
		return app.NewErrorf("StatefulSet %q PVC template %q volume mode is immutable", statefulSetName, desired.Name)
	}
	if !samePVCAccessModes(existing.Spec.AccessModes, desired.Spec.AccessModes) {
		return app.NewErrorf("StatefulSet %q PVC template %q access modes are immutable", statefulSetName, desired.Name)
	}

	existingCapacity := existing.Spec.Resources.Requests[corev1.ResourceStorage]
	desiredCapacity := desired.Spec.Resources.Requests[corev1.ResourceStorage]
	if desiredCapacity.Cmp(existingCapacity) != 0 {
		return app.NewErrorf(
			"StatefulSet %q PVC template %q capacity is immutable; recreate the StatefulSet to change it",
			statefulSetName,
			desired.Name,
		)
	}

	existingSpec := existing.Spec.DeepCopy()
	desiredSpec := desired.Spec.DeepCopy()
	normalizeComparedPVCTemplateSpec(existingSpec)
	normalizeComparedPVCTemplateSpec(desiredSpec)
	if !apiequality.Semantic.DeepEqual(existingSpec, desiredSpec) {
		return app.NewErrorf("StatefulSet %q PVC template %q contains an immutable specification change", statefulSetName, desired.Name)
	}
	return nil
}

func normalizeComparedPVCTemplateSpec(spec *corev1.PersistentVolumeClaimSpec) {
	spec.StorageClassName = nil
	spec.VolumeMode = nil
	spec.AccessModes = nil
	if spec.Resources.Requests != nil {
		delete(spec.Resources.Requests, corev1.ResourceStorage)
		if len(spec.Resources.Requests) == 0 {
			spec.Resources.Requests = nil
		}
	}
}

func copyPersistentVolumeClaimTemplates(items []corev1.PersistentVolumeClaim) []corev1.PersistentVolumeClaim {
	if items == nil {
		return nil
	}
	result := make([]corev1.PersistentVolumeClaim, len(items))
	for i := range items {
		items[i].DeepCopyInto(&result[i])
	}
	return result
}

// DeleteVolumeFromK8s deletes a PVC from Kubernetes.
func DeleteVolumeFromK8s(ctx context.Context, appCtx *models.AppContext, volume *entities.AppVolume) error {
	if appCtx.EnvContext.Env.ClusterID == "" {
		return app.NewErrorf("app environment has no cluster configured")
	}

	// Only delete PVC for pvc type volumes.
	if volume.VolumeType != app.VolumeTypePVC {
		return nil
	}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	// Delete PVC.
	err = client.CoreV1().PersistentVolumeClaims(appCtx.EnvContext.Env.ClusterNamespace).Delete(
		ctx,
		volume.Slug,
		metav1.DeleteOptions{},
	)

	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

func validatePersistentVolumeClaimSpec(existing, desired *corev1.PersistentVolumeClaim) error {
	// A nil desired StorageClassName means "let Kubernetes choose the default"
	// on creation. Once a PVC exists, preserve its live/defaulted value rather
	// than treating that omitted field as an attempted immutable change.
	if desired.Spec.StorageClassName != nil &&
		normalizePVCStorageClass(existing.Spec.StorageClassName) != normalizePVCStorageClass(desired.Spec.StorageClassName) {
		return app.NewErrorf("PVC %q storage class cannot be changed after creation", existing.Name)
	}
	if normalizePVCVolumeMode(existing.Spec.VolumeMode) != normalizePVCVolumeMode(desired.Spec.VolumeMode) {
		return app.NewErrorf("PVC %q volume mode cannot be changed after creation", existing.Name)
	}
	if !samePVCAccessModes(existing.Spec.AccessModes, desired.Spec.AccessModes) {
		return app.NewErrorf("PVC %q access modes cannot be changed after creation", existing.Name)
	}

	currentCapacity := existing.Spec.Resources.Requests[corev1.ResourceStorage]
	desiredCapacity := desired.Spec.Resources.Requests[corev1.ResourceStorage]
	if desiredCapacity.Cmp(currentCapacity) < 0 {
		return app.NewErrorf("PVC %q capacity cannot be reduced", existing.Name)
	}
	return nil
}

func normalizePVCStorageClass(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func normalizePVCVolumeMode(value *corev1.PersistentVolumeMode) corev1.PersistentVolumeMode {
	if value == nil {
		return corev1.PersistentVolumeFilesystem
	}
	return *value
}

func samePVCAccessModes(left, right []corev1.PersistentVolumeAccessMode) bool {
	leftCopy := append([]corev1.PersistentVolumeAccessMode(nil), left...)
	rightCopy := append([]corev1.PersistentVolumeAccessMode(nil), right...)
	sort.Slice(leftCopy, func(i, j int) bool { return leftCopy[i] < leftCopy[j] })
	sort.Slice(rightCopy, func(i, j int) bool { return rightCopy[i] < rightCopy[j] })
	if len(leftCopy) != len(rightCopy) {
		return false
	}
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}
