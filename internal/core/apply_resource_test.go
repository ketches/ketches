package core

import (
	"context"
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestApplyResourcePreservesResourceVersions(t *testing.T) {
	namespace := "app-ns"
	client := fake.NewSimpleClientset(
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: namespace, ResourceVersion: "11"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: namespace, ResourceVersion: "12"}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: namespace, ResourceVersion: "13"}},
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "statefulset", Namespace: namespace, ResourceVersion: "14"}},
		&autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: "hpa", Namespace: namespace, ResourceVersion: "15"}},
	)

	resources := []struct {
		name string
		res  runtime.Object
	}{
		{name: "config map", res: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: namespace}}},
		{name: "secret", res: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: namespace}}},
		{name: "deployment", res: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: namespace}}},
		{name: "statefulset", res: &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "statefulset", Namespace: namespace}}},
		{name: "hpa", res: &autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: "hpa", Namespace: namespace}}},
	}
	for _, item := range resources {
		t.Run(item.name, func(t *testing.T) {
			if err := ApplyResource(context.Background(), client, item.res.DeepCopyObject()); err != nil {
				t.Fatalf("ApplyResource(%s): %v", item.name, err)
			}
		})
	}

	if updated, _ := client.CoreV1().ConfigMaps(namespace).Get(context.Background(), "config", metav1.GetOptions{}); updated.ResourceVersion != "11" {
		t.Fatalf("ConfigMap resourceVersion = %q", updated.ResourceVersion)
	}
	if updated, _ := client.CoreV1().Secrets(namespace).Get(context.Background(), "secret", metav1.GetOptions{}); updated.ResourceVersion != "12" {
		t.Fatalf("Secret resourceVersion = %q", updated.ResourceVersion)
	}
	if updated, _ := client.AppsV1().Deployments(namespace).Get(context.Background(), "deployment", metav1.GetOptions{}); updated.ResourceVersion != "13" {
		t.Fatalf("Deployment resourceVersion = %q", updated.ResourceVersion)
	}
	if updated, _ := client.AppsV1().StatefulSets(namespace).Get(context.Background(), "statefulset", metav1.GetOptions{}); updated.ResourceVersion != "14" {
		t.Fatalf("StatefulSet resourceVersion = %q", updated.ResourceVersion)
	}
	if updated, _ := client.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(context.Background(), "hpa", metav1.GetOptions{}); updated.ResourceVersion != "15" {
		t.Fatalf("HPA resourceVersion = %q", updated.ResourceVersion)
	}
}

func TestApplyServicePreservesAllocatedFields(t *testing.T) {
	namespace := "app-ns"
	policy := corev1.IPFamilyPolicySingleStack
	client := fake.NewSimpleClientset(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: namespace, ResourceVersion: "21"},
		Spec: corev1.ServiceSpec{
			ClusterIP:      "10.96.0.8",
			ClusterIPs:     []string{"10.96.0.8"},
			IPFamilies:     []corev1.IPFamily{corev1.IPv4Protocol},
			IPFamilyPolicy: &policy,
			Ports:          []corev1.ServicePort{{Name: "http", Protocol: corev1.ProtocolTCP, Port: 80, NodePort: 32080}},
		},
	})
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: namespace},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeNodePort, Ports: []corev1.ServicePort{{Name: "http", Protocol: corev1.ProtocolTCP, Port: 80}}},
	}
	if err := ApplyResource(context.Background(), client, desired); err != nil {
		t.Fatalf("ApplyResource(Service): %v", err)
	}
	updated, err := client.CoreV1().Services(namespace).Get(context.Background(), "app", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ResourceVersion != "21" || updated.Spec.ClusterIP != "10.96.0.8" || updated.Spec.Ports[0].NodePort != 32080 {
		t.Fatalf("allocated fields were not preserved: %#v", updated.Spec)
	}
}

func TestDeleteLegacyAppResourcesByStableNames(t *testing.T) {
	const namespace = "app-ns"
	const slug = "demo-app"
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: slug, Namespace: namespace}},
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: slug, Namespace: namespace}},
		&autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: slug, Namespace: namespace}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: slug + "-config", Namespace: namespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: slug + "-config-secret", Namespace: namespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: slug + "-env-secret", Namespace: namespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: slug + "-registry", Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: slug, Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: slug + "-np-8080", Namespace: namespace}},
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: namespace}},
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data-" + slug + "-0", Namespace: namespace}},
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data-" + slug + "-1", Namespace: namespace}},
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data-other-app-0", Namespace: namespace}},
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: namespace}},
	)
	appCtx := &models.AppContext{
		App:        entities.App{Base: entities.Base{ID: "app-1"}, Slug: slug},
		EnvContext: models.EnvContext{Env: entities.Env{ClusterNamespace: namespace}},
		Volumes:    []entities.AppVolume{{Slug: "data", VolumeType: app.VolumeTypePVC}},
	}

	if err := deleteLegacyAppResources(context.Background(), client, appCtx, false); err != nil {
		t.Fatalf("deleteLegacyAppResources: %v", err)
	}
	if err := deleteOwnedServices(context.Background(), client, namespace, appCtx.App.ID, slug); err != nil {
		t.Fatalf("deleteOwnedServices: %v", err)
	}

	for _, check := range []struct {
		name string
		get  func() error
	}{
		{"deployment", func() error {
			_, err := client.AppsV1().Deployments(namespace).Get(context.Background(), slug, metav1.GetOptions{})
			return err
		}},
		{"statefulset", func() error {
			_, err := client.AppsV1().StatefulSets(namespace).Get(context.Background(), slug, metav1.GetOptions{})
			return err
		}},
		{"hpa", func() error {
			_, err := client.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(context.Background(), slug, metav1.GetOptions{})
			return err
		}},
		{"config map", func() error {
			_, err := client.CoreV1().ConfigMaps(namespace).Get(context.Background(), slug+"-config", metav1.GetOptions{})
			return err
		}},
		{"config secret", func() error {
			_, err := client.CoreV1().Secrets(namespace).Get(context.Background(), slug+"-config-secret", metav1.GetOptions{})
			return err
		}},
		{"env secret", func() error {
			_, err := client.CoreV1().Secrets(namespace).Get(context.Background(), slug+"-env-secret", metav1.GetOptions{})
			return err
		}},
		{"registry secret", func() error {
			_, err := client.CoreV1().Secrets(namespace).Get(context.Background(), slug+"-registry", metav1.GetOptions{})
			return err
		}},
		{"cluster ip service", func() error {
			_, err := client.CoreV1().Services(namespace).Get(context.Background(), slug, metav1.GetOptions{})
			return err
		}},
		{"node port service", func() error {
			_, err := client.CoreV1().Services(namespace).Get(context.Background(), slug+"-np-8080", metav1.GetOptions{})
			return err
		}},
		{"pvc", func() error {
			_, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), "data", metav1.GetOptions{})
			return err
		}},
		{"statefulset pvc ordinal 0", func() error {
			_, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), "data-"+slug+"-0", metav1.GetOptions{})
			return err
		}},
		{"statefulset pvc ordinal 1", func() error {
			_, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), "data-"+slug+"-1", metav1.GetOptions{})
			return err
		}},
	} {
		t.Run(check.name, func(t *testing.T) {
			if err := check.get(); !apierrors.IsNotFound(err) {
				t.Fatalf("resource still exists, get error = %v", err)
			}
		})
	}
	if _, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), "unrelated", metav1.GetOptions{}); err != nil {
		t.Fatalf("unrelated PVC should remain: %v", err)
	}
	if _, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), "data-other-app-0", metav1.GetOptions{}); err != nil {
		t.Fatalf("another app's StatefulSet PVC should remain: %v", err)
	}
}

func TestDeleteLegacyAppResourcesKeepsPVCWhenRequested(t *testing.T) {
	const namespace = "app-ns"
	client := fake.NewSimpleClientset(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: namespace}})
	appCtx := &models.AppContext{
		App:        entities.App{Slug: "demo-app"},
		EnvContext: models.EnvContext{Env: entities.Env{ClusterNamespace: namespace}},
		Volumes:    []entities.AppVolume{{Slug: "data", VolumeType: app.VolumeTypePVC}},
	}
	if err := deleteLegacyAppResources(context.Background(), client, appCtx, true); err != nil {
		t.Fatalf("deleteLegacyAppResources: %v", err)
	}
	if _, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), "data", metav1.GetOptions{}); err != nil {
		t.Fatalf("PVC should be retained: %v", err)
	}
}

func TestUpdateAppConfigRevisionUpdatesDeploymentAndStatefulSet(t *testing.T) {
	for _, tc := range []struct {
		name      string
		appType   string
		resource  string
		getObject func(*testing.T, *fake.Clientset) metav1.Object
	}{
		{
			name:     "deployment",
			appType:  app.AppTypeDeployment,
			resource: "deployments",
			getObject: func(t *testing.T, client *fake.Clientset) metav1.Object {
				deployment, err := client.AppsV1().Deployments("app-ns").Get(context.Background(), "demo-app", metav1.GetOptions{})
				require.NoError(t, err)
				return &deployment.Spec.Template.ObjectMeta
			},
		},
		{
			name:     "statefulset",
			appType:  app.AppTypeStatefulSet,
			resource: "statefulsets",
			getObject: func(t *testing.T, client *fake.Clientset) metav1.Object {
				statefulSet, err := client.AppsV1().StatefulSets("app-ns").Get(context.Background(), "demo-app", metav1.GetOptions{})
				require.NoError(t, err)
				return &statefulSet.Spec.Template.ObjectMeta
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			namespace := "app-ns"
			var client *fake.Clientset
			if tc.appType == app.AppTypeStatefulSet {
				client = fake.NewSimpleClientset(&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{Name: "demo-app", Namespace: namespace},
					Spec:       appsv1.StatefulSetSpec{Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"existing": "value"}}}},
				})
			} else {
				client = fake.NewSimpleClientset(&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "demo-app", Namespace: namespace},
					Spec:       appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"existing": "value"}}}},
				})
			}

			updateCount := 0
			client.PrependReactor("update", tc.resource, func(action k8stesting.Action) (bool, runtime.Object, error) {
				updateCount++
				return false, nil, nil
			})

			require.NoError(t, updateAppConfigRevision(context.Background(), client, tc.appType, namespace, "demo-app", "revision-1"))
			metadata := tc.getObject(t, client)
			assert.Equal(t, "value", metadata.GetAnnotations()["existing"])
			assert.Equal(t, "revision-1", metadata.GetAnnotations()[configRevisionAnnotation])
			assert.Equal(t, 1, updateCount)

			require.NoError(t, updateAppConfigRevision(context.Background(), client, tc.appType, namespace, "demo-app", "revision-1"))
			assert.Equal(t, 1, updateCount, "unchanged revisions should not update the workload")
		})
	}
}

func TestUpdateAppConfigRevisionIgnoresMissingWorkload(t *testing.T) {
	client := fake.NewSimpleClientset()
	require.NoError(t, updateAppConfigRevision(context.Background(), client, app.AppTypeDeployment, "app-ns", "missing", "revision-1"))
	require.NoError(t, updateAppConfigRevision(context.Background(), client, app.AppTypeStatefulSet, "app-ns", "missing", "revision-1"))
}

func TestUpdateAppConfigRevisionIgnoresWorkloadDeletedBeforeUpdate(t *testing.T) {
	for _, tc := range []struct {
		name     string
		appType  string
		resource string
		object   runtime.Object
	}{
		{
			name:     "deployment",
			appType:  app.AppTypeDeployment,
			resource: "deployments",
			object:   &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "demo-app", Namespace: "app-ns"}},
		},
		{
			name:     "statefulset",
			appType:  app.AppTypeStatefulSet,
			resource: "statefulsets",
			object:   &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "demo-app", Namespace: "app-ns"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewSimpleClientset(tc.object)
			client.PrependReactor("update", tc.resource, func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "apps", Resource: tc.resource}, "demo-app")
			})

			require.NoError(t, updateAppConfigRevision(
				context.Background(), client, tc.appType, "app-ns", "demo-app", "revision-1",
			))
		})
	}
}
