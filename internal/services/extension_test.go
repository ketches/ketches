package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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
		ID:          "ext-1",
		Name:        "envoy-gateway",
		DisplayName: "Envoy Gateway",
		OCIUrl:      "oci://example.com/envoy-gateway",
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
		ID:          "ext-1",
		Name:        "envoy-gateway",
		DisplayName: "Envoy Gateway",
		OCIUrl:      "oci://example.com/envoy-gateway",
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

func TestInstallClusterExtensionNormalizesReleaseNameAndStoresDisplayName(t *testing.T) {
	setupExtensionServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Extension{
		ID:          "ext-1",
		Name:        "envoy-gateway",
		DisplayName: "Envoy Gateway",
		OCIUrl:      "oci://example.com/envoy-gateway",
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
		ID:          "ext-1",
		Name:        "envoy-gateway",
		DisplayName: "Envoy Gateway",
		OCIUrl:      "oci://example.com/envoy-gateway",
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
		ID:          "ext-1",
		Name:        "envoy-gateway",
		DisplayName: "Envoy Gateway",
		OCIUrl:      "oci://example.com/envoy-gateway",
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

func TestRetryClusterExtensionNormalizesLegacyInvalidReleaseName(t *testing.T) {
	setupExtensionServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Extension{
		ID:          "ext-1",
		Name:        "envoy-gateway",
		DisplayName: "Envoy Gateway",
		OCIUrl:      "oci://example.com/envoy-gateway",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ClusterExtension{
		ID:           "ce-legacy",
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

	result, err := RetryClusterExtension("cluster-1", "ce-legacy", nil)
	require.NoError(t, err)
	assert.True(t, called)
	require.NotNil(t, result)
	assert.Equal(t, "envoy-gateway", result.ReleaseName)
	assert.Equal(t, "Envoy Gateway", result.Name)

	var stored entities.ClusterExtension
	require.NoError(t, db.DB.First(&stored, "id = ?", "ce-legacy").Error)
	assert.Equal(t, "envoy-gateway", stored.ReleaseName)
	assert.Equal(t, "Envoy Gateway", stored.Name)
}

func TestRetryClusterExtensionQueuesInstallRetryForFailedInstall(t *testing.T) {
	setupExtensionServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Extension{
		ID:          "ext-1",
		Name:        "envoy-gateway",
		DisplayName: "Envoy Gateway",
		OCIUrl:      "oci://example.com/envoy-gateway",
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
		Name:         "envoyGateway",
		DisplayName:  "Envoy Gateway",
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
		Name:         "envoyGateway",
		DisplayName:  "Envoy Gateway",
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

	ext := &entities.Extension{ID: "ext-1", Name: "metrics-server", Capabilities: `["observability"]`}
	record := &entities.ClusterExtension{ClusterID: "cluster-1", ReleaseName: "metrics-server"}

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
