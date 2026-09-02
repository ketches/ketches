package core

import (
	"context"
	"errors"
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestSyncConfigMapToK8sHashesDeployedEnvSecret(t *testing.T) {
	const (
		namespace = "app-ns"
		slug      = "demo-app"
	)
	appCtx := &models.AppContext{
		App: entities.App{
			Base:    entities.Base{ID: "app-1"},
			Slug:    slug,
			AppType: app.AppTypeDeployment,
		},
		EnvContext: models.EnvContext{Env: entities.Env{ClusterNamespace: namespace}},
		EnvVars: []entities.AppEnvVar{
			{Key: "SECRET_TOKEN", Value: "pending-secret", IsSecret: true},
		},
		ConfigFiles: []entities.AppConfigFile{
			{Slug: "app.conf", Content: "mode=production"},
			{Slug: "credentials.conf", Content: "password=secret", IsSecret: true},
		},
	}
	deployedEnvSecretData := map[string][]byte{"SECRET_TOKEN": []byte("deployed-secret")}
	client := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: slug, Namespace: namespace},
			Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"existing": "value"}},
			}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: slug + "-env-secret", Namespace: namespace},
			Data:       deployedEnvSecretData,
		},
	)

	revision, err := (&AppMetadata{AppContext: appCtx}).buildConfigRevisionWithEnvSecretData(deployedEnvSecretData)
	require.NoError(t, err)
	desiredRevision, err := (&AppMetadata{AppContext: appCtx}).BuildConfigRevision()
	require.NoError(t, err)
	require.NotEqual(t, desiredRevision, revision)

	require.NoError(t, syncConfigMapToK8s(context.Background(), client, appCtx))

	configMap, err := client.CoreV1().ConfigMaps(namespace).Get(context.Background(), slug+"-config", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"app.conf": "mode=production"}, configMap.Data)

	configSecret, err := client.CoreV1().Secrets(namespace).Get(context.Background(), slug+"-config-secret", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, map[string][]byte{"credentials.conf": []byte("password=secret")}, configSecret.Data)

	envSecret, err := client.CoreV1().Secrets(namespace).Get(context.Background(), slug+"-env-secret", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, deployedEnvSecretData, envSecret.Data)

	deployment, err := client.AppsV1().Deployments(namespace).Get(context.Background(), slug, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "value", deployment.Spec.Template.Annotations["existing"])
	assert.Equal(t, revision, deployment.Spec.Template.Annotations[configRevisionAnnotation])
}

func configSyncTestAppContext(appType string, files []entities.AppConfigFile) *models.AppContext {
	return &models.AppContext{
		App: entities.App{
			Base:           entities.Base{ID: "app-config-sync"},
			Slug:           "demo-app",
			AppType:        appType,
			ContainerImage: "busybox:1.36",
			Replicas:       1,
		},
		EnvContext: models.EnvContext{Env: entities.Env{
			ClusterID:        "cluster-config-sync",
			ClusterNamespace: "app-ns",
		}},
		ConfigFiles: files,
	}
}

func configSyncTestWorkload(t *testing.T, appCtx *models.AppContext) *appsv1.Deployment {
	t.Helper()
	oldContext := *appCtx
	oldContext.ConfigFiles = nil
	return (&AppMetadata{AppContext: &oldContext}).BuildDeployment()
}

func configSyncTestStatefulSet(t *testing.T, appCtx *models.AppContext) *appsv1.StatefulSet {
	t.Helper()
	oldContext := *appCtx
	oldContext.ConfigFiles = nil
	return (&AppMetadata{AppContext: &oldContext}).BuildStatefulSet()
}

func configSyncActionIndex(actions []k8stesting.Action, verb, resource string) int {
	for index, action := range actions {
		if action.GetVerb() == verb && action.GetResource().Resource == resource {
			return index
		}
	}
	return -1
}

func configSyncContainer(t *testing.T, template *corev1.PodTemplateSpec, name string) *corev1.Container {
	t.Helper()
	for index := range template.Spec.InitContainers {
		if template.Spec.InitContainers[index].Name == name {
			return &template.Spec.InitContainers[index]
		}
	}
	for index := range template.Spec.Containers {
		if template.Spec.Containers[index].Name == name {
			return &template.Spec.Containers[index]
		}
	}
	t.Fatalf("container %q not found", name)
	return nil
}

func configSyncTestAppWithPlugins(appType string, files []entities.AppConfigFile) *models.AppContext {
	appCtx := configSyncTestAppContext(appType, files)
	appCtx.Plugins = map[string]entities.Plugin{
		"plugin-init": {
			ID:         "plugin-init",
			Slug:       "init-helper",
			Image:      "busybox:1.36",
			PluginType: "init",
		},
		"plugin-sidecar": {
			ID:         "plugin-sidecar",
			Slug:       "log-helper",
			Image:      "busybox:1.36",
			PluginType: "sidecar",
		},
	}
	appCtx.AppPlugins = []entities.AppPlugin{
		{PluginID: "plugin-init", Enabled: true},
		{PluginID: "plugin-sidecar", Enabled: true},
	}
	return appCtx
}

func TestSyncConfigMapToK8sAddsConfigVolumeAndMountsToExistingWorkloads(t *testing.T) {
	files := []entities.AppConfigFile{{
		Slug:      "app.conf",
		MountPath: "/etc/demo/app.conf",
		Content:   "mode=production",
	}}
	for _, appType := range []string{app.AppTypeDeployment, app.AppTypeStatefulSet} {
		t.Run(appType, func(t *testing.T) {
			appCtx := configSyncTestAppWithPlugins(appType, files)
			var client *fake.Clientset
			if appType == app.AppTypeStatefulSet {
				client = fake.NewSimpleClientset(configSyncTestStatefulSet(t, appCtx))
			} else {
				client = fake.NewSimpleClientset(configSyncTestWorkload(t, appCtx))
			}

			require.NoError(t, syncConfigMapToK8s(context.Background(), client, appCtx))

			configMap, err := client.CoreV1().ConfigMaps("app-ns").Get(context.Background(), "demo-app-config", metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, "mode=production", configMap.Data["app.conf"])

			var template *corev1.PodTemplateSpec
			if appType == app.AppTypeStatefulSet {
				statefulSet, getErr := client.AppsV1().StatefulSets("app-ns").Get(context.Background(), "demo-app", metav1.GetOptions{})
				require.NoError(t, getErr)
				template = &statefulSet.Spec.Template
			} else {
				deployment, getErr := client.AppsV1().Deployments("app-ns").Get(context.Background(), "demo-app", metav1.GetOptions{})
				require.NoError(t, getErr)
				template = &deployment.Spec.Template
			}

			configVolumeFound := false
			for _, volume := range template.Spec.Volumes {
				if volume.Name != "config-files" {
					continue
				}
				configVolumeFound = true
				require.NotNil(t, volume.Projected)
				require.Len(t, volume.Projected.Sources, 1)
				require.NotNil(t, volume.Projected.Sources[0].ConfigMap)
				assert.Equal(t, "demo-app-config", volume.Projected.Sources[0].ConfigMap.Name)
			}
			assert.True(t, configVolumeFound)

			for _, name := range []string{"app-demo-app", "init-helper", "log-helper"} {
				container := configSyncContainer(t, template, name)
				require.Len(t, container.VolumeMounts, 1)
				assert.Equal(t, "config-files", container.VolumeMounts[0].Name)
				assert.Equal(t, "/etc/demo/app.conf", container.VolumeMounts[0].MountPath)
				assert.Equal(t, "app.conf", container.VolumeMounts[0].SubPath)
			}

			actions := client.Actions()
			createConfigMap := configSyncActionIndex(actions, "create", "configmaps")
			updateWorkload := configSyncActionIndex(actions, "update", map[string]string{
				app.AppTypeDeployment:  "deployments",
				app.AppTypeStatefulSet: "statefulsets",
			}[appType])
			assert.GreaterOrEqual(t, createConfigMap, 0)
			assert.Greater(t, updateWorkload, createConfigMap, "the resource must exist before the workload references it")
		})
	}
}

func TestSyncConfigMapToK8sRemovesConfigReferencesBeforeDeletingLastResources(t *testing.T) {
	files := []entities.AppConfigFile{{
		Slug:      "app.conf",
		MountPath: "/etc/demo/app.conf",
		Content:   "mode=production",
	}}
	for _, appType := range []string{app.AppTypeDeployment, app.AppTypeStatefulSet} {
		t.Run(appType, func(t *testing.T) {
			appCtx := configSyncTestAppContext(appType, nil)
			oldContext := *appCtx
			oldContext.ConfigFiles = files
			var client *fake.Clientset
			objects := []runtime.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "demo-app-config", Namespace: "app-ns"},
					Data:       map[string]string{"app.conf": "mode=production"},
				},
			}
			if appType == app.AppTypeStatefulSet {
				objects = append(objects, (&AppMetadata{AppContext: &oldContext}).BuildStatefulSet())
				client = fake.NewSimpleClientset(objects...)
			} else {
				objects = append(objects, (&AppMetadata{AppContext: &oldContext}).BuildDeployment())
				client = fake.NewSimpleClientset(objects...)
			}

			require.NoError(t, syncConfigMapToK8s(context.Background(), client, appCtx))
			_, err := client.CoreV1().ConfigMaps("app-ns").Get(context.Background(), "demo-app-config", metav1.GetOptions{})
			assert.Error(t, err)

			var template *corev1.PodTemplateSpec
			if appType == app.AppTypeStatefulSet {
				statefulSet, getErr := client.AppsV1().StatefulSets("app-ns").Get(context.Background(), "demo-app", metav1.GetOptions{})
				require.NoError(t, getErr)
				template = &statefulSet.Spec.Template
			} else {
				deployment, getErr := client.AppsV1().Deployments("app-ns").Get(context.Background(), "demo-app", metav1.GetOptions{})
				require.NoError(t, getErr)
				template = &deployment.Spec.Template
			}
			for _, volume := range template.Spec.Volumes {
				assert.NotEqual(t, "config-files", volume.Name)
			}
			for _, container := range append(template.Spec.InitContainers, template.Spec.Containers...) {
				for _, mount := range container.VolumeMounts {
					assert.NotEqual(t, "config-files", mount.Name)
				}
			}

			actions := client.Actions()
			updateWorkload := configSyncActionIndex(actions, "update", map[string]string{
				app.AppTypeDeployment:  "deployments",
				app.AppTypeStatefulSet: "statefulsets",
			}[appType])
			deleteConfigMap := configSyncActionIndex(actions, "delete", "configmaps")
			assert.GreaterOrEqual(t, updateWorkload, 0)
			assert.Greater(t, deleteConfigMap, updateWorkload, "the workload must stop referencing the resource before it is deleted")
		})
	}
}

func TestSyncConfigMapToK8sOrdersPublicSecretTransitionsSafely(t *testing.T) {
	tests := []struct {
		name          string
		oldSecret     bool
		currentSecret bool
		oldData       string
		currentData   string
	}{
		{name: "public to secret", oldSecret: false, currentSecret: true, oldData: "public", currentData: "secret"},
		{name: "secret to public", oldSecret: true, currentSecret: false, oldData: "old-secret", currentData: "public"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldFile := entities.AppConfigFile{Slug: "app.conf", MountPath: "/etc/demo/app.conf", Content: test.oldData, IsSecret: test.oldSecret}
			currentFile := oldFile
			currentFile.Content = test.currentData
			currentFile.IsSecret = test.currentSecret
			appCtx := configSyncTestAppContext(app.AppTypeDeployment, []entities.AppConfigFile{currentFile})
			oldContext := *appCtx
			oldContext.ConfigFiles = []entities.AppConfigFile{oldFile}
			oldWorkload := (&AppMetadata{AppContext: &oldContext}).BuildDeployment()
			objects := []runtime.Object{oldWorkload}
			if test.oldSecret {
				objects = append(objects, &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "demo-app-config-secret", Namespace: "app-ns"},
					Data:       map[string][]byte{"app.conf": []byte(test.oldData)},
				})
			} else {
				objects = append(objects, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "demo-app-config", Namespace: "app-ns"},
					Data:       map[string]string{"app.conf": test.oldData},
				})
			}
			client := fake.NewSimpleClientset(objects...)

			require.NoError(t, syncConfigMapToK8s(context.Background(), client, appCtx))
			workload, err := client.AppsV1().Deployments("app-ns").Get(context.Background(), "demo-app", metav1.GetOptions{})
			require.NoError(t, err)
			var configVolume *corev1.Volume
			for index := range workload.Spec.Template.Spec.Volumes {
				if workload.Spec.Template.Spec.Volumes[index].Name == "config-files" {
					configVolume = &workload.Spec.Template.Spec.Volumes[index]
				}
			}
			require.NotNil(t, configVolume)
			require.NotNil(t, configVolume.Projected)
			require.Len(t, configVolume.Projected.Sources, 1)
			if test.currentSecret {
				require.NotNil(t, configVolume.Projected.Sources[0].Secret)
				assert.Equal(t, "demo-app-config-secret", configVolume.Projected.Sources[0].Secret.Name)
				secret, getErr := client.CoreV1().Secrets("app-ns").Get(context.Background(), "demo-app-config-secret", metav1.GetOptions{})
				require.NoError(t, getErr)
				assert.Equal(t, []byte(test.currentData), secret.Data["app.conf"])
				_, getErr = client.CoreV1().ConfigMaps("app-ns").Get(context.Background(), "demo-app-config", metav1.GetOptions{})
				assert.Error(t, getErr)
			} else {
				require.NotNil(t, configVolume.Projected.Sources[0].ConfigMap)
				assert.Equal(t, "demo-app-config", configVolume.Projected.Sources[0].ConfigMap.Name)
				configMap, getErr := client.CoreV1().ConfigMaps("app-ns").Get(context.Background(), "demo-app-config", metav1.GetOptions{})
				require.NoError(t, getErr)
				assert.Equal(t, test.currentData, configMap.Data["app.conf"])
				_, getErr = client.CoreV1().Secrets("app-ns").Get(context.Background(), "demo-app-config-secret", metav1.GetOptions{})
				assert.Error(t, getErr)
			}

			actions := client.Actions()
			updateWorkload := configSyncActionIndex(actions, "update", "deployments")
			if test.currentSecret {
				createOrUpdateSecret := configSyncActionIndex(actions, "update", "secrets")
				if createOrUpdateSecret < 0 {
					createOrUpdateSecret = configSyncActionIndex(actions, "create", "secrets")
				}
				deleteConfigMap := configSyncActionIndex(actions, "delete", "configmaps")
				assert.GreaterOrEqual(t, createOrUpdateSecret, 0)
				assert.Greater(t, updateWorkload, createOrUpdateSecret)
				assert.Greater(t, deleteConfigMap, updateWorkload)
			} else {
				createOrUpdateConfigMap := configSyncActionIndex(actions, "update", "configmaps")
				if createOrUpdateConfigMap < 0 {
					createOrUpdateConfigMap = configSyncActionIndex(actions, "create", "configmaps")
				}
				deleteConfigSecret := configSyncActionIndex(actions, "delete", "secrets")
				assert.GreaterOrEqual(t, createOrUpdateConfigMap, 0)
				assert.Greater(t, updateWorkload, createOrUpdateConfigMap)
				assert.Greater(t, deleteConfigSecret, updateWorkload)
			}
		})
	}
}

func TestSyncConfigMapToK8sRejectsMissingEnvironmentSecretReferencedByWorkload(t *testing.T) {
	appCtx := configSyncTestAppContext(app.AppTypeDeployment, []entities.AppConfigFile{{
		Slug: "app.conf", MountPath: "/etc/demo/app.conf", Content: "mode=production",
	}})
	appCtx.EnvVars = []entities.AppEnvVar{{Key: "TOKEN", Value: "pending", IsSecret: true}}
	workload := configSyncTestWorkload(t, appCtx)
	workload.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{
		Name: "TOKEN",
		ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "demo-app-env-secret"},
			Key:                  "TOKEN",
		}},
	}}
	client := fake.NewSimpleClientset(workload)

	err := syncConfigMapToK8s(context.Background(), client, appCtx)
	require.ErrorContains(t, err, "missing environment Secret")
	for _, action := range client.Actions() {
		assert.NotEqual(t, "create", action.GetVerb())
		assert.NotEqual(t, "update", action.GetVerb())
		assert.NotEqual(t, "delete", action.GetVerb())
	}
}

func TestSyncConfigMapToK8sKeepsOldResourceWhenWorkloadUpdateFails(t *testing.T) {
	appCtx := configSyncTestAppContext(app.AppTypeDeployment, nil)
	oldContext := *appCtx
	oldContext.ConfigFiles = []entities.AppConfigFile{{
		Slug: "app.conf", MountPath: "/etc/demo/app.conf", Content: "old",
	}}
	client := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "demo-app-config", Namespace: "app-ns"},
			Data:       map[string]string{"app.conf": "old"},
		},
		(&AppMetadata{AppContext: &oldContext}).BuildDeployment(),
	)
	client.PrependReactor("update", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("workload update failed")
	})

	require.Error(t, syncConfigMapToK8s(context.Background(), client, appCtx))
	configMap, err := client.CoreV1().ConfigMaps("app-ns").Get(context.Background(), "demo-app-config", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "old", configMap.Data["app.conf"])
	for _, action := range client.Actions() {
		assert.False(t, action.GetVerb() == "delete" && action.GetResource().Resource == "configmaps")
	}
}

func TestBuildConfigRevisionTracksConfigFileShapeAndIgnoresOrder(t *testing.T) {
	files := []entities.AppConfigFile{
		{Slug: "b.conf", MountPath: "/etc/b.conf", Content: "b"},
		{Slug: "a.conf", MountPath: "/etc/a.conf", Content: "a"},
	}
	first := configSyncTestAppContext(app.AppTypeDeployment, files)
	reversed := configSyncTestAppContext(app.AppTypeDeployment, []entities.AppConfigFile{files[1], files[0]})
	firstRevision, err := (&AppMetadata{AppContext: first}).BuildConfigRevision()
	require.NoError(t, err)
	reversedRevision, err := (&AppMetadata{AppContext: reversed}).BuildConfigRevision()
	require.NoError(t, err)
	assert.Equal(t, firstRevision, reversedRevision)

	variants := []struct {
		name   string
		mutate func(*entities.AppConfigFile)
	}{
		{name: "slug", mutate: func(file *entities.AppConfigFile) { file.Slug = "renamed.conf" }},
		{name: "mount path", mutate: func(file *entities.AppConfigFile) { file.MountPath = "/opt/renamed.conf" }},
		{name: "secret type", mutate: func(file *entities.AppConfigFile) { file.IsSecret = true }},
	}
	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			variantContext := configSyncTestAppContext(app.AppTypeDeployment, files)
			variant.mutate(&variantContext.ConfigFiles[0])
			variantRevision, revisionErr := (&AppMetadata{AppContext: variantContext}).BuildConfigRevision()
			require.NoError(t, revisionErr)
			assert.NotEqual(t, firstRevision, variantRevision)
		})
	}
}
