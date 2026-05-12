package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"helm.sh/helm/v3/pkg/storage/driver"
)

func setupExtensionServiceTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	sqlDB, err := testDB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	db.DB = testDB
	require.NoError(t, testDB.AutoMigrate(
		&entities.Cluster{},
		&entities.Extension{},
		&entities.ClusterExtension{},
		&entities.ClusterGatewayProvider{},
	))
}

func TestListExtensionsCountsClusterExtensionRowsWithoutDeletedAtColumn(t *testing.T) {
	setupExtensionServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Extension{
		ID:     "ext-1",
		Slug:   "envoy-gateway",
		Name:   "Envoy Gateway",
		OCIUrl: "oci://example.com/envoy-gateway",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ClusterExtension{
		ID:          "ce-1",
		ClusterID:   "cluster-1",
		ExtensionID: "ext-1",
		Namespace:   "ketches-extensions",
		ReleaseName: "envoy-gateway",
		Status:      entities.ClusterExtensionStatusDeployed,
	}).Error)

	items, err := ListExtensions()
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, 1, items[0].InstallCount)
}

func TestGetInstalledClustersForExtensionQueriesClusterExtensionsWithoutDeletedAtColumn(t *testing.T) {
	setupExtensionServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Extension{
		ID:     "ext-1",
		Slug:   "envoy-gateway",
		Name:   "Envoy Gateway",
		OCIUrl: "oci://example.com/envoy-gateway",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "cluster-one",
		Name:       "Cluster One",
		KubeConfig: "enc:v1:test",
		Enabled:    true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ClusterExtension{
		ID:          "ce-1",
		ClusterID:   "cluster-1",
		ExtensionID: "ext-1",
		Namespace:   "ketches-extensions",
		ReleaseName: "envoy-gateway",
		Version:     "1.0.0",
		Status:      entities.ClusterExtensionStatusDeployed,
	}).Error)

	clusters, err := GetInstalledClustersForExtension("ext-1")
	require.NoError(t, err)
	require.Len(t, clusters, 1)
	assert.Equal(t, "cluster-1", clusters[0].ClusterID)
	assert.Equal(t, "Cluster One", clusters[0].ClusterName)
}

func TestInstallClusterExtensionNormalizesReleaseNameAndStoresName(t *testing.T) {
	setupExtensionServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Extension{
		ID:     "ext-1",
		Slug:   "envoy-gateway",
		Name:   "Envoy Gateway",
		OCIUrl: "oci://example.com/envoy-gateway",
	}).Error)

	originalLaunch := launchClusterExtensionInstall
	called := false
	launchClusterExtensionInstall = func(clusterID string, ext *entities.Extension, record *entities.ClusterExtension) {
		called = true
		assert.Equal(t, "envoy-gateway", record.ReleaseName)
		assert.Equal(t, "Envoy Gateway", record.Name)
	}
	t.Cleanup(func() {
		launchClusterExtensionInstall = originalLaunch
	})

	result, err := InstallClusterExtension("cluster-1", &models.InstallExtensionRequest{
		ExtensionID: "ext-1",
		ReleaseName: "envoyGateway",
		Namespace:   "ketches-extensions",
	}, "user-1")
	require.NoError(t, err)
	assert.True(t, called)
	require.NotNil(t, result)
	assert.Equal(t, "envoy-gateway", result.ReleaseName)
	assert.Equal(t, "Envoy Gateway", result.Name)

	var stored entities.ClusterExtension
	require.NoError(t, db.DB.First(&stored, "id = ?", result.ID).Error)
	assert.Equal(t, "envoy-gateway", stored.ReleaseName)
	assert.Equal(t, "Envoy Gateway", stored.Name)
}

func TestInstallClusterExtensionRejectsCompletelyInvalidReleaseName(t *testing.T) {
	setupExtensionServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Extension{
		ID:     "ext-1",
		Slug:   "envoy-gateway",
		Name:   "Envoy Gateway",
		OCIUrl: "oci://example.com/envoy-gateway",
	}).Error)

	result, err := InstallClusterExtension("cluster-1", &models.InstallExtensionRequest{
		ExtensionID: "ext-1",
		ReleaseName: "!!!",
		Namespace:   "ketches-extensions",
	}, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid release name")
}

func TestInstallClusterExtensionRejectsDuplicateInstallWithoutDeletedAtColumn(t *testing.T) {
	setupExtensionServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Extension{
		ID:     "ext-1",
		Slug:   "envoy-gateway",
		Name:   "Envoy Gateway",
		OCIUrl: "oci://example.com/envoy-gateway",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ClusterExtension{
		ID:          "ce-1",
		ClusterID:   "cluster-1",
		ExtensionID: "ext-1",
		Namespace:   "ketches-extensions",
		ReleaseName: "envoy-gateway",
		Status:      entities.ClusterExtensionStatusDeployed,
	}).Error)

	result, err := InstallClusterExtension("cluster-1", &models.InstallExtensionRequest{
		ExtensionID: "ext-1",
		ReleaseName: "envoy-gateway",
		Namespace:   "ketches-extensions",
	}, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "extension already installed")
}

func TestRetryClusterExtensionNormalizesInvalidReleaseName(t *testing.T) {
	setupExtensionServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Extension{
		ID:     "ext-1",
		Slug:   "envoy-gateway",
		Name:   "Envoy Gateway",
		OCIUrl: "oci://example.com/envoy-gateway",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ClusterExtension{
		ID:           "ce-invalid-release",
		ClusterID:    "cluster-1",
		ExtensionID:  "ext-1",
		Namespace:    "ketches-extensions",
		ReleaseName:  "envoyGateway",
		Status:       entities.ClusterExtensionStatusFailed,
		Phase:        "installing",
		ErrorMessage: "release name is invalid",
	}).Error)

	originalLaunch := launchClusterExtensionInstall
	called := false
	launchClusterExtensionInstall = func(clusterID string, ext *entities.Extension, record *entities.ClusterExtension) {
		called = true
		assert.Equal(t, "envoy-gateway", record.ReleaseName)
	}
	t.Cleanup(func() {
		launchClusterExtensionInstall = originalLaunch
	})

	result, err := RetryClusterExtension("cluster-1", "ce-invalid-release", nil)
	require.NoError(t, err)
	assert.True(t, called)
	require.NotNil(t, result)
	assert.Equal(t, "envoy-gateway", result.ReleaseName)
	assert.Equal(t, "Envoy Gateway", result.Name)

	var stored entities.ClusterExtension
	require.NoError(t, db.DB.First(&stored, "id = ?", "ce-invalid-release").Error)
	assert.Equal(t, "envoy-gateway", stored.ReleaseName)
	assert.Equal(t, "Envoy Gateway", stored.Name)
}

func TestRetryClusterExtensionQueuesInstallRetryForFailedInstall(t *testing.T) {
	setupExtensionServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Extension{
		ID:     "ext-1",
		Slug:   "envoy-gateway",
		Name:   "Envoy Gateway",
		OCIUrl: "oci://example.com/envoy-gateway",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ClusterExtension{
		ID:              "ce-1",
		ClusterID:       "cluster-1",
		ExtensionID:     "ext-1",
		Namespace:       "ketches-extensions",
		ReleaseName:     "envoy-gateway",
		CreateNamespace: true,
		Status:          entities.ClusterExtensionStatusFailed,
		Phase:           "installing",
		ErrorMessage:    "boom",
	}).Error)

	originalLaunch := launchClusterExtensionInstall
	called := false
	launchClusterExtensionInstall = func(clusterID string, ext *entities.Extension, record *entities.ClusterExtension) {
		called = true
	}
	t.Cleanup(func() {
		launchClusterExtensionInstall = originalLaunch
	})

	result, err := RetryClusterExtension("cluster-1", "ce-1", nil)
	require.NoError(t, err)
	assert.True(t, called)
	require.NotNil(t, result)
	assert.Equal(t, string(entities.ClusterExtensionStatusPending), result.Status)
	assert.Equal(t, "installing", result.Phase)
	assert.True(t, result.CreateNamespace)

	var stored entities.ClusterExtension
	require.NoError(t, db.DB.First(&stored, "id = ?", "ce-1").Error)
	assert.Equal(t, entities.ClusterExtensionStatusPending, stored.Status)
	assert.Equal(t, "installing", stored.Phase)
	assert.Equal(t, "", stored.ErrorMessage)
}

func TestDefaultLaunchClusterExtensionInstallCreatesGatewayProviderForGatewayExtension(t *testing.T) {
	setupExtensionServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "cluster-one",
		Name:       "Cluster One",
		KubeConfig: "enc:v1:test",
		Enabled:    true,
	}).Error)

	record := &entities.ClusterExtension{
		ID:          "ce-1",
		ClusterID:   "cluster-1",
		ExtensionID: "ext-1",
		Name:        "Envoy Gateway",
		Namespace:   "ketches-extensions",
		ReleaseName: "envoy-gateway",
		Status:      entities.ClusterExtensionStatusPending,
	}
	require.NoError(t, db.DB.Create(record).Error)

	ext := &entities.Extension{
		ID:           "ext-1",
		Slug:         "envoyGateway",
		Name:         "Envoy Gateway",
		Capabilities: `["gateway-api"]`,
		Metadata:     entities.JSONBlob(`{"gateway_api":{"controller_name":"gateway.envoyproxy.io/gatewayclass-controller"}}`),
	}

	originalExecuteInstall := executeClusterExtensionInstall
	originalEnsureGatewayClass := ensureGatewayClassForExtensionInstall
	originalEnsureSharedGateway := ensureSharedGatewayForExtensionInstall
	t.Cleanup(func() {
		executeClusterExtensionInstall = originalExecuteInstall
		ensureGatewayClassForExtensionInstall = originalEnsureGatewayClass
		ensureSharedGatewayForExtensionInstall = originalEnsureSharedGateway
	})

	executeClusterExtensionInstall = func(clusterID string, ext *entities.Extension, record *entities.ClusterExtension) error {
		return nil
	}
	ensureGatewayClassForExtensionInstall = func(clusterID, gatewayClassName, controllerName string) error {
		return nil
	}
	ensureSharedGatewayForExtensionInstall = func(clusterID string) error {
		return nil
	}

	defaultLaunchClusterExtensionInstall("cluster-1", ext, record)

	require.Eventually(t, func() bool {
		var provider entities.ClusterGatewayProvider
		if err := db.DB.Where("cluster_id = ? AND cluster_extension_id = ?", "cluster-1", "ce-1").First(&provider).Error; err != nil {
			return false
		}
		return provider.SourceType == "managed" && provider.IsDefault && provider.GatewayClassName == "ketches-envoy-gateway"
	}, time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		var stored entities.ClusterExtension
		if err := db.DB.First(&stored, "id = ?", "ce-1").Error; err != nil {
			return false
		}
		return stored.Status == entities.ClusterExtensionStatusDeployed && stored.Phase == "" && stored.ErrorMessage == ""
	}, time.Second, 10*time.Millisecond)
}

func TestDefaultLaunchClusterExtensionUninstallDeletesRecordWhenHelmReleaseIsMissing(t *testing.T) {
	setupExtensionServiceTestDB(t)

	record := &entities.ClusterExtension{
		ID:          "ce-1",
		ClusterID:   "cluster-1",
		ExtensionID: "ext-1",
		Namespace:   "ketches-extensions",
		ReleaseName: "envoy-gateway",
		Status:      entities.ClusterExtensionStatusUninstalling,
		Phase:       "uninstalling",
	}
	require.NoError(t, db.DB.Create(record).Error)

	originalExecuteUninstall := executeClusterExtensionUninstall
	executeClusterExtensionUninstall = func(clusterID string, record *entities.ClusterExtension) error {
		return fmt.Errorf("uninstall: Release not loaded: %s: %w", record.ReleaseName, driver.ErrReleaseNotFound)
	}
	t.Cleanup(func() {
		executeClusterExtensionUninstall = originalExecuteUninstall
	})

	defaultLaunchClusterExtensionUninstall("cluster-1", record)

	require.Eventually(t, func() bool {
		var count int64
		require.NoError(t, db.DB.Model(&entities.ClusterExtension{}).Where("id = ?", "ce-1").Count(&count).Error)
		return count == 0
	}, time.Second, 10*time.Millisecond)
}

func TestRetryClusterExtensionAllowsVersionOverrideForFailedInstall(t *testing.T) {
	setupExtensionServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Extension{
		ID:     "ext-1",
		Slug:   "kube-prometheus-stack",
		Name:   "Kube Prometheus Stack",
		OCIUrl: "oci://example.com/kube-prometheus-stack",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ClusterExtension{
		ID:          "ce-1",
		ClusterID:   "cluster-1",
		ExtensionID: "ext-1",
		Namespace:   "ketches-extensions",
		ReleaseName: "kube-prometheus-stack",
		Version:     "9.4.9",
		Status:      entities.ClusterExtensionStatusFailed,
		Phase:       "installing",
	}).Error)

	originalLaunch := launchClusterExtensionInstall
	called := false
	launchClusterExtensionInstall = func(clusterID string, ext *entities.Extension, record *entities.ClusterExtension) {
		called = true
		assert.Equal(t, "69.8.2", record.Version)
	}
	t.Cleanup(func() {
		launchClusterExtensionInstall = originalLaunch
	})

	version := "69.8.2"
	result, err := RetryClusterExtension("cluster-1", "ce-1", &models.RetryClusterExtensionRequest{Version: &version})
	require.NoError(t, err)
	assert.True(t, called)
	require.NotNil(t, result)
	assert.Equal(t, "69.8.2", result.Version)

	var stored entities.ClusterExtension
	require.NoError(t, db.DB.First(&stored, "id = ?", "ce-1").Error)
	assert.Equal(t, "69.8.2", stored.Version)
	assert.Equal(t, entities.ClusterExtensionStatusPending, stored.Status)
}

func setupExtensionMetadataTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() { db.DB = originalDB })

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.Extension{}, &entities.Cluster{}, &entities.ClusterGatewayProvider{}))
	db.DB = testDB
}

func TestToExtensionModelIncludesCapabilitiesAndMetadata(t *testing.T) {
	ext := &entities.Extension{
		ID:           "ext-1",
		Slug:         "envoyGateway",
		Name:         "Envoy Gateway",
		Capabilities: `["gateway-api","observability"]`,
		Metadata:     entities.JSONBlob(`{"gateway_api":{"controller_name":"gateway.envoyproxy.io/gatewayclass-controller"}}`),
	}

	model := toExtensionModel(ext)
	assert.Equal(t, []string{"gateway-api", "observability"}, model.Capabilities)
	gatewayMetadata, ok := model.Metadata["gateway_api"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "gateway.envoyproxy.io/gatewayclass-controller", gatewayMetadata["controller_name"])
}

func TestReconcileClusterExtensionInstallSuccessSetsManagedDefaultGatewayClassWhenMissing(t *testing.T) {
	setupExtensionMetadataTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "cluster-one",
		Name:       "Cluster One",
		KubeConfig: "enc:v1:test",
		Enabled:    true,
	}).Error)

	ext := &entities.Extension{
		ID:           "ext-1",
		Slug:         "envoyGateway",
		Name:         "Envoy Gateway",
		Capabilities: `["gateway-api"]`,
		Metadata:     entities.JSONBlob(`{"gateway_api":{"controller_name":"gateway.envoyproxy.io/gatewayclass-controller"}}`),
	}
	record := &entities.ClusterExtension{ClusterID: "cluster-1", ReleaseName: "envoy-gateway"}

	originalEnsureGatewayClass := ensureGatewayClassForExtensionInstall
	originalEnsureSharedGateway := ensureSharedGatewayForExtensionInstall
	defer func() {
		ensureGatewayClassForExtensionInstall = originalEnsureGatewayClass
		ensureSharedGatewayForExtensionInstall = originalEnsureSharedGateway
	}()

	calledGatewayClass := false
	calledSharedGateway := false
	ensureGatewayClassForExtensionInstall = func(clusterID, name, controllerName string) error {
		calledGatewayClass = true
		assert.Equal(t, "cluster-1", clusterID)
		assert.Equal(t, "ketches-envoy-gateway", name)
		assert.Equal(t, "gateway.envoyproxy.io/gatewayclass-controller", controllerName)
		return nil
	}
	ensureSharedGatewayForExtensionInstall = func(clusterID string) error {
		calledSharedGateway = true
		assert.Equal(t, "cluster-1", clusterID)
		return nil
	}

	require.NoError(t, reconcileClusterExtensionInstallSuccess("cluster-1", ext, record))
	assert.True(t, calledGatewayClass)
	assert.True(t, calledSharedGateway)

	provider, err := GetDefaultClusterGatewayProvider("cluster-1")
	require.NoError(t, err)
	require.NotNil(t, provider)
	assert.Equal(t, "ketches-envoy-gateway", provider.GatewayClassName)
	assert.Equal(t, "gateway.envoyproxy.io/gatewayclass-controller", provider.ControllerName)
	assert.Equal(t, "managed", provider.SourceType)
}

func TestReconcileClusterExtensionInstallSuccessSkipsNonGatewayExtensions(t *testing.T) {
	setupExtensionMetadataTestDB(t)

	ext := &entities.Extension{ID: "ext-1", Slug: "cert-manager", Name: "Cert Manager", Capabilities: `["observability"]`}
	record := &entities.ClusterExtension{ClusterID: "cluster-1", ReleaseName: "cert-manager"}

	originalEnsureGatewayClass := ensureGatewayClassForExtensionInstall
	originalEnsureSharedGateway := ensureSharedGatewayForExtensionInstall
	defer func() {
		ensureGatewayClassForExtensionInstall = originalEnsureGatewayClass
		ensureSharedGatewayForExtensionInstall = originalEnsureSharedGateway
	}()

	ensureGatewayClassForExtensionInstall = func(clusterID, name, controllerName string) error {
		t.Fatal("should not create gateway class for non-gateway extension")
		return nil
	}
	ensureSharedGatewayForExtensionInstall = func(clusterID string) error {
		t.Fatal("should not create shared gateway for non-gateway extension")
		return nil
	}

	require.NoError(t, reconcileClusterExtensionInstallSuccess("cluster-1", ext, record))
}

func TestSortExtensionVersionsPrefersNewestStableSemver(t *testing.T) {
	versions := sortExtensionVersions([]string{"9.4.9", "68.4.4", "70.0.0", "10.0.0"})

	assert.Equal(t, []string{"70.0.0", "68.4.4", "10.0.0", "9.4.9"}, versions)
}

func TestSortExtensionVersionsPlacesPrereleasesAfterStableVersions(t *testing.T) {
	versions := sortExtensionVersions([]string{"70.0.0-rc.1", "69.8.2", "69.8.1", "latest", "sha256-deadbeef"})

	assert.Equal(t, []string{"69.8.2", "69.8.1", "70.0.0-rc.1", "sha256-deadbeef", "latest"}, versions)
}
