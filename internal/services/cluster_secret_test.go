package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupClusterSecretTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.Cluster{}, &entities.ClusterIntegration{}))

	db.DB = testDB
}

func testClusterKubeConfig(serverURL string) string {
	return `apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: ` + serverURL + `
users:
- name: test
  user:
    token: test-token
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
`
}

func TestCreateClusterEncryptsKubeConfig(t *testing.T) {
	setupClusterSecretTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	req := &models.CreateClusterRequest{
		Slug:        "demo-cluster",
		Name:        "Demo Cluster",
		KubeConfig:  testClusterKubeConfig("https://127.0.0.1"),
		GatewayHost: "gateway.example.com",
	}

	cluster, err := CreateCluster(req)
	require.NoError(t, err)
	t.Cleanup(func() {
		kube.GlobalClusterStore.RemoveClient(cluster.ID)
	})

	assert.NotEqual(t, req.KubeConfig, cluster.KubeConfig)

	var stored entities.Cluster
	require.NoError(t, db.DB.First(&stored, "id = ?", cluster.ID).Error)
	assert.NotEqual(t, req.KubeConfig, stored.KubeConfig)
	assert.Equal(t, "gateway.example.com", stored.GatewayHost)

	_, err = kube.GlobalClusterStore.GetClient(cluster.ID)
	require.NoError(t, err)
}

func TestUpdateClusterCredentialsPersistsGatewayHost(t *testing.T) {
	setupClusterSecretTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	encryptedKubeConfig, err := secrets.EncryptString(testClusterKubeConfig("https://127.0.0.1"))
	require.NoError(t, err)

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base:        entities.Base{ID: "cluster-credentials"},
		Slug:        "cluster-credentials",
		Name:        "Cluster Credentials",
		KubeConfig:  encryptedKubeConfig,
		GatewayHost: "old-gateway.example.com",
		Enabled:     false,
	}).Error)

	updated, err := UpdateClusterCredentials("cluster-credentials", &models.UpdateClusterCredentialsRequest{
		GatewayHost: "new-gateway.example.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "new-gateway.example.com", updated.GatewayHost)

	var stored entities.Cluster
	require.NoError(t, db.DB.First(&stored, "id = ?", "cluster-credentials").Error)
	assert.Equal(t, "new-gateway.example.com", stored.GatewayHost)
}

func TestInitClustersLoadsEncryptedKubeConfig(t *testing.T) {
	setupClusterSecretTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	encryptedKubeConfig, err := secrets.EncryptString(testClusterKubeConfig("https://127.0.0.1"))
	require.NoError(t, err)

	cluster := entities.Cluster{
		Base:       entities.Base{ID: "cluster-1"},
		Slug:       "demo-cluster",
		Name:       "Demo Cluster",
		KubeConfig: encryptedKubeConfig,
		Enabled:    true,
	}
	require.NoError(t, db.DB.Create(&cluster).Error)
	t.Cleanup(func() {
		kube.GlobalClusterStore.RemoveClient(cluster.ID)
	})

	require.NoError(t, InitClusters())

	_, err = kube.GlobalClusterStore.GetClient(cluster.ID)
	require.NoError(t, err)
}

func TestInitClustersMigratesLegacyPlaintextKubeConfig(t *testing.T) {
	setupClusterSecretTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	plaintextKubeConfig := testClusterKubeConfig("https://127.0.0.1")
	cluster := entities.Cluster{
		Base:       entities.Base{ID: "legacy-plaintext-cluster"},
		Slug:       "legacy-plaintext-cluster",
		Name:       "Legacy Plaintext Cluster",
		KubeConfig: plaintextKubeConfig,
		Enabled:    true,
	}
	require.NoError(t, db.DB.Create(&cluster).Error)
	t.Cleanup(func() {
		kube.GlobalClusterStore.RemoveClient(cluster.ID)
	})

	require.NoError(t, InitClusters())

	_, err := kube.GlobalClusterStore.GetClient(cluster.ID)
	require.NoError(t, err)

	var stored entities.Cluster
	require.NoError(t, db.DB.First(&stored, "id = ?", cluster.ID).Error)
	assert.True(t, secrets.IsEncrypted(stored.KubeConfig))
	decrypted, err := secrets.DecryptString(stored.KubeConfig)
	require.NoError(t, err)
	assert.Equal(t, plaintextKubeConfig, decrypted)
}

func TestInitClustersSkipsClustersWithUndecryptableKubeConfig(t *testing.T) {
	setupClusterSecretTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "current-master-key"

	validKubeConfig := testClusterKubeConfig("https://127.0.0.1")
	validEncryptedKubeConfig, err := secrets.EncryptString(validKubeConfig)
	require.NoError(t, err)

	app.Config.SecretEncryptionKey = "stale-master-key"
	staleEncryptedKubeConfig, err := secrets.EncryptString(testClusterKubeConfig("https://10.0.0.1"))
	require.NoError(t, err)
	app.Config.SecretEncryptionKey = "current-master-key"

	goodCluster := entities.Cluster{
		Base:       entities.Base{ID: "cluster-good"},
		Slug:       "good-cluster",
		Name:       "Good Cluster",
		KubeConfig: validEncryptedKubeConfig,
		Enabled:    true,
	}
	badCluster := entities.Cluster{
		Base:       entities.Base{ID: "cluster-bad"},
		Slug:       "bad-cluster",
		Name:       "Bad Cluster",
		KubeConfig: staleEncryptedKubeConfig,
		Enabled:    true,
	}

	require.NoError(t, db.DB.Create(&goodCluster).Error)
	require.NoError(t, db.DB.Create(&badCluster).Error)
	t.Cleanup(func() {
		kube.GlobalClusterStore.RemoveClient(goodCluster.ID)
		kube.GlobalClusterStore.RemoveClient(badCluster.ID)
	})

	require.NoError(t, InitClusters())

	_, err = kube.GlobalClusterStore.GetClient(goodCluster.ID)
	require.NoError(t, err)

	_, err = kube.GlobalClusterStore.GetClient(badCluster.ID)
	require.Error(t, err)

	var storedBad entities.Cluster
	require.NoError(t, db.DB.First(&storedBad, "id = ?", badCluster.ID).Error)
	assert.Equal(t, "disconnected", storedBad.ConnectionStatus)
	assert.Contains(t, storedBad.ConnectionStatusReason, "decrypt ciphertext")
	assert.NotNil(t, storedBad.LastCheckedAt)
}

func TestCreateClusterIntegrationEncryptsSecretsAtRest(t *testing.T) {
	setupClusterSecretTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	integration, err := CreateClusterIntegration("cluster-1", &models.CreateClusterIntegrationRequest{
		IntegrationType: "prometheus",
		Name:            "Prometheus",
		Endpoint:        "https://prometheus.example.com",
		Username:        "demo",
		Password:        "super-secret-password",
		Token:           "super-secret-token",
		CACert:          "super-secret-ca-cert",
		Enabled:         true,
	})
	require.NoError(t, err)

	assert.NotEqual(t, "super-secret-password", integration.Password)
	assert.NotEqual(t, "super-secret-token", integration.Token)
	assert.NotEqual(t, "super-secret-ca-cert", integration.CACert)
	assert.Contains(t, integration.Password, "enc:v1:")
	assert.Contains(t, integration.Token, "enc:v1:")
	assert.Contains(t, integration.CACert, "enc:v1:")

	var stored entities.ClusterIntegration
	require.NoError(t, db.DB.First(&stored, "id = ?", integration.ID).Error)
	assert.Contains(t, stored.Password, "enc:v1:")
	assert.Contains(t, stored.Token, "enc:v1:")
	assert.Contains(t, stored.CACert, "enc:v1:")

	decryptedPassword, err := secrets.DecryptString(stored.Password)
	require.NoError(t, err)
	assert.Equal(t, "super-secret-password", decryptedPassword)
	decryptedToken, err := secrets.DecryptString(stored.Token)
	require.NoError(t, err)
	assert.Equal(t, "super-secret-token", decryptedToken)
	decryptedCACert, err := secrets.DecryptString(stored.CACert)
	require.NoError(t, err)
	assert.Equal(t, "super-secret-ca-cert", decryptedCACert)
}

func TestUpdateClusterIntegrationEncryptsSecretsAtRest(t *testing.T) {
	setupClusterSecretTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	require.NoError(t, db.DB.Create(&entities.ClusterIntegration{
		ID:              "integration-1",
		ClusterID:       "cluster-1",
		IntegrationType: entities.IntegrationTypePrometheus,
		Name:            "Prometheus",
		Endpoint:        "https://prometheus.example.com",
		Enabled:         true,
	}).Error)

	password := "new-super-secret-password"
	token := "new-super-secret-token"
	caCert := "new-super-secret-ca-cert"
	updated, err := UpdateClusterIntegration("integration-1", &models.UpdateClusterIntegrationRequest{
		Password: &password,
		Token:    &token,
		CACert:   &caCert,
	})
	require.NoError(t, err)

	assert.NotEqual(t, password, updated.Password)
	assert.NotEqual(t, token, updated.Token)
	assert.NotEqual(t, caCert, updated.CACert)
	assert.Contains(t, updated.Password, "enc:v1:")
	assert.Contains(t, updated.Token, "enc:v1:")
	assert.Contains(t, updated.CACert, "enc:v1:")

	var stored entities.ClusterIntegration
	require.NoError(t, db.DB.First(&stored, "id = ?", updated.ID).Error)
	assert.Contains(t, stored.Password, "enc:v1:")
	assert.Contains(t, stored.Token, "enc:v1:")
	assert.Contains(t, stored.CACert, "enc:v1:")
}

func TestUpdateClusterIntegrationClearsSecretsAtRest(t *testing.T) {
	setupClusterSecretTestDB(t)

	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "test-master-key"

	encryptedPassword, err := secrets.EncryptString("secret-password")
	require.NoError(t, err)
	encryptedToken, err := secrets.EncryptString("secret-token")
	require.NoError(t, err)
	encryptedCACert, err := secrets.EncryptString("secret-ca")
	require.NoError(t, err)

	require.NoError(t, db.DB.Create(&entities.ClusterIntegration{
		ID:              "integration-2",
		ClusterID:       "cluster-1",
		IntegrationType: entities.IntegrationTypePrometheus,
		Name:            "Prometheus",
		Endpoint:        "https://prometheus.example.com",
		Password:        encryptedPassword,
		Token:           encryptedToken,
		CACert:          encryptedCACert,
		Enabled:         true,
	}).Error)

	clearPassword := true
	clearToken := true
	clearCACert := true
	updated, err := UpdateClusterIntegration("integration-2", &models.UpdateClusterIntegrationRequest{
		ClearPassword: &clearPassword,
		ClearToken:    &clearToken,
		ClearCACert:   &clearCACert,
	})
	require.NoError(t, err)
	assert.Empty(t, updated.Password)
	assert.Empty(t, updated.Token)
	assert.Empty(t, updated.CACert)

	var stored entities.ClusterIntegration
	require.NoError(t, db.DB.First(&stored, "id = ?", updated.ID).Error)
	assert.Empty(t, stored.Password)
	assert.Empty(t, stored.Token)
	assert.Empty(t, stored.CACert)
}
