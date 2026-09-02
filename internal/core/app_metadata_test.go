package core

import (
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestBuildPluginContainerUsesAppPluginOverridesAndResources(t *testing.T) {
	metadata := AppMetadata{
		AppContext: &models.AppContext{
			EnvVars: []entities.AppEnvVar{
				{Key: "APP_SHARED", Value: "shared"},
			},
		},
	}

	plugin := &entities.Plugin{
		Slug:            "log-sidecar",
		Image:           "busybox:1.36",
		ImagePullPolicy: "IfNotPresent",
		EnvVars:         `[{"key":"PLUGIN_MODE","value":"template"}]`,
	}
	appPlugin := &entities.AppPlugin{
		EnvVars:       `[{"key":"PLUGIN_MODE","value":"runtime"},{"key":"PLUGIN_LEVEL","value":"debug"}]`,
		RequestCPU:    150,
		RequestMemory: 64,
		LimitCPU:      300,
		LimitMemory:   128,
	}

	container := metadata.buildPluginContainer(plugin, appPlugin)

	require.Equal(t, int64(150), container.Resources.Requests.Cpu().MilliValue())
	require.Equal(t, "64Mi", container.Resources.Requests.Memory().String())
	require.Equal(t, int64(300), container.Resources.Limits.Cpu().MilliValue())
	require.Equal(t, "128Mi", container.Resources.Limits.Memory().String())

	envMap := make(map[string]string, len(container.Env))
	for _, envVar := range container.Env {
		envMap[envVar.Name] = envVar.Value
	}

	assert.Equal(t, "shared", envMap["APP_SHARED"])
	assert.Equal(t, "runtime", envMap["PLUGIN_MODE"])
	assert.Equal(t, "debug", envMap["PLUGIN_LEVEL"])
}

func TestBuildVolumesCreatesConfiguredVolumeSources(t *testing.T) {
	metadata := AppMetadata{AppContext: &models.AppContext{
		App: entities.App{AppType: app.AppTypeDeployment},
		Volumes: []entities.AppVolume{
			{Slug: "cache", VolumeType: app.VolumeTypeEmptyDir},
			{Slug: "node-data", VolumeType: app.VolumeTypeHostPath, HostPath: "/var/lib/ketches"},
			{Slug: "data", VolumeType: app.VolumeTypePVC},
		},
	}}

	volumes := metadata.buildVolumes()
	require.Len(t, volumes, 3)
	byName := make(map[string]corev1.Volume, len(volumes))
	for _, volume := range volumes {
		byName[volume.Name] = volume
	}
	require.NotNil(t, byName["cache"].EmptyDir)
	require.NotNil(t, byName["node-data"].HostPath)
	assert.Equal(t, "/var/lib/ketches", byName["node-data"].HostPath.Path)
	require.NotNil(t, byName["data"].PersistentVolumeClaim)
	assert.Equal(t, "data", byName["data"].PersistentVolumeClaim.ClaimName)
}

func TestBuildPVCUsesStorageConfiguration(t *testing.T) {
	metadata := AppMetadata{AppContext: &models.AppContext{EnvContext: models.EnvContext{
		Env: entities.Env{ClusterNamespace: "project-prod"},
	}}}
	pvc := metadata.BuildPVC(entities.AppVolume{
		Slug:         "database",
		Capacity:     20,
		StorageClass: "fast-ssd",
		VolumeMode:   string(corev1.PersistentVolumeBlock),
		AccessModes:  "ReadWriteMany,ReadOnlyMany",
	})

	require.NotNil(t, pvc.Spec.StorageClassName)
	assert.Equal(t, "fast-ssd", *pvc.Spec.StorageClassName)
	require.NotNil(t, pvc.Spec.VolumeMode)
	assert.Equal(t, corev1.PersistentVolumeBlock, *pvc.Spec.VolumeMode)
	assert.Equal(t, []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany, corev1.ReadOnlyMany}, pvc.Spec.AccessModes)
	assert.Equal(t, "20Gi", pvc.Spec.Resources.Requests.Storage().String())
}

func TestValidateAppVolumePolicyRejectsHostPathWhenDisabled(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() { app.Config = originalConfig })
	app.Config.AllowHostPathVolumes = false
	appCtx := &models.AppContext{Volumes: []entities.AppVolume{{
		Slug: "node-data", VolumeType: app.VolumeTypeHostPath, HostPath: "/var/lib/ketches",
	}}}

	require.ErrorContains(t, validateAppVolumePolicy(appCtx), "disabled")
	app.Config.AllowHostPathVolumes = true
	require.NoError(t, validateAppVolumePolicy(appCtx))
}

func TestBuildContainerUsesVolumeDeviceForBlockPVC(t *testing.T) {
	metadata := AppMetadata{AppContext: &models.AppContext{
		App: entities.App{Slug: "database"},
		Volumes: []entities.AppVolume{
			{Slug: "disk", MountPath: "/dev/xvda", VolumeType: app.VolumeTypePVC, VolumeMode: string(corev1.PersistentVolumeBlock)},
			{Slug: "cache", MountPath: "/cache", VolumeType: app.VolumeTypeEmptyDir},
		},
	}}

	container := metadata.buildContainer()
	require.Len(t, container.VolumeDevices, 1)
	assert.Equal(t, "disk", container.VolumeDevices[0].Name)
	assert.Equal(t, "/dev/xvda", container.VolumeDevices[0].DevicePath)
	require.Len(t, container.VolumeMounts, 1)
	assert.Equal(t, "cache", container.VolumeMounts[0].Name)
}

func TestBuildPluginContainerFallsBackToPluginTemplateEnvVars(t *testing.T) {
	metadata := AppMetadata{
		AppContext: &models.AppContext{},
	}

	plugin := &entities.Plugin{
		Slug:    "init-sidecar",
		Image:   "busybox:1.36",
		EnvVars: `[{"key":"PLUGIN_MODE","value":"template"}]`,
	}
	appPlugin := &entities.AppPlugin{
		EnvVars: "null",
	}

	container := metadata.buildPluginContainer(plugin, appPlugin)

	require.Len(t, container.Env, 1)
	assert.Equal(t, "PLUGIN_MODE", container.Env[0].Name)
	assert.Equal(t, "template", container.Env[0].Value)
}

func newConfigRevisionAppContext(appType string) *models.AppContext {
	return &models.AppContext{
		App: entities.App{
			Base:           entities.Base{ID: "app-1"},
			Slug:           "demo-app",
			AppType:        appType,
			ContainerImage: "busybox:1.36",
			Replicas:       1,
			RequestCPU:     100,
			RequestMemory:  128,
			LimitCPU:       200,
			LimitMemory:    256,
		},
		EnvContext: models.EnvContext{Env: entities.Env{ClusterNamespace: "app-ns"}},
		EnvVars: []entities.AppEnvVar{
			{Key: "PUBLIC_MODE", Value: "production"},
			{Key: "SECRET_TOKEN", Value: "secret-one", IsSecret: true},
		},
		ConfigFiles: []entities.AppConfigFile{
			{Slug: "app.conf", Content: "mode=production"},
			{Slug: "credentials.conf", Content: "password-one", IsSecret: true},
		},
	}
}

func TestBuildConfigRevisionIsStableAndTracksSecretAndPublicConfiguration(t *testing.T) {
	appCtx := newConfigRevisionAppContext(app.AppTypeDeployment)
	metadata := &AppMetadata{AppContext: appCtx}

	revision, err := metadata.BuildConfigRevision()
	require.NoError(t, err)
	require.Len(t, revision, 64)

	stableRevision, err := metadata.BuildConfigRevision()
	require.NoError(t, err)
	assert.Equal(t, revision, stableRevision)

	appCtx.EnvVars[1].Value = "secret-two"
	secretRevision, err := metadata.BuildConfigRevision()
	require.NoError(t, err)
	assert.NotEqual(t, revision, secretRevision)

	appCtx.EnvVars[1].Value = "secret-one"
	appCtx.ConfigFiles[0].Content = "mode=staging"
	publicRevision, err := metadata.BuildConfigRevision()
	require.NoError(t, err)
	assert.NotEqual(t, revision, publicRevision)
}

func TestBuildWorkloadIncludesConfigRevisionAnnotation(t *testing.T) {
	for _, appType := range []string{app.AppTypeDeployment, app.AppTypeStatefulSet} {
		t.Run(appType, func(t *testing.T) {
			metadata := &AppMetadata{AppContext: newConfigRevisionAppContext(appType), configRevision: "revision-123"}
			if appType == app.AppTypeStatefulSet {
				statefulSet := metadata.BuildStatefulSet()
				assert.Equal(t, "revision-123", statefulSet.Spec.Template.Annotations[configRevisionAnnotation])
				return
			}
			deployment := metadata.BuildDeployment()
			assert.Equal(t, "revision-123", deployment.Spec.Template.Annotations[configRevisionAnnotation])
		})
	}
}
