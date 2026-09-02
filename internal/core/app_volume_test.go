package core

import (
	"context"
	"testing"
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func testPVC(name, namespace, capacity, storageClass string) *corev1.PersistentVolumeClaim {
	volumeMode := corev1.PersistentVolumeFilesystem
	var storageClassName *string
	if storageClass != "" {
		storageClassName = &storageClass
	}
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: storageClassName,
			VolumeMode:       &volumeMode,
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse(capacity),
			}},
		},
	}
}

func testStatefulSetWithPVC(name, namespace, capacity, storageClass string) *appsv1.StatefulSet {
	claim := testPVC("data", namespace, capacity, storageClass)
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: appsv1.StatefulSetSpec{
			Template:             corev1.PodTemplateSpec{},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{*claim},
		},
	}
}

func TestValidatePersistentVolumeClaimSpecRejectsImmutableChangesAndShrink(t *testing.T) {
	const namespace = "app-ns"
	existing := testPVC("data", namespace, "10Gi", "fast")

	tests := []struct {
		name   string
		mutate func(*corev1.PersistentVolumeClaim)
		match  string
	}{
		{
			name: "storage class",
			mutate: func(claim *corev1.PersistentVolumeClaim) {
				storageClass := "slow"
				claim.Spec.StorageClassName = &storageClass
			},
			match: "storage class",
		},
		{
			name: "volume mode",
			mutate: func(claim *corev1.PersistentVolumeClaim) {
				mode := corev1.PersistentVolumeBlock
				claim.Spec.VolumeMode = &mode
			},
			match: "volume mode",
		},
		{
			name: "access modes",
			mutate: func(claim *corev1.PersistentVolumeClaim) {
				claim.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
			},
			match: "access modes",
		},
		{
			name: "capacity shrink",
			mutate: func(claim *corev1.PersistentVolumeClaim) {
				claim.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("9Gi")
			},
			match: "cannot be reduced",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			desired := existing.DeepCopy()
			test.mutate(desired)
			err := validatePersistentVolumeClaimSpec(existing, desired)
			require.ErrorContains(t, err, test.match)
		})
	}
}

func TestValidatePersistentVolumeClaimSpecPreservesDefaultedStorageClassWhenOmitted(t *testing.T) {
	existing := testPVC("data", "app-ns", "10Gi", "default-storage")
	desired := testPVC("data", "app-ns", "10Gi", "")

	require.NoError(t, validatePersistentVolumeClaimSpec(existing, desired))

	desired.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("20Gi")
	require.NoError(t, validatePersistentVolumeClaimSpec(existing, desired))

	desired.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("9Gi")
	require.ErrorContains(t, validatePersistentVolumeClaimSpec(existing, desired), "cannot be reduced")

	changedStorageClass := "other-storage"
	desired.Spec.StorageClassName = &changedStorageClass
	desired.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("10Gi")
	require.ErrorContains(t, validatePersistentVolumeClaimSpec(existing, desired), "storage class")
}

func TestApplyPersistentVolumeClaimExpandsCapacity(t *testing.T) {
	const namespace = "app-ns"
	client := fake.NewSimpleClientset(testPVC("data", namespace, "10Gi", "fast"))
	desired := testPVC("data", namespace, "20Gi", "fast")

	require.NoError(t, applyPersistentVolumeClaim(context.Background(), client, desired))
	updated, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), "data", metav1.GetOptions{})
	require.NoError(t, err)
	updatedCapacity := updated.Spec.Resources.Requests[corev1.ResourceStorage]
	assert.Zero(t, updatedCapacity.Cmp(resource.MustParse("20Gi")))
}

func TestApplyStatefulSetRejectsImmutableVolumeClaimTemplateChanges(t *testing.T) {
	const (
		name      = "database"
		namespace = "app-ns"
	)

	tests := []struct {
		name   string
		mutate func(*appsv1.StatefulSet)
		match  string
	}{
		{
			name: "storage class",
			mutate: func(statefulSet *appsv1.StatefulSet) {
				storageClass := "slow"
				statefulSet.Spec.VolumeClaimTemplates[0].Spec.StorageClassName = &storageClass
			},
			match: "storage class",
		},
		{
			name: "volume mode",
			mutate: func(statefulSet *appsv1.StatefulSet) {
				mode := corev1.PersistentVolumeBlock
				statefulSet.Spec.VolumeClaimTemplates[0].Spec.VolumeMode = &mode
			},
			match: "volume mode",
		},
		{
			name: "access modes",
			mutate: func(statefulSet *appsv1.StatefulSet) {
				statefulSet.Spec.VolumeClaimTemplates[0].Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany}
			},
			match: "access modes",
		},
		{
			name: "capacity expansion",
			mutate: func(statefulSet *appsv1.StatefulSet) {
				statefulSet.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("20Gi")
			},
			match: "capacity is immutable",
		},
		{
			name: "template addition",
			mutate: func(statefulSet *appsv1.StatefulSet) {
				statefulSet.Spec.VolumeClaimTemplates = append(
					statefulSet.Spec.VolumeClaimTemplates,
					*testPVC("cache", namespace, "1Gi", "fast"),
				)
			},
			match: "add or remove",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			existing := testStatefulSetWithPVC(name, namespace, "10Gi", "fast")
			client := fake.NewSimpleClientset(existing)
			desired := existing.DeepCopy()
			test.mutate(desired)

			err := applyStatefulSet(context.Background(), client, desired)
			require.ErrorContains(t, err, test.match)
			for _, action := range client.Actions() {
				assert.False(t, action.Matches("update", "statefulsets"), "an invalid template must not reach Kubernetes Update")
			}
		})
	}
}

func TestApplyStatefulSetPreservesDefaultedVolumeClaimTemplate(t *testing.T) {
	const (
		name      = "database"
		namespace = "app-ns"
	)
	existing := testStatefulSetWithPVC(name, namespace, "10Gi", "default-storage")
	desired := testStatefulSetWithPVC(name, namespace, "10Gi", "")
	replicas := int32(2)
	desired.Spec.Replicas = &replicas
	client := fake.NewSimpleClientset(existing)

	require.NoError(t, applyStatefulSet(context.Background(), client, desired))
	updated, err := client.AppsV1().StatefulSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, updated.Spec.VolumeClaimTemplates[0].Spec.StorageClassName)
	assert.Equal(t, "default-storage", *updated.Spec.VolumeClaimTemplates[0].Spec.StorageClassName)
	require.NotNil(t, updated.Spec.Replicas)
	assert.Equal(t, int32(2), *updated.Spec.Replicas)
}

func TestSyncVolumeToK8sSkipsStandalonePVCForStatefulSet(t *testing.T) {
	appCtx := &models.AppContext{
		App: entities.App{AppType: app.AppTypeStatefulSet},
		EnvContext: models.EnvContext{Env: entities.Env{
			ClusterID:        "cluster-not-registered",
			ClusterNamespace: "app-ns",
		}},
	}
	volume := &entities.AppVolume{Slug: "data", VolumeType: app.VolumeTypePVC, Capacity: 10}

	// A client lookup would fail because the cluster is intentionally absent.
	// Returning nil proves the StatefulSet path does not create a standalone PVC.
	require.NoError(t, SyncVolumeToK8s(context.Background(), appCtx, volume))
}

func TestSyncVolumeToK8sUsesAppReconcileFence(t *testing.T) {
	originalDB := db.DB
	db.DB = nil
	t.Cleanup(func() { db.DB = originalDB })

	volume := entities.AppVolume{ID: "volume-fenced", Slug: "data", VolumeType: app.VolumeTypePVC, Capacity: 10}
	appCtx := &models.AppContext{
		App: entities.App{
			Base:    entities.Base{ID: "app-volume-fenced"},
			AppType: app.AppTypeStatefulSet,
		},
		EnvContext: models.EnvContext{Env: entities.Env{
			ClusterID:        "cluster-not-registered",
			ClusterNamespace: "app-ns",
		}},
		Volumes: []entities.AppVolume{volume},
	}

	fenceHeld := make(chan struct{})
	releaseFence := make(chan struct{})
	fenceResult := make(chan error, 1)
	go func() {
		fenceResult <- WithAppReconcileFence(context.Background(), appCtx.App.ID, func() error {
			close(fenceHeld)
			<-releaseFence
			return nil
		})
	}()
	select {
	case <-fenceHeld:
	case <-time.After(5 * time.Second):
		t.Fatal("test fence was not acquired")
	}

	syncResult := make(chan error, 1)
	go func() {
		syncResult <- SyncVolumeToK8s(context.Background(), appCtx, &volume)
	}()
	select {
	case err := <-syncResult:
		t.Fatalf("volume sync completed while the app fence was held: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseFence)
	require.NoError(t, <-fenceResult)
	require.NoError(t, <-syncResult)
}
