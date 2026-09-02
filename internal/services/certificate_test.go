package services

import (
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
)

func setupCertificateServiceTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	originalConfig := app.Config
	t.Cleanup(func() {
		db.DB = originalDB
		app.Config = originalConfig
	})
	app.Config.SecretEncryptionKey = "certificate-test-key"

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.Certificate{}, &entities.AppGatewayHTTPRoute{}))
	db.DB = testDB
}

func TestEnvCertificateOperationsRequireMatchingParent(t *testing.T) {
	setupCertificateServiceTestDB(t)

	envID := "env-2"
	require.NoError(t, db.DB.Create(&entities.Certificate{
		ID:        "cert-2",
		Name:      "Other Environment",
		Cert:      "certificate",
		Key:       "private-key",
		Scope:     "env",
		ClusterID: "cluster-2",
		EnvID:     &envID,
	}).Error)

	_, err := GetCertificate("env-1", "cert-2")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	name := "Tampered"
	_, err = UpdateCertificate("env-1", "cert-2", &models.UpdateCertificateRequest{Name: &name})
	assert.Error(t, err)

	require.Error(t, DeleteCertificate("env-1", "cert-2"))
	var stored entities.Certificate
	require.NoError(t, db.DB.First(&stored, "id = ?", "cert-2").Error)
	assert.Equal(t, "Other Environment", stored.Name)
}
