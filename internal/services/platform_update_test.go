package services

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	ktesting "k8s.io/client-go/testing"
)

func setupPlatformUpdateServiceTestDB(t *testing.T) {
	t.Helper()
	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(
		&entities.SystemSetting{},
		&entities.OperationLog{},
		&entities.Notification{},
		&entities.User{},
	))
	db.DB = testDB
}

func TestPlatformUpdateConfigDefaults(t *testing.T) {
	setupPlatformUpdateServiceTestDB(t)

	cfg, err := GetPlatformUpdateConfig()
	require.NoError(t, err)

	assert.Equal(t, "ghcr.io/ketches/ketches/ketches-api", cfg.API.ImageRepository)
	assert.Equal(t, "ketches", cfg.API.Namespace)
	assert.Equal(t, "ketches-api", cfg.API.DeploymentName)
	assert.Equal(t, "ketches-api", cfg.API.ContainerName)

	assert.Equal(t, "ghcr.io/ketches/ketches/ketches-ui", cfg.UI.ImageRepository)
	assert.Equal(t, "ketches", cfg.UI.Namespace)
	assert.Equal(t, "ketches-ui", cfg.UI.DeploymentName)
	assert.Equal(t, "ketches-ui", cfg.UI.ContainerName)
}

func TestPlatformUpdateSemverFilteringAndSorting(t *testing.T) {
	got := filterPlatformUpdateVersions([]string{
		"latest",
		"v1.2.0-beta.1",
		"1.2.0",
		"sha-deadbee",
		"main-123",
		"v1.10.0",
		"v1.2.3",
	})

	assert.Equal(t, []string{
		"v1.10.0",
		"v1.2.3",
		"1.2.0",
		"v1.2.0-beta.1",
	}, got)

	shared := recommendedSharedPlatformVersion(
		[]string{"v1.3.0", "v1.2.0"},
		[]string{"v1.3.0-beta.1", "v1.2.0"},
	)
	assert.Equal(t, "v1.2.0", shared)
}

func TestPlatformUpdateStatusPrefersRuntimeDeploymentTags(t *testing.T) {
	setupPlatformUpdateServiceTestDB(t)

	originalVersion := app.Version
	app.Version = "v0.9.0"
	t.Cleanup(func() { app.Version = originalVersion })

	originalListTags := platformUpdateListTags
	originalInClusterConfig := platformUpdateInClusterConfig
	originalNewKubeClient := platformUpdateNewKubeClientForConfig
	t.Cleanup(func() {
		platformUpdateListTags = originalListTags
		platformUpdateInClusterConfig = originalInClusterConfig
		platformUpdateNewKubeClientForConfig = originalNewKubeClient
	})

	platformUpdateListTags = func(repo string) ([]string, error) {
		switch repo {
		case "ghcr.io/ketches/ketches/ketches-api":
			return []string{"v1.2.0", "v1.1.0"}, nil
		case "ghcr.io/ketches/ketches/ketches-ui":
			return []string{"v1.3.0", "v1.2.0"}, nil
		default:
			return nil, errors.New("unexpected repo")
		}
	}
	platformUpdateInClusterConfig = func() (*rest.Config, error) {
		return &rest.Config{Host: "https://cluster.local"}, nil
	}

	kubeClient := fake.NewSimpleClientset(
		newPlatformUpdateDeployment("ketches", "ketches-api", "ketches-api", "ghcr.io/ketches/ketches/ketches-api:v1.2.0"),
		newPlatformUpdateDeployment("ketches", "ketches-ui", "ketches-ui", "ghcr.io/ketches/ketches/ketches-ui:v1.3.0"),
	)
	platformUpdateNewKubeClientForConfig = func(*rest.Config) (platformUpdateKubeClient, error) {
		return kubeClient, nil
	}

	status, err := GetPlatformUpdateStatus()
	require.NoError(t, err)

	assert.True(t, status.RunningInKubernetes)
	assert.True(t, status.CanRollout)
	assert.Equal(t, "deployment", status.API.VersionSource)
	assert.Equal(t, "v1.2.0", status.API.CurrentVersion)
	assert.Equal(t, "deployment", status.UI.VersionSource)
	assert.Equal(t, "v1.3.0", status.UI.CurrentVersion)
	assert.Equal(t, "v1.2.0", status.RecommendedSharedVersion)
}

func TestPlatformUpdateStatusUsesEmptyVersionListsWhenRegistryLookupFails(t *testing.T) {
	setupPlatformUpdateServiceTestDB(t)

	originalListTags := platformUpdateListTags
	originalInClusterConfig := platformUpdateInClusterConfig
	t.Cleanup(func() {
		platformUpdateListTags = originalListTags
		platformUpdateInClusterConfig = originalInClusterConfig
	})

	platformUpdateListTags = func(string) ([]string, error) {
		return nil, errors.New("registry unavailable")
	}
	platformUpdateInClusterConfig = func() (*rest.Config, error) {
		return nil, errors.New("not in cluster")
	}

	status, err := GetPlatformUpdateStatus()
	require.NoError(t, err)

	assert.Empty(t, status.API.AvailableVersions)
	assert.Empty(t, status.UI.AvailableVersions)
	assert.NotNil(t, status.API.AvailableVersions)
	assert.NotNil(t, status.UI.AvailableVersions)
}

func TestPlatformUpdateRolloutRollsBackUISpecWhenAPIPatchFails(t *testing.T) {
	setupPlatformUpdateServiceTestDB(t)

	originalListTags := platformUpdateListTags
	originalInClusterConfig := platformUpdateInClusterConfig
	originalNewKubeClient := platformUpdateNewKubeClientForConfig
	t.Cleanup(func() {
		platformUpdateListTags = originalListTags
		platformUpdateInClusterConfig = originalInClusterConfig
		platformUpdateNewKubeClientForConfig = originalNewKubeClient
	})

	platformUpdateListTags = func(repo string) ([]string, error) {
		return []string{"v2.0.0", "v1.0.0"}, nil
	}
	platformUpdateInClusterConfig = func() (*rest.Config, error) {
		return &rest.Config{Host: "https://cluster.local"}, nil
	}

	kubeClient := fake.NewSimpleClientset(
		newPlatformUpdateDeployment("ketches", "ketches-api", "ketches-api", "ghcr.io/ketches/ketches/ketches-api:v1.0.0"),
		newPlatformUpdateDeployment("ketches", "ketches-ui", "ketches-ui", "ghcr.io/ketches/ketches/ketches-ui:v1.0.0"),
	)

	var uiPatchCount int
	var apiPatchCount int
	deploymentResource := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}

	kubeClient.PrependReactor("patch", "deployments", func(action ktesting.Action) (bool, runtime.Object, error) {
		patchAction := action.(ktesting.PatchAction)
		switch patchAction.GetName() {
		case "ketches-ui":
			uiPatchCount++
			obj, getErr := kubeClient.Tracker().Get(deploymentResource, "ketches", "ketches-ui")
			require.NoError(t, getErr)
			deployment := obj.(*appsv1.Deployment).DeepCopy()
			if uiPatchCount == 1 {
				deployment.Spec.Template.Spec.Containers[0].Image = "ghcr.io/ketches/ketches/ketches-ui:v2.0.0"
			} else {
				deployment.Spec.Template.Spec.Containers[0].Image = "ghcr.io/ketches/ketches/ketches-ui:v1.0.0"
			}
			return true, deployment, kubeClient.Tracker().Update(deploymentResource, deployment, "ketches")
		case "ketches-api":
			apiPatchCount++
			return true, nil, errors.New("api patch failed")
		default:
			return false, nil, nil
		}
	})

	platformUpdateNewKubeClientForConfig = func(*rest.Config) (platformUpdateKubeClient, error) {
		return kubeClient, nil
	}

	_, err := TriggerPlatformRollout(&models.TriggerPlatformRolloutRequest{
		SharedVersion: "v2.0.0",
	}, &app.Claims{
		UserID:   "admin-1",
		Username: "admin",
		Role:     app.UserRoleAdmin,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api patch failed")
	assert.Equal(t, 2, uiPatchCount)
	assert.Equal(t, 1, apiPatchCount)

	uiDeployment, getErr := kubeClient.AppsV1().Deployments("ketches").Get(t.Context(), "ketches-ui", metav1.GetOptions{})
	require.NoError(t, getErr)
	assert.Equal(t, "ghcr.io/ketches/ketches/ketches-ui:v1.0.0", uiDeployment.Spec.Template.Spec.Containers[0].Image)
}

func TestPlatformUpdateAutoCheckCreatesNotificationsWhenRecommendedVersionChanges(t *testing.T) {
	setupPlatformUpdateServiceTestDB(t)
	createPlatformUpdateTestUser(t, "admin-1", "admin-1", app.UserRoleAdmin)
	createPlatformUpdateTestUser(t, "admin-2", "admin-2", app.UserRoleAdmin)
	createPlatformUpdateTestUser(t, "user-1", "user-1", app.UserRoleUser)

	originalListTags := platformUpdateListTags
	originalInClusterConfig := platformUpdateInClusterConfig
	t.Cleanup(func() {
		platformUpdateListTags = originalListTags
		platformUpdateInClusterConfig = originalInClusterConfig
	})

	platformUpdateListTags = func(repo string) ([]string, error) {
		switch repo {
		case "ghcr.io/ketches/ketches/ketches-api", "ghcr.io/ketches/ketches/ketches-ui":
			return []string{"v1.2.0", "v1.0.0"}, nil
		default:
			return nil, errors.New("unexpected repo")
		}
	}
	platformUpdateInClusterConfig = func() (*rest.Config, error) {
		return nil, errors.New("not in cluster")
	}

	status, err := CheckPlatformUpdate(&models.CheckPlatformUpdateRequest{Mode: "auto"}, &app.Claims{
		UserID:   "admin-1",
		Username: "admin-1",
		Role:     app.UserRoleAdmin,
	})
	require.NoError(t, err)
	assert.Equal(t, "v1.2.0", status.RecommendedSharedVersion)

	var notifications []entities.Notification
	require.NoError(t, db.DB.Order("recipient_id").Find(&notifications).Error)
	require.Len(t, notifications, 2)
	assert.Equal(t, "admin-1", notifications[0].RecipientID)
	assert.Equal(t, "platform_update_available", notifications[0].EventType)
	assert.Equal(t, "admin-2", notifications[1].RecipientID)

	setting := loadPlatformUpdateSystemSetting(t, "platform_update_notification_state")
	var state map[string]string
	require.NoError(t, json.Unmarshal([]byte(setting.Value), &state))
	assert.Equal(t, "v1.2.0", state["last_notified_recommended_version"])
}

func TestPlatformUpdateAutoCheckSkipsNotificationsWhenRecommendedVersionIsUnchanged(t *testing.T) {
	setupPlatformUpdateServiceTestDB(t)
	createPlatformUpdateTestUser(t, "admin-1", "admin-1", app.UserRoleAdmin)

	require.NoError(t, db.DB.Create(&entities.SystemSetting{
		Base:  entities.Base{ID: "state-1"},
		Key:   "platform_update_notification_state",
		Value: `{"last_notified_recommended_version":"v1.2.0"}`,
	}).Error)

	originalListTags := platformUpdateListTags
	originalInClusterConfig := platformUpdateInClusterConfig
	t.Cleanup(func() {
		platformUpdateListTags = originalListTags
		platformUpdateInClusterConfig = originalInClusterConfig
	})

	platformUpdateListTags = func(repo string) ([]string, error) {
		switch repo {
		case "ghcr.io/ketches/ketches/ketches-api", "ghcr.io/ketches/ketches/ketches-ui":
			return []string{"v1.2.0", "v1.0.0"}, nil
		default:
			return nil, errors.New("unexpected repo")
		}
	}
	platformUpdateInClusterConfig = func() (*rest.Config, error) {
		return nil, errors.New("not in cluster")
	}

	status, err := CheckPlatformUpdate(&models.CheckPlatformUpdateRequest{Mode: "auto"}, &app.Claims{
		UserID:   "admin-1",
		Username: "admin-1",
		Role:     app.UserRoleAdmin,
	})
	require.NoError(t, err)
	assert.Equal(t, "v1.2.0", status.RecommendedSharedVersion)

	var count int64
	require.NoError(t, db.DB.Model(&entities.Notification{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestPlatformUpdateManualCheckNeverCreatesNotifications(t *testing.T) {
	setupPlatformUpdateServiceTestDB(t)
	createPlatformUpdateTestUser(t, "admin-1", "admin-1", app.UserRoleAdmin)

	originalListTags := platformUpdateListTags
	originalInClusterConfig := platformUpdateInClusterConfig
	t.Cleanup(func() {
		platformUpdateListTags = originalListTags
		platformUpdateInClusterConfig = originalInClusterConfig
	})

	platformUpdateListTags = func(repo string) ([]string, error) {
		switch repo {
		case "ghcr.io/ketches/ketches/ketches-api", "ghcr.io/ketches/ketches/ketches-ui":
			return []string{"v2.0.0", "v1.0.0"}, nil
		default:
			return nil, errors.New("unexpected repo")
		}
	}
	platformUpdateInClusterConfig = func() (*rest.Config, error) {
		return nil, errors.New("not in cluster")
	}

	status, err := CheckPlatformUpdate(&models.CheckPlatformUpdateRequest{Mode: "manual"}, &app.Claims{
		UserID:   "admin-1",
		Username: "admin-1",
		Role:     app.UserRoleAdmin,
	})
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", status.RecommendedSharedVersion)

	var notificationCount int64
	require.NoError(t, db.DB.Model(&entities.Notification{}).Count(&notificationCount).Error)
	assert.Equal(t, int64(0), notificationCount)

	var settingCount int64
	require.NoError(t, db.DB.Model(&entities.SystemSetting{}).
		Where("key = ?", "platform_update_notification_state").
		Count(&settingCount).Error)
	assert.Equal(t, int64(0), settingCount)
}

func newPlatformUpdateDeployment(namespace, name, containerName, image string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  containerName,
							Image: image,
						},
					},
				},
			},
		},
	}
}

func createPlatformUpdateTestUser(t *testing.T, id, username, role string) {
	t.Helper()
	require.NoError(t, db.DB.Create(&entities.User{
		Base:     entities.Base{ID: id},
		Username: username,
		Email:    username + "@example.com",
		Password: "secret",
		Role:     role,
	}).Error)
}

func loadPlatformUpdateSystemSetting(t *testing.T, key string) entities.SystemSetting {
	t.Helper()
	var setting entities.SystemSetting
	require.NoError(t, db.DB.Where("key = ?", key).First(&setting).Error)
	return setting
}
