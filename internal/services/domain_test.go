package services

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupDomainTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, testDB.AutoMigrate(&entities.Cluster{}, &entities.Project{}, &entities.Env{}, &entities.Domain{}))
	db.DB = testDB
}

func TestCreateClusterDomain(t *testing.T) {
	setupDomainTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base: entities.Base{ID: "cluster-1"},
		Slug: "cluster-1",
		Name: "Cluster 1",
	}).Error)

	item, err := CreateClusterDomain("cluster-1", &models.CreateDomainRequest{
		Name:        "Primary",
		Domain:      "*.Example.COM",
		Description: "primary domain",
	})
	require.NoError(t, err)
	assert.Equal(t, "Primary", item.Name)
	assert.Equal(t, "*.example.com", item.Domain)
	assert.Equal(t, "cluster", item.Scope)
	assert.Equal(t, "cluster-1", item.ClusterID)
}

func TestCreateEnvDomain(t *testing.T) {
	setupDomainTestDB(t)

	require.NoError(t, db.DB.Create(&entities.Project{
		Base: entities.Base{ID: "project-1"},
		Slug: "project-1",
		Name: "Project 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Cluster{
		Base: entities.Base{ID: "cluster-1"},
		Slug: "cluster-1",
		Name: "Cluster 1",
	}).Error)
	require.NoError(t, db.DB.Create(&entities.Env{
		Base:      entities.Base{ID: "env-1"},
		Slug:      "env-1",
		Name:      "Env 1",
		ProjectID: "project-1",
		ClusterID: "cluster-1",
	}).Error)

	item, err := CreateEnvDomain("env-1", &models.CreateDomainRequest{
		Name:   "Env Primary",
		Domain: "*.env.example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, item.EnvID)
	assert.Equal(t, "env", item.Scope)
	assert.Equal(t, "env-1", *item.EnvID)
	assert.Equal(t, "cluster-1", item.ClusterID)
}

func TestEnvDomainOperationsRequireMatchingParent(t *testing.T) {
	setupDomainTestDB(t)

	for _, env := range []entities.Env{
		{Base: entities.Base{ID: "env-1"}, Slug: "env-1", Name: "Env 1", ProjectID: "project-1", ClusterID: "cluster-1"},
		{Base: entities.Base{ID: "env-2"}, Slug: "env-2", Name: "Env 2", ProjectID: "project-2", ClusterID: "cluster-2"},
	} {
		require.NoError(t, db.DB.Create(&env).Error)
	}

	domainEnvID := "env-2"
	domain := &entities.Domain{
		Base:      entities.Base{ID: "domain-2"},
		Name:      "Other Environment",
		Domain:    "other.example.com",
		Scope:     "env",
		ClusterID: "cluster-2",
		EnvID:     &domainEnvID,
	}
	require.NoError(t, db.DB.Create(domain).Error)

	_, err := GetDomain("env-1", "domain-2")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	name := "Tampered"
	_, err = UpdateDomain("env-1", "domain-2", &models.UpdateDomainRequest{Name: &name})
	assert.Error(t, err)

	require.Error(t, DeleteDomain("env-1", "domain-2"))
	var stored entities.Domain
	require.NoError(t, db.DB.First(&stored, "id = ?", "domain-2").Error)
	assert.Equal(t, "Other Environment", stored.Name)
}
