package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAppGatewayServiceTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	originalSync := syncGatewaysToK8s
	originalReadNodePorts := readNodePortsFromK8s
	originalDeleteGateway := deleteGatewayFromK8s
	originalEnsureSharedGateway := ensureSharedGatewayAfterGatewayDelete
	t.Cleanup(func() {
		db.DB = originalDB
		syncGatewaysToK8s = originalSync
		readNodePortsFromK8s = originalReadNodePorts
		deleteGatewayFromK8s = originalDeleteGateway
		ensureSharedGatewayAfterGatewayDelete = originalEnsureSharedGateway
	})

	testDB, err := gorm.Open(sqlite.Open(t.TempDir()+"/app-gateway-test.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(
		&entities.Project{},
		&entities.Cluster{},
		&entities.Env{},
		&entities.App{},
		&entities.Certificate{},
		&entities.AppGateway{},
		&entities.AppGatewayHTTPRoute{},
		&entities.AppGatewayHTTPRouteBackend{},
		&entities.AppEnvVar{},
		&entities.AppVolume{},
		&entities.AppConfigFile{},
		&entities.AppProbe{},
		&entities.AppAutoScaling{},
		&entities.AppSchedulingRule{},
		&entities.AppPlugin{},
		&entities.Plugin{},
	))
	db.DB = testDB

	syncGatewaysToK8s = func(context.Context, *models.AppContext) error {
		return nil
	}
	readNodePortsFromK8s = func(context.Context, *models.AppContext) (map[int]int, error) {
		return nil, nil
	}
	deleteGatewayFromK8s = func(context.Context, *models.AppContext, *entities.AppGateway) error {
		return nil
	}
	ensureSharedGatewayAfterGatewayDelete = func(context.Context, string) error {
		return nil
	}
}

func seedAppGatewayServiceApp(t *testing.T) {
	t.Helper()

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "demo-project",
		Name: "Demo Project",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:        entities.Base{ID: "cluster-1"},
		Slug:        "demo-cluster",
		Name:        "Demo Cluster",
		KubeConfig:  "enc:v1:test",
		GatewayHost: "gw.example.com",
		Enabled:     true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:             entities.Base{ID: "env-1"},
		Slug:             "prod",
		Name:             "Production",
		ProjectID:        "project-1",
		ClusterID:        "cluster-1",
		ClusterNamespace: "prod",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.App{
		Base:           entities.Base{ID: "app-1"},
		Slug:           "api",
		Name:           "API",
		EnvID:          "env-1",
		AppType:        "Deployment",
		ContainerImage: "nginx:latest",
		Replicas:       1,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Certificate{
		ID:        "cert-1",
		ClusterID: "cluster-1",
		Scope:     "cluster",
		Name:      "Primary",
		Cert:      "cert",
		Key:       "key",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Certificate{
		ID:        "cert-2",
		ClusterID: "cluster-1",
		Scope:     "cluster",
		Name:      "Secondary",
		Cert:      "cert",
		Key:       "key",
	}).Error)
}

func TestCreateAppGatewayCreatesRoutesAndDefaultBackends(t *testing.T) {
	setupAppGatewayServiceTestDB(t)
	seedAppGatewayServiceApp(t)

	response, err := CreateAppGateway(context.Background(), "app-1", &models.CreateGatewayRequest{
		Port:        8080,
		Protocol:    "http",
		ServiceType: "ClusterIP",
		Routes: []models.GatewayRouteSpec{
			{
				Host:             "api.example.com",
				ListenerProtocol: "http",
				Path:             "/",
				PathMatchType:    "PathPrefix",
				Enabled:          true,
			},
			{
				Host:             "secure.example.com",
				ListenerProtocol: "https",
				Path:             "/secure",
				PathMatchType:    "PathPrefix",
				Enabled:          true,
				CertID:           "cert-1",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "gw.example.com", response.GatewayHost)
	assert.Equal(t, "api.prod:8080", response.InternalAddress)
	require.Len(t, response.Routes, 2)

	var routes []entities.AppGatewayHTTPRoute
	require.NoError(t, db.DB.Order("host").Find(&routes).Error)
	require.Len(t, routes, 2)
	assert.Equal(t, "api.example.com", routes[0].Host)
	assert.Equal(t, "secure.example.com", routes[1].Host)

	var backends []entities.AppGatewayHTTPRouteBackend
	require.NoError(t, db.DB.Order("backend_port").Find(&backends).Error)
	require.Len(t, backends, 2)
	for _, backend := range backends {
		assert.Equal(t, "app-1", backend.BackendAppID)
		assert.Equal(t, 8080, backend.BackendPort)
		assert.Equal(t, 1, backend.Weight)
	}
}

func TestCreateAppGatewayRequiresHTTPSRouteCertificate(t *testing.T) {
	setupAppGatewayServiceTestDB(t)
	seedAppGatewayServiceApp(t)

	_, err := CreateAppGateway(context.Background(), "app-1", &models.CreateGatewayRequest{
		Port:        8080,
		Protocol:    "http",
		ServiceType: "ClusterIP",
		Routes: []models.GatewayRouteSpec{
			{Host: "secure.example.com", ListenerProtocol: "https", Path: "/", Enabled: true},
		},
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidGatewayCertificate))
}

func TestCreateAppGatewayRejectsRouteWithTooManyBackends(t *testing.T) {
	setupAppGatewayServiceTestDB(t)
	seedAppGatewayServiceApp(t)

	backends := make([]models.GatewayRouteBackendSpec, 17)
	for i := range backends {
		backends[i] = models.GatewayRouteBackendSpec{BackendAppID: "app-1", BackendPort: 8080, Weight: 1}
	}

	_, err := CreateAppGateway(context.Background(), "app-1", &models.CreateGatewayRequest{
		Port:        8080,
		Protocol:    "http",
		ServiceType: "ClusterIP",
		Routes: []models.GatewayRouteSpec{
			{Host: "api.example.com", ListenerProtocol: "http", Path: "/", Enabled: true, Backends: backends},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no more than 16 backends")
}

func TestCreateAppGatewayRejectsBackendTimeoutGreaterThanRequestTimeout(t *testing.T) {
	setupAppGatewayServiceTestDB(t)
	seedAppGatewayServiceApp(t)

	_, err := CreateAppGateway(context.Background(), "app-1", &models.CreateGatewayRequest{
		Port:        8080,
		Protocol:    "http",
		ServiceType: "ClusterIP",
		Routes: []models.GatewayRouteSpec{
			{
				Host:             "api.example.com",
				ListenerProtocol: "http",
				Path:             "/",
				Enabled:          true,
				Timeouts:         &models.GatewayRouteTimeouts{Request: "5s", BackendRequest: "10s"},
			},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "backend_request timeout must not exceed request timeout")
}

func TestCreateAppGatewayRejectsSameHTTPSHostWithDifferentCertificate(t *testing.T) {
	setupAppGatewayServiceTestDB(t)
	seedAppGatewayServiceApp(t)

	require.NoError(t, db.DB.Create(&entities.AppGateway{
		ID:          "gateway-existing",
		AppID:       "app-1",
		Port:        8080,
		Protocol:    "http",
		ServiceType: "ClusterIP",
	}).Error)
	certID := "cert-1"
	require.NoError(t, db.DB.Create(&entities.AppGatewayHTTPRoute{
		ID:               "route-existing",
		AppGatewayID:     "gateway-existing",
		Host:             "secure.example.com",
		ListenerProtocol: "https",
		Path:             "/",
		PathMatchType:    "PathPrefix",
		Enabled:          true,
		CertID:           &certID,
	}).Error)

	_, err := CreateAppGateway(context.Background(), "app-1", &models.CreateGatewayRequest{
		Port:        9090,
		Protocol:    "http",
		ServiceType: "ClusterIP",
		Routes: []models.GatewayRouteSpec{
			{
				Host:             "secure.example.com",
				ListenerProtocol: "https",
				Path:             "/v2",
				Enabled:          true,
				CertID:           "cert-2",
			},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "different certificate")
}

func TestUpdateAppGatewayReplacesRoutesAndBackends(t *testing.T) {
	setupAppGatewayServiceTestDB(t)
	seedAppGatewayServiceApp(t)

	require.NoError(t, db.DB.Create(&entities.AppGateway{
		ID:          "gateway-1",
		AppID:       "app-1",
		Port:        8080,
		Protocol:    "http",
		ServiceType: "ClusterIP",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppGatewayHTTPRoute{
		ID:               "route-old",
		AppGatewayID:     "gateway-1",
		Host:             "old.example.com",
		ListenerProtocol: "http",
		Path:             "/",
		PathMatchType:    "PathPrefix",
		Enabled:          true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppGatewayHTTPRouteBackend{
		ID:           "backend-old",
		RouteID:      "route-old",
		BackendAppID: "app-1",
		BackendPort:  8080,
		Weight:       1,
	}).Error)

	response, err := UpdateAppGateway(context.Background(), "gateway-1", &models.UpdateGatewayRequest{
		Port:        8080,
		Protocol:    "http",
		ServiceType: "ClusterIP",
		Routes: []models.GatewayRouteSpec{
			{Host: "new.example.com", ListenerProtocol: "http", Path: "/new", Enabled: true},
		},
	})

	require.NoError(t, err)
	require.Len(t, response.Routes, 1)
	assert.Equal(t, "new.example.com", response.Routes[0].Host)

	var routes []entities.AppGatewayHTTPRoute
	require.NoError(t, db.DB.Find(&routes).Error)
	require.Len(t, routes, 1)
	assert.Equal(t, "new.example.com", routes[0].Host)

	var backends []entities.AppGatewayHTTPRouteBackend
	require.NoError(t, db.DB.Find(&backends).Error)
	require.Len(t, backends, 1)
	assert.Equal(t, routes[0].ID, backends[0].RouteID)
}

func TestDeleteAppGatewayUsesAppFenceAndReconcilesSharedGateway(t *testing.T) {
	setupAppGatewayServiceTestDB(t)
	seedAppGatewayServiceApp(t)

	require.NoError(t, db.DB.Create(&entities.AppGateway{
		ID:          "gateway-delete",
		AppID:       "app-1",
		Port:        8080,
		Protocol:    "http",
		ServiceType: "ClusterIP",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppGatewayHTTPRoute{
		ID:               "route-delete",
		AppGatewayID:     "gateway-delete",
		Host:             "delete.example.com",
		ListenerProtocol: "https",
		Path:             "/",
		PathMatchType:    "PathPrefix",
		Enabled:          true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.AppGatewayHTTPRouteBackend{
		ID:           "backend-delete",
		RouteID:      "route-delete",
		BackendAppID: "app-1",
		BackendPort:  8080,
		Weight:       1,
	}).Error)

	fenceHeld := make(chan struct{})
	releaseFence := make(chan struct{})
	fenceResult := make(chan error, 1)
	go func() {
		fenceResult <- core.WithAppReconcileFence(context.Background(), "app-1", func() error {
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

	kubernetesDeleteCalled := make(chan struct{})
	deleteGatewayFromK8s = func(context.Context, *models.AppContext, *entities.AppGateway) error {
		close(kubernetesDeleteCalled)
		return nil
	}
	var reconciledClusterID string
	var sharedGatewayQueryErr error
	var sharedGatewayObservedCount int64
	ensureSharedGatewayAfterGatewayDelete = func(_ context.Context, clusterID string) error {
		reconciledClusterID = clusterID
		sharedGatewayQueryErr = db.DB.Model(&entities.AppGateway{}).
			Where("id = ?", "gateway-delete").
			Count(&sharedGatewayObservedCount).Error
		return sharedGatewayQueryErr
	}

	deleteResult := make(chan error, 1)
	go func() {
		deleteResult <- DeleteAppGateway(context.Background(), "gateway-delete")
	}()
	select {
	case <-kubernetesDeleteCalled:
		t.Fatal("gateway deletion entered Kubernetes while the app fence was held")
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseFence)
	require.NoError(t, <-fenceResult)
	require.NoError(t, <-deleteResult)
	assert.Equal(t, "cluster-1", reconciledClusterID)
	require.NoError(t, sharedGatewayQueryErr)
	require.Zero(t, sharedGatewayObservedCount, "shared Gateway must be reconciled from committed desired state")

	for _, model := range []any{
		&entities.AppGateway{},
		&entities.AppGatewayHTTPRoute{},
		&entities.AppGatewayHTTPRouteBackend{},
	} {
		var count int64
		require.NoError(t, db.DB.Model(model).Count(&count).Error)
		require.Zero(t, count)
	}
}

func TestCertificateInUseChecksGatewayRoutes(t *testing.T) {
	setupAppGatewayServiceTestDB(t)
	seedAppGatewayServiceApp(t)

	require.NoError(t, db.DB.Create(&entities.AppGateway{
		ID:          "gateway-1",
		AppID:       "app-1",
		Port:        8080,
		Protocol:    "http",
		ServiceType: "ClusterIP",
	}).Error)
	certID := "cert-1"
	require.NoError(t, db.DB.Create(&entities.AppGatewayHTTPRoute{
		ID:               "route-1",
		AppGatewayID:     "gateway-1",
		Host:             "secure.example.com",
		ListenerProtocol: "https",
		Path:             "/",
		PathMatchType:    "PathPrefix",
		Enabled:          true,
		CertID:           &certID,
	}).Error)

	inUse, err := certificateInUse("cert-1")

	require.NoError(t, err)
	assert.True(t, inUse)
}
